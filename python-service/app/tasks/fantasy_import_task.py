"""Celery task for importing fantasy league data.

Runs inside a Celery worker process.  Fetches league/team/roster data
from Yahoo or ESPN via the registered collectors, resolves roster
players against the master player list, and persists everything into
the ``fantasy_*`` tables.
"""

import asyncio
import logging
from datetime import UTC, datetime
from typing import Any, Dict, List, Optional

from celery.exceptions import SoftTimeLimitExceeded
from sqlalchemy import select

from app.celery_app import celery_app
from app.data_collectors import CollectorFactory
from app.database import SyncSession, create_tables_sync
from app.models.models import (
    CollectionHistory,
    FantasyLeague,
    FantasyMatchup,
    FantasyRoster,
    FantasyTeam,
)
from app.player_resolver import PlayerResolver

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# History helpers (reuse pattern from import_task.py)
# ------------------------------------------------------------------


def _record_history(
    collector_type: str,
    status: str,
    params: Optional[Dict[str, Any]] = None,
    started_at: Optional[datetime] = None,
    **kwargs: Any,
) -> int:
    """Insert a row into ``collection_history`` and return its id."""
    with SyncSession() as session:
        entry = CollectionHistory(
            collector_type=collector_type,
            status=status,
            params=params,
            started_at=started_at or datetime.now(UTC),
            finished_at=None,
            progress=0.0 if status in ("running", "pending") else None,
            **kwargs,
        )
        session.add(entry)
        session.commit()
        return entry.id


def _update_history(history_id: int, **kwargs: Any) -> None:
    """Update an existing ``collection_history`` entry in place."""
    with SyncSession() as session:
        entry = session.get(CollectionHistory, history_id)
        if entry is None:
            logger.warning("collection_history row %d not found", history_id)
            return
        for key, value in kwargs.items():
            setattr(entry, key, value)
        session.commit()


# ------------------------------------------------------------------
# Persistence
# ------------------------------------------------------------------


def _persist_fantasy_league(
    data: Dict[str, Any],
    resolver: PlayerResolver,
    platform: str,
) -> Dict[str, Any]:
    """Create/update league, teams, roster rows, and matchup rows.

    Returns a counters dict suitable for collection_history.
    """
    league_data = data["league"]
    teams_data = data["teams"]
    matchups_data = data.get("matchups", [])

    inserted = 0
    updated = 0
    skipped = 0
    matched_players = 0
    unmatched_players: List[Dict[str, str]] = []

    with SyncSession() as session:
        # ---- League upsert (dedup on external_league_id + platform + season)
        league = session.execute(
            select(FantasyLeague).where(
                FantasyLeague.external_league_id == str(league_data["external_league_id"]),
                FantasyLeague.platform == platform,
                FantasyLeague.season == league_data["season"],
            )
        ).scalar_one_or_none()

        if league is None:
            league = FantasyLeague(
                external_league_id=str(league_data["external_league_id"]),
                league_name=league_data["league_name"],
                platform=platform,
                season=league_data["season"],
                num_teams=league_data.get("num_teams"),
                scoring_type=league_data.get("scoring_type"),
                settings=league_data.get("settings"),
            )
            session.add(league)
            session.flush()  # populate league.id
            inserted += 1
            logger.info("Created league: %s", league_data["league_name"])
        else:
            league.league_name = league_data["league_name"]
            league.num_teams = league_data.get("num_teams")
            league.scoring_type = league_data.get("scoring_type")
            league.settings = league_data.get("settings")
            updated += 1
            logger.info("Updated league: %s", league_data["league_name"])

            # Clear existing teams + rosters for re-import (cascade)
            existing_teams = session.execute(
                select(FantasyTeam).where(FantasyTeam.league_id == league.id)
            ).scalars().all()
            for t in existing_teams:
                # Delete rosters for this team first
                session.execute(
                    select(FantasyRoster).where(FantasyRoster.team_id == t.id)
                )
                for r in session.execute(
                    select(FantasyRoster).where(FantasyRoster.team_id == t.id)
                ).scalars().all():
                    session.delete(r)
                session.delete(t)

            # Clear existing matchups for re-import
            existing_matchups = session.execute(
                select(FantasyMatchup).where(FantasyMatchup.league_id == league.id)
            ).scalars().all()
            for m in existing_matchups:
                session.delete(m)

            session.flush()

        # ---- Teams + rosters
        for team_data in teams_data:
            team = FantasyTeam(
                league_id=league.id,
                external_team_id=team_data.get("external_team_id"),
                team_name=team_data["team_name"],
                owner_name=team_data.get("owner_name"),
                wins=team_data.get("wins", 0),
                losses=team_data.get("losses", 0),
                ties=team_data.get("ties", 0),
                points_for=team_data.get("points_for", 0.0),
                points_against=team_data.get("points_against", 0.0),
                standing_rank=team_data.get("standing_rank"),
                playoff_seed=team_data.get("playoff_seed"),
                logo_url=team_data.get("logo_url", ""),
                streak_type=team_data.get("streak_type", ""),
                streak_value=team_data.get("streak_value", 0),
                waiver_priority=team_data.get("waiver_priority", 0),
                number_of_moves=team_data.get("number_of_moves", 0),
                number_of_trades=team_data.get("number_of_trades", 0),
                clinched_playoffs=team_data.get("clinched_playoffs", False),
                draft_grade=team_data.get("draft_grade", ""),
            )
            session.add(team)
            session.flush()  # populate team.id
            inserted += 1

            # Resolve and create roster entries
            roster_entries = team_data.get("roster", [])
            results = resolver.resolve_roster(roster_entries, source=platform)

            for entry, resolve_result in zip(roster_entries, results):
                roster_row = FantasyRoster(
                    team_id=team.id,
                    player_id=resolve_result.player_id,
                    player_name=resolve_result.player_name,
                    player_position=resolve_result.position,
                    nfl_team=resolve_result.team,
                    roster_position=entry.get("roster_position"),
                    external_player_id=entry.get("external_player_id"),
                    matched=resolve_result.matched,
                )
                session.add(roster_row)
                inserted += 1

                if resolve_result.matched:
                    matched_players += 1
                elif not resolve_result.skipped_defense:
                    unmatched_players.append({
                        "name": resolve_result.player_name,
                        "position": resolve_result.position,
                        "team": resolve_result.team,
                    })

        # ---- Matchups (weekly scores)
        for m in matchups_data:
            matchup_row = FantasyMatchup(
                league_id=league.id,
                week=m["week"],
                matchup_id=m["matchup_id"],
                team_name=m["team_name"],
                external_team_id=m.get("external_team_id"),
                points=m.get("points", 0.0),
                result=m.get("result"),
                is_playoff=m.get("is_playoff", False),
            )
            session.add(matchup_row)
            inserted += 1

        session.commit()
        logger.info(
            "Persisted %d matchup rows for league %s",
            len(matchups_data), league_data["league_name"],
        )

    return {
        "inserted": inserted,
        "updated": updated,
        "skipped": skipped,
        "matched_players": matched_players,
        "unmatched_players": unmatched_players,
    }


