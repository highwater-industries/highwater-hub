"""Pydantic schemas for NFL stats API requests and responses."""

from enum import Enum
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


# ------------------------------------------------------------------
# Enums
# ------------------------------------------------------------------


class CollectorType(str, Enum):
    """Supported data-collector types."""

    nflreadpy = "nflreadpy"


class CollectionStrategy(str, Enum):
    """How incoming records are reconciled with existing data."""

    merge = "merge"
    replace = "replace"
    append = "append"
    dry_run = "dry_run"


# ------------------------------------------------------------------
# Request schemas
# ------------------------------------------------------------------


class ImportRequest(BaseModel):
    """Body of ``POST /api/v1/nflstats/import``."""

    collector_type: CollectorType = Field(
        default=CollectorType.nflreadpy,
        description="Which collector to run.",
    )
    seasons: List[int] = Field(
        description="NFL seasons to import (e.g. [2023, 2024]).",
    )
    strategy: CollectionStrategy = Field(
        default=CollectionStrategy.merge,
        description="How to reconcile records: merge | replace | append | dry_run.",
    )

    model_config = {"json_schema_extra": {"examples": [{"seasons": [2024]}]}}


# ------------------------------------------------------------------
# Response schemas
# ------------------------------------------------------------------


class PlayerOut(BaseModel):
    """Public representation of a single player."""

    player_id: Optional[str] = None
    player_name: str
    team: Optional[str] = None
    player_position: Optional[str] = None
    source: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None


class ImportAccepted(BaseModel):
    """Returned by ``POST /import`` (HTTP 202).

    The ``job_id`` is a Celery task ID that can be polled via
    ``GET /jobs/{job_id}``.
    """

    job_id: str
    status: str = "accepted"
    collector_type: str
    seasons: List[int]
    strategy: str


class JobStatus(BaseModel):
    """Returned by ``GET /jobs/{job_id}``.

    States: pending → progress → completed | failed.
    """

    job_id: str
    status: str
    progress: Optional[float] = Field(
        None, description="0.0–1.0 fraction (available during 'progress' state)."
    )
    meta: Optional[Dict[str, Any]] = Field(
        None, description="Arbitrary progress metadata from the worker."
    )
    result: Optional[Dict[str, Any]] = Field(
        None, description="Final result payload (available when status='completed')."
    )
    error: Optional[str] = Field(
        None, description="Error message (available when status='failed')."
    )


class ErrorOut(BaseModel):
    """Uniform error envelope."""

    error: str
    detail: Optional[str] = None
