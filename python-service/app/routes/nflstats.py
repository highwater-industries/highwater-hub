"""
NFL Stats routes.

Endpoints for triggering and monitoring NFL statistics import jobs.
"""
from fastapi import APIRouter

router = APIRouter()


@router.post("/import")
async def start_import():
    """Kick off an NFL stats import job.

    Returns a job ID immediately. The actual import runs async.
    """
    # TODO: dispatch Celery task, return job ID
    return {"job_id": "not-yet-implemented", "status": "accepted"}


@router.get("/jobs/{job_id}")
async def get_job_status(job_id: str):
    """Check the status of an import job."""
    # TODO: look up job status from Postgres
    return {"job_id": job_id, "status": "unknown"}
