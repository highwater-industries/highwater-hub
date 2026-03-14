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

from celery.exceptions import SoftTimeLimitExceeded
from sqlalchemy import delete, select

from app.celery_app import celery_app
from app.data_collectors import CollectorFactory
from app.database import SyncSession, create_tables_sync
from app.models.models import (
    CollectionHistory,
    FantasyRanking,
    Game,
    PlayerAlias,
    PlayerDB,
    PlayerStat,
)
from app.player_resolver import PlayerResolver

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

    # Seed aliases from newly imported roster data
    try:
        with SyncSession() as session:
            resolver = PlayerResolver(session)
            seeded = resolver.seed_from_roster()
            if seeded:
                logger.info("Seeded %d aliases from roster import", seeded)
    except Exception as exc:
        logger.warning("Failed to seed aliases: %s", exc)

    return {"inserted": inserted, "updated": updated, "skipped": skipped}


def _persist_stats(
    stats: List[Dict[str, Any]],
    strategy: str,
    collector_type: str,
) -> Dict[str, int]:
    """Write collected stat dicts to the ``player_stats`` table."""
    inserted = updated = skipped = 0

    with SyncSession() as session:
        if strategy == "replace":
            session.execute(
                delete(PlayerStat).where(PlayerStat.source == collector_type)
            )
            session.flush()

        for s in stats:
            if strategy == "dry_run":
                skipped += 1
                continue

            pid = s.get("player_id")
            season = s.get("season")
            week = s.get("week", 0)
            stat_type = s.get("stat_type", "actual")
            source = s.get("source", collector_type)

            # Skip rows with no player_id (team aggregates / unknown players)
            if not pid:
                skipped += 1
                continue

            existing = None
            if pid and season is not None and strategy == "merge":
                existing = session.execute(
                    select(PlayerStat).where(
                        PlayerStat.player_id == pid,
                        PlayerStat.season == season,
                        PlayerStat.week == week,
                        PlayerStat.stat_type == stat_type,
                        PlayerStat.source == source,
                    )
                ).scalar_one_or_none()

            if existing:
                for col in (
                    "player_name", "player_display_name", "position",
                    "position_group", "team", "season_type", "opponent_team",
                    "completions", "attempts", "passing_yards", "passing_tds",
                    "interceptions", "sacks", "sack_yards", "passing_air_yards",
                    "passing_yards_after_catch", "passing_2pt_conversions",
                    "carries", "rushing_yards", "rushing_tds", "rushing_fumbles",
                    "rushing_fumbles_lost", "rushing_2pt_conversions",
                    "receptions", "targets", "receiving_yards", "receiving_tds",
                    "receiving_fumbles", "receiving_fumbles_lost",
                    "receiving_air_yards", "receiving_yards_after_catch",
                    "receiving_2pt_conversions",
                    "fantasy_points", "fantasy_points_ppr", "special_teams_tds",
                ):
                    if s.get(col) is not None:
                        setattr(existing, col, s[col])
                updated += 1
            else:
                session.add(PlayerStat(
                    player_id=pid,
                    player_name=s.get("player_name", ""),
                    player_display_name=s.get("player_display_name"),
                    position=s.get("position"),
                    position_group=s.get("position_group"),
                    team=s.get("team"),
                    season=season,
                    week=week,
                    stat_type=stat_type,
                    season_type=s.get("season_type"),
                    opponent_team=s.get("opponent_team"),
                    completions=s.get("completions"),
                    attempts=s.get("attempts"),
                    passing_yards=s.get("passing_yards"),
                    passing_tds=s.get("passing_tds"),
                    interceptions=s.get("interceptions"),
                    sacks=s.get("sacks"),
                    sack_yards=s.get("sack_yards"),
                    passing_air_yards=s.get("passing_air_yards"),
                    passing_yards_after_catch=s.get("passing_yards_after_catch"),
                    passing_2pt_conversions=s.get("passing_2pt_conversions"),
                    carries=s.get("carries"),
                    rushing_yards=s.get("rushing_yards"),
                    rushing_tds=s.get("rushing_tds"),
                    rushing_fumbles=s.get("rushing_fumbles"),
                    rushing_fumbles_lost=s.get("rushing_fumbles_lost"),
                    rushing_2pt_conversions=s.get("rushing_2pt_conversions"),
                    receptions=s.get("receptions"),
                    targets=s.get("targets"),
                    receiving_yards=s.get("receiving_yards"),
                    receiving_tds=s.get("receiving_tds"),
                    receiving_fumbles=s.get("receiving_fumbles"),
                    receiving_fumbles_lost=s.get("receiving_fumbles_lost"),
                    receiving_air_yards=s.get("receiving_air_yards"),
                    receiving_yards_after_catch=s.get("receiving_yards_after_catch"),
                    receiving_2pt_conversions=s.get("receiving_2pt_conversions"),
                    fantasy_points=s.get("fantasy_points"),
                    fantasy_points_ppr=s.get("fantasy_points_ppr"),
                    special_teams_tds=s.get("special_teams_tds"),
                    source=s.get("source", collector_type),
                ))
                inserted += 1

        session.commit()

    return {"inserted": inserted, "updated": updated, "skipped": skipped}


