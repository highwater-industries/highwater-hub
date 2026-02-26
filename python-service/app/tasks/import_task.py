"""Celery task that runs the NFL data-collector pipeline.

This executes inside a Celery worker process.  Because the collector
uses ``async def``, we spin up a small asyncio event loop for the
duration of the task.

After collection, the task persists results to Postgres according to
the chosen strategy (merge / replace / append / dry_run) and records
the run in the ``collection_history`` table.
"""

import asyncio
import logging
from datetime import UTC, date, datetime
from typing import Any, Dict, List

from sqlalchemy import delete, select

from app.celery_app import celery_app
from app.data_collectors import CollectorFactory
from app.database import SyncSession, create_tables_sync
from app.models.models import CollectionHistory, PlayerDB

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


def _sanitize_for_json(obj: Any) -> Any:
    """Recursively convert non-JSON-serializable types (date, datetime, etc.)."""
    if isinstance(obj, dict):
        return {k: _sanitize_for_json(v) for k, v in obj.items()}
    if isinstance(obj, (list, tuple)):
        return [_sanitize_for_json(v) for v in obj]
    if isinstance(obj, datetime):
        return obj.isoformat()
    if isinstance(obj, date):
        return obj.isoformat()
    return obj


# ------------------------------------------------------------------
# Persistence helpers
# ------------------------------------------------------------------


def _persist_players(
    players: List[Dict[str, Any]],
    strategy: str,
    collector_type: str,
) -> Dict[str, int]:
    """Write collected player dicts to the ``players`` table.

    Returns a counters dict: {inserted, updated, skipped}.
    """
    inserted = updated = skipped = 0

    with SyncSession() as session:
        if strategy == "replace":
            # Wipe all rows from this source before inserting fresh data
            session.execute(
                delete(PlayerDB).where(PlayerDB.source == collector_type)
            )
            session.flush()

        for p in players:
            pid = p.get("player_id")

            if strategy == "dry_run":
                skipped += 1
                continue

            if strategy == "append":
                # Always insert — no dedup
                session.add(PlayerDB(
                    player_id=pid,
                    player_name=p["player_name"],
                    team=p.get("team"),
                    player_position=p.get("player_position"),
                    source=p.get("source"),
                    metadata_=_sanitize_for_json(p.get("metadata")),
                ))
                inserted += 1
                continue

            # strategy == "merge" or "replace" (after wipe, replace is
            # the same as append, but we still dedup on player_id)
            existing = None
            if pid:
                existing = session.execute(
                    select(PlayerDB).where(PlayerDB.player_id == pid)
                ).scalar_one_or_none()

            if existing:
                # Update mutable fields
                existing.player_name = p["player_name"]
                existing.team = p.get("team")
                existing.player_position = p.get("player_position")
                existing.source = p.get("source")
                existing.metadata_ = _sanitize_for_json(p.get("metadata"))
                existing.updated_at = datetime.now(UTC)
                updated += 1
            else:
                session.add(PlayerDB(
                    player_id=pid,
                    player_name=p["player_name"],
                    team=p.get("team"),
                    player_position=p.get("player_position"),
                    source=p.get("source"),
                    metadata_=_sanitize_for_json(p.get("metadata")),
                ))
                inserted += 1

        session.commit()

    return {"inserted": inserted, "updated": updated, "skipped": skipped}


def _record_history(
    collector_type: str,
    status: str,
    records_fetched: int = 0,
    records_inserted: int = 0,
    records_updated: int = 0,
    records_skipped: int = 0,
    error_message: str | None = None,
    params: Dict[str, Any] | None = None,
    started_at: datetime | None = None,
) -> int:
    """Insert a row into ``collection_history`` and return its id."""
    with SyncSession() as session:
        entry = CollectionHistory(
            collector_type=collector_type,
            status=status,
            records_fetched=records_fetched,
            records_inserted=records_inserted,
            records_updated=records_updated,
            records_skipped=records_skipped,
            error_message=error_message,
            params=params,
            started_at=started_at or datetime.now(UTC),
            finished_at=datetime.now(UTC) if status != "running" else None,
            progress=1.0 if status == "completed" else None,
        )
        session.add(entry)
        session.commit()
        return entry.id