# ------------------------------------------------------------------
# Celery task
# ------------------------------------------------------------------


@celery_app.task(bind=True, name="fantasy.run_import")
def run_fantasy_import(
    self,
    platform: str,
    league_id: str,
    season: int,
    espn_swid: Optional[str] = None,
    espn_s2: Optional[str] = None,
    history_id: Optional[int] = None,
) -> Dict[str, Any]:
    """Fetch and persist a fantasy league from Yahoo or ESPN.

    Parameters
    ----------
    platform : str
        ``"yahoo"`` or ``"espn"``.
    league_id : str
        External league ID on the platform.
    season : int
        NFL season year.
    espn_swid, espn_s2 : str | None
        ESPN authentication cookies (private leagues only).
    history_id : int | None
        Pre-created ``collection_history`` row id.
    """
    task_started = datetime.now(UTC)
    collector_type = f"{platform}_fantasy"
    job_params = {
        "platform": platform,
        "league_id": league_id,
        "season": season,
        "celery_task_id": self.request.id,
    }

    logger.info(
        "Fantasy import started: platform=%s league=%s season=%d history_id=%s",
        platform, league_id, season, history_id,
    )

    create_tables_sync()

    # Transition pending → running, or create new history row
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

    # Progress callback → Celery state + DB
    def _progress(current: int, total: int, info: dict) -> None:
        frac = round(current / total, 2) if total else 0
        self.update_state(
            state="PROGRESS",
            meta={
                "platform": platform,
                "league_id": league_id,
                "progress": frac,
                **info,
            },
        )
        _update_history(history_id, progress=frac)

    # Build collector kwargs
    kwargs: Dict[str, Any] = {
        "league_id": league_id,
        "season": season,
        "progress_callback": _progress,
    }
    if platform == "espn":
        kwargs["swid"] = espn_swid
        kwargs["espn_s2"] = espn_s2

    # Create the collector via the factory
    try:
        collector = CollectorFactory.create(collector_type, **kwargs)
    except (ValueError, ImportError) as exc:
        msg = f"Failed to create {collector_type} collector: {exc}"
        logger.error(msg)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    # Run the async collect pipeline
    try:
        result = asyncio.run(collector.collect())
    except SoftTimeLimitExceeded:
        msg = f"Task timed out for {collector_type} league={league_id}"
        logger.error(msg)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    if result is None:
        msg = f"Collection returned None for {collector_type} league={league_id}"
        logger.error(msg)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    # Persist league + teams + rosters
    try:
        with SyncSession() as session:
            resolver = PlayerResolver(session)
            counters = _persist_fantasy_league(result, resolver, platform)
    except Exception as exc:
        msg = f"Persist failed for {collector_type} league={league_id}: {exc}"
        logger.error(msg, exc_info=True)
        _update_history(
            history_id,
            status="failed",
            error_message=msg,
            finished_at=datetime.now(UTC),
        )
        return {"status": "failed", "error": msg}

    logger.info(
        "Fantasy import complete: league=%s inserted=%d updated=%d "
        "matched=%d unmatched=%d",
        league_id,
        counters["inserted"],
        counters["updated"],
        counters["matched_players"],
        len(counters["unmatched_players"]),
    )

    _update_history(
        history_id,
        status="completed",
        records_fetched=result.get("total_players", 0),
        records_inserted=counters["inserted"],
        records_updated=counters["updated"],
        records_skipped=counters["skipped"],
        finished_at=datetime.now(UTC),
        progress=1.0,
    )

    return {
        "status": "completed",
        "platform": platform,
        "league_id": league_id,
        "season": season,
        "records_inserted": counters["inserted"],
        "records_updated": counters["updated"],
        "matched_players": counters["matched_players"],
        "unmatched_players": counters["unmatched_players"],
    }