def _persist_games(
    games: List[Dict[str, Any]],
    strategy: str,
    collector_type: str,
) -> Dict[str, int]:
    """Write collected game dicts to the ``games`` table."""
    inserted = updated = skipped = 0

    with SyncSession() as session:
        if strategy == "replace":
            session.execute(
                delete(Game).where(Game.source == collector_type)
            )
            session.flush()

        for g in games:
            if strategy == "dry_run":
                skipped += 1
                continue

            gid = g.get("game_id")

            existing = None
            if gid and strategy == "merge":
                existing = session.execute(
                    select(Game).where(Game.game_id == gid)
                ).scalar_one_or_none()

            if existing:
                for col in (
                    "season", "game_type", "week", "gameday", "weekday",
                    "gametime", "away_team", "home_team", "away_score",
                    "home_score", "result", "total", "spread_line",
                    "total_line", "overtime", "location", "roof",
                    "surface", "stadium",
                ):
                    if g.get(col) is not None:
                        setattr(existing, col, g[col])
                updated += 1
            else:
                session.add(Game(
                    game_id=gid,
                    season=g.get("season"),
                    game_type=g.get("game_type"),
                    week=g.get("week"),
                    gameday=g.get("gameday"),
                    weekday=g.get("weekday"),
                    gametime=g.get("gametime"),
                    away_team=g.get("away_team"),
                    home_team=g.get("home_team"),
                    away_score=g.get("away_score"),
                    home_score=g.get("home_score"),
                    result=g.get("result"),
                    total=g.get("total"),
                    spread_line=g.get("spread_line"),
                    total_line=g.get("total_line"),
                    overtime=g.get("overtime"),
                    location=g.get("location"),
                    roof=g.get("roof"),
                    surface=g.get("surface"),
                    stadium=g.get("stadium"),
                    source=g.get("source", collector_type),
                ))
                inserted += 1

        session.commit()

    return {"inserted": inserted, "updated": updated, "skipped": skipped}


def _persist_ff_rankings(
    rankings: List[Dict[str, Any]],
    strategy: str,
    collector_type: str,
) -> Dict[str, int]:
    """Write collected ranking dicts to the ``fantasy_rankings`` table."""
    inserted = updated = skipped = 0

    with SyncSession() as session:
        # Scope delete to matching source + rank_type (not wholesale nuke)
        if strategy in ("replace", "merge"):
            q = delete(FantasyRanking).where(FantasyRanking.source == collector_type)
            session.execute(q)
            session.flush()

        # Create a resolver for name → player_id lookups
        resolver = PlayerResolver(session)

        for r in rankings:
            if strategy == "dry_run":
                skipped += 1
                continue

            # Try to resolve player_id if not already set
            pid = r.get("player_id")
            if not pid:
                pid = resolver.resolve(
                    name=r.get("player_name", ""),
                    team=r.get("team"),
                    source=collector_type,
                )

            session.add(FantasyRanking(
                player_id=pid,
                player_name=r.get("player_name", ""),
                pos=r.get("pos"),
                team=r.get("team"),
                rank=r.get("rank"),
                ecr=r.get("ecr"),
                sd=r.get("sd"),
                best=r.get("best"),
                worst=r.get("worst"),
                avg=r.get("avg"),
                rank_type=r.get("rank_type"),
                page_type=r.get("page_type"),
                season=r.get("season", 0),
                week=r.get("week", 0),
                source=r.get("source", collector_type),
            ))
            inserted += 1

        session.commit()

    return {"inserted": inserted, "updated": updated, "skipped": skipped}


# Map collector type → (persist_fn, data_key)
_PERSIST_MAP: Dict[str, tuple] = {
    "nflreadpy": (_persist_players, "players"),
    "nflreadpy_stats": (_persist_stats, "stats"),
    "nflreadpy_schedules": (_persist_games, "games"),
    "nflreadpy_ff_rankings": (_persist_ff_rankings, "rankings"),
}


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
            finished_at=datetime.now(UTC) if status not in ("running", "pending") else None,
            progress=1.0 if status == "completed" else (0.0 if status in ("running", "pending") else None),
        )
        session.add(entry)
        session.commit()
        return entry.id