# ------------------------------------------------------------------
# Celery task
# ------------------------------------------------------------------


@celery_app.task(bind=True, name="nflstats.run_import")
def run_import(
    self,
    collector_type: str,
    seasons: List[int],
    strategy: str,
) -> Dict[str, Any]:
    """Run the collector pipeline, persist results, and return a summary.

    Parameters
    ----------
    self : celery.Task
        Bound task instance — gives access to ``self.update_state``.
    collector_type : str
        Registered collector key (e.g. ``"nflreadpy"``).
    seasons : list[int]
        NFL seasons to fetch.
    strategy : str
        Collection strategy: merge | replace | append | dry_run.

    Returns
    -------
    dict
        Summary of the import (counts, status, etc.).
    """
    task_started = datetime.now(UTC)

    logger.info(
        "Celery task started: collector=%s seasons=%s strategy=%s",
        collector_type,
        seasons,
        strategy,
    )

    # Ensure tables exist before writing
    create_tables_sync()

    # Report initial progress
    self.update_state(
        state="PROGRESS",
        meta={
            "collector_type": collector_type,
            "seasons": seasons,
            "strategy": strategy,
            "current_season": None,
            "progress": 0.0,
        },
    )

    # Build a progress callback that pushes state updates to Celery
    def _progress(current: int, total: int, info: dict) -> None:
        self.update_state(
            state="PROGRESS",
            meta={
                "collector_type": collector_type,
                "seasons": seasons,
                "strategy": strategy,
                "current_season": info.get("season"),
                "seasons_completed": current,
                "seasons_total": total,
                "progress": round(current / total, 2) if total else 0,
                "total_players_so_far": info.get("total_players_so_far", 0),
            },
        )

    # Create the collector
    try:
        collector = CollectorFactory.create(
            collector_type,
            seasons=seasons,
            progress_callback=_progress,
        )
    except ValueError as exc:
        logger.error("Failed to create collector: %s", exc)
        _record_history(
            collector_type=collector_type,
            status="failed",
            error_message=str(exc),
            params={"seasons": seasons, "strategy": strategy},
            started_at=task_started,
        )
        return {"status": "failed", "error": str(exc)}

    # Run the async pipeline inside a one-shot event loop
    result = asyncio.run(collector.collect())

    if result is None:
        msg = f"Collection failed for {collector_type}. Check worker logs."
        logger.error(msg)
        _record_history(
            collector_type=collector_type,
            status="failed",
            error_message=msg,
            params={"seasons": seasons, "strategy": strategy},
            started_at=task_started,
        )
        return {"status": "failed", "error": msg}

    # Persist collected players to the database
    players: list[dict] = result.get("players", [])
    counters = _persist_players(players, strategy, collector_type)
    logger.info(
        "Persisted %d players: inserted=%d updated=%d skipped=%d",
        len(players),
        counters["inserted"],
        counters["updated"],
        counters["skipped"],
    )

    # Record the run in collection_history
    _record_history(
        collector_type=collector_type,
        status="completed",
        records_fetched=len(players),
        records_inserted=counters["inserted"],
        records_updated=counters["updated"],
        records_skipped=counters["skipped"],
        params={"seasons": seasons, "strategy": strategy},
        started_at=task_started,
    )

    return {
        "status": "completed",
        "collector_type": collector_type,
        "seasons": result.get("seasons", seasons),
        "strategy": strategy,
        "total_players": result.get("total_players", len(players)),
        "records_inserted": counters["inserted"],
        "records_updated": counters["updated"],
        "records_skipped": counters["skipped"],
        # Return a small sample in the result payload
        "players_sample": players[:20],
    }
