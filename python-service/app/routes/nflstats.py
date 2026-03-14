"""NFL Stats routes.

Endpoints for triggering and monitoring NFL statistics import jobs.

``POST /import`` dispatches a Celery task to a background worker and
returns a job ID immediately (HTTP 202 Accepted).  The Go server can
then poll ``GET /jobs/{job_id}`` until the job completes.
"""

import asyncio
import logging

from fastapi import APIRouter, HTTPException

from app.data_collectors import CollectorFactory
from app.schemas.nflstats import (
    ErrorOut,
    ImportAccepted,
    ImportRequest,
    JobStatus,
)
from app.tasks.import_task import _record_history, run_import

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post(
    "/import",
    response_model=ImportAccepted,
    status_code=202,
    responses={400: {"model": ErrorOut}},
)
async def start_import(body: ImportRequest):
    """Dispatch an NFL stats import job to the Celery worker.

    Creates a "pending" row in ``collection_history`` immediately so
    the job is visible in the UI before the Celery worker picks it up.
    """
    logger.info(
        "Import requested: collector=%s seasons=%s strategy=%s",
        body.collector_type.value,
        body.seasons,
        body.strategy.value,
    )

    # Validate that the collector type is registered before dispatching
    if body.collector_type.value not in CollectorFactory.get_available_types():
        raise HTTPException(
            status_code=400,
            detail=f"Unknown collector type: '{body.collector_type.value}'. "
                   f"Available: {', '.join(CollectorFactory.get_available_types())}",
        )

    # Build optional collector kwargs
    collector_kwargs: dict = {}
    if body.summary_level is not None:
        collector_kwargs["summary_level"] = body.summary_level
    if body.rank_type is not None:
        collector_kwargs["rank_type"] = body.rank_type

    # Pre-create a "pending" row so the job is visible in the UI immediately
    history_id: int = await asyncio.to_thread(
        _record_history,
        collector_type=body.collector_type.value,
        status="pending",
        params={
            "seasons": body.seasons,
            "strategy": body.strategy.value,
            **collector_kwargs,
        },
    )

    # Dispatch to Celery — returns immediately
    task = run_import.delay(
        collector_type=body.collector_type.value,
        seasons=body.seasons,
        strategy=body.strategy.value,
        collector_kwargs=collector_kwargs,
        history_id=history_id,
    )

    logger.info("Dispatched import task %s (history_id=%d)", task.id, history_id)
    return ImportAccepted(
        job_id=task.id,
        status="accepted",
        collector_type=body.collector_type.value,
        seasons=body.seasons,
        strategy=body.strategy.value,
    )


@router.get(
    "/jobs/{job_id}",
    response_model=JobStatus,
    responses={404: {"model": ErrorOut}},
)
async def get_job_status(job_id: str):
    """Poll the status of a previously dispatched import job.

    Possible states:
    - **PENDING** – task is queued but hasn't started yet.
    - **PROGRESS** – task is running; ``meta`` contains progress info.
    - **SUCCESS** – task finished; ``result`` holds the collected data.
    - **FAILURE** – task raised an exception; ``error`` has details.
    """
    result = run_import.AsyncResult(job_id)

    if result.state == "PENDING":
        return JobStatus(
            job_id=job_id,
            status="pending",
        )

    if result.state == "PROGRESS":
        meta = result.info or {}
        return JobStatus(
            job_id=job_id,
            status="progress",
            progress=meta.get("progress"),
            meta=meta,
        )

    if result.state == "SUCCESS":
        return JobStatus(
            job_id=job_id,
            status="completed",
            result=result.result,
        )

    if result.state == "FAILURE":
        return JobStatus(
            job_id=job_id,
            status="failed",
            error=str(result.result),
        )

    # Catch-all for unexpected Celery states
    return JobStatus(
        job_id=job_id,
        status=result.state.lower(),
        meta=result.info if isinstance(result.info, dict) else None,
    )