def _update_history(history_id: int, **kwargs: Any) -> None:
    """Update an existing ``collection_history`` entry in place."""
    with SyncSession() as session:
        entry = session.get(CollectionHistory, history_id)
        if entry is None:
            logger.warning("collection_history row %d not found for update", history_id)
            return
        for key, value in kwargs.items():
            setattr(entry, key, value)
        session.commit()


# ------------------------------------------------------------------
# Celery task
# ------------------------------------------------------------------


@celery_app.task(bind=True, name="nflstats.run_import")
def run_import(
    self,
    collector_type: str,
    seasons: List[int],
    strategy: str,
    collector_kwargs: Dict[str, Any] | None = None,
    history_id: int | None = None,
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
    collector_kwargs : dict | None
        Extra keyword arguments forwarded to the collector constructor
        (e.g. ``summary_level``, ``rank_type``).
    history_id : int | None
        If provided, update this existing collection_history row instead
        of creating a new one.  The ``start_import`` route pre-creates a
        "pending" row so the job is visible in the UI before the worker
        picks it up.

    Returns
    -------
    dict
        Summary of the import (counts, status, etc.).
    """
    if collector_kwargs is None:
        collector_kwargs = {}

    task_started = datetime.now(UTC)
    job_params = {"seasons": seasons, "strategy": strategy, "celery_task_id": self.request.id, **collector_kwargs}

    logger.info(
        "Celery task started: collector=%s seasons=%s strategy=%s history_id=%s",
        collector_type,
        seasons,
        strategy,
        history_id,
    )

    # Ensure tables exist before writing
    create_tables_sync()

    # Transition existing "pending" row to "running", or create a fresh one
    if history_id:
        _update_history(
            history_id,
            status="running",
            started_at=task_started,
            progress=0.0,
            params=job_params,
        )
    else:
        history_id = _record_history(
            collector_type=collector_type,
            status="running",
            params=job_params,
            started_at=task_started,
        )

    # Report initial progress via Celery broker
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

    # Build a progress callback that pushes state updates to Celery + DB
    def _progress(current: int, total: int, info: dict) -> None:
        frac = round(current / total, 2) if total else 0
        self.update_state(
            state="PROGRESS",
            meta={
                "collector_type": collector_type,
                "seasons": seasons,
                "strategy": strategy,
                "current_season": info.get("season"),
                "seasons_completed": current,
                "seasons_total": total,
                "progress": frac,
                "total_players_so_far": info.get("total_players_so_far", 0),
            },
        )
        # Update progress in the database so the list endpoint reflects it
        if history_id:
            _update_history(history_id, progress=frac)

    # Create the collector
    try:
        collector = CollectorFactory.create(
            collector_type,
            seasons=seasons,
            progress_callback=_progress,
            **collector_kwargs,
        )
    except ValueError as exc:
        logger.error("Failed to create collector: %s", exc)
        _update_history(
            history_id,
            status="failed",
            error_message=str(exc),
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": str(exc)}

    # Run the async pipeline inside a one-shot event loop
    try:
        result = asyncio.run(collector.collect())
    except SoftTimeLimitExceeded:
        msg = (
            f"Task timed out after soft limit "
            f"({celery_app.conf.task_soft_time_limit}s) "
            f"for {collector_type} seasons={seasons}"
        )
        logger.error(msg)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    if result is None:
        msg = f"Collection failed for {collector_type}. Check worker logs."
        logger.error(msg)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    # Dispatch to the correct persist function
    persist_fn, data_key = _PERSIST_MAP.get(
        collector_type, (_persist_players, "players")
    )
    records: list[dict] = result.get(data_key, [])
    try:
        counters = persist_fn(records, strategy, collector_type)
    except SoftTimeLimitExceeded:
        msg = (
            f"Task timed out during DB persist "
            f"({celery_app.conf.task_soft_time_limit}s) "
            f"for {collector_type} seasons={seasons}"
        )
        logger.error(msg)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    logger.info(
        "Persisted %d %s: inserted=%d updated=%d skipped=%d",
        len(records),
        data_key,
        counters["inserted"],
        counters["updated"],
        counters["skipped"],
    )

    # Update the collection_history row to completed
    _update_history(
        history_id,
        status="completed",
        records_fetched=len(records),
        records_inserted=counters["inserted"],
        records_updated=counters["updated"],
        records_skipped=counters["skipped"],
        finished_at=datetime.now(UTC),
        progress=1.0,
    )

    return {
        "status": "completed",
        "collector_type": collector_type,
        "seasons": result.get("seasons", seasons),
        "strategy": strategy,
        "total_records": len(records),
        "records_inserted": counters["inserted"],
        "records_updated": counters["updated"],
        "records_skipped": counters["skipped"],
        "sample": records[:20],
    }
