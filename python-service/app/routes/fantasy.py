"""Fantasy league import routes.

``POST /import`` dispatches a Celery task to import a Yahoo or ESPN
fantasy league.  The Go server triggers this endpoint and then polls
the existing ``/api/v1/nflstats/jobs/{job_id}`` endpoint (or the
collection_history table directly) for status.
"""

import asyncio
import logging

from fastapi import APIRouter, HTTPException

from app.schemas.fantasy import (
    ErrorOut,
    FantasyImportAccepted,
    FantasyImportRequest,
)
from app.tasks.fantasy_import_task import _record_history, run_fantasy_import

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post(
    "/import",
    response_model=FantasyImportAccepted,
    status_code=202,
    responses={400: {"model": ErrorOut}},
)
async def start_fantasy_import(body: FantasyImportRequest):
    """Dispatch a fantasy league import job to the Celery worker.

    Creates a "pending" row in ``collection_history`` so the job is
    visible in the UI before the worker picks it up.
    """
    platform = body.platform.value
    collector_type = f"{platform}_fantasy"

    logger.info(
        "Fantasy import requested: platform=%s league=%s season=%d swid=%s espn_s2=%s",
        platform,
        body.league_id,
        body.season,
        body.espn_swid[:8] + "..." if body.espn_swid else None,
        body.espn_s2[:8] + "..." if body.espn_s2 else None,
    )

    # ESPN private leagues require cookies
    if platform == "espn" and not body.espn_swid and not body.espn_s2:
        logger.info(
            "No ESPN cookies provided for league %s — will attempt public access",
            body.league_id,
        )

    # Pre-create a "pending" history row
    history_id: int = await asyncio.to_thread(
        _record_history,
        collector_type=collector_type,
        status="pending",
        params={
            "platform": platform,
            "league_id": body.league_id,
            "season": body.season,
        },
    )

    # Dispatch Celery task
    task = run_fantasy_import.delay(
        platform=platform,
        league_id=body.league_id,
        season=body.season,
        espn_swid=body.espn_swid,
        espn_s2=body.espn_s2,
        history_id=history_id,
    )

    logger.info(
        "Dispatched fantasy import task %s (history_id=%d)", task.id, history_id
    )

    return FantasyImportAccepted(
        job_id=task.id,
        status="accepted",
        platform=platform,
        league_id=body.league_id,
        season=body.season,
    )
