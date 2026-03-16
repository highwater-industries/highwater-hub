"""Pydantic schemas for the fantasy league import API."""

from enum import Enum
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


# ------------------------------------------------------------------
# Enums
# ------------------------------------------------------------------


class FantasyPlatform(str, Enum):
    yahoo = "yahoo"
    espn = "espn"


# ------------------------------------------------------------------
# Request schemas
# ------------------------------------------------------------------


class FantasyImportRequest(BaseModel):
    """Body of ``POST /api/v1/fantasy/import``."""

    platform: FantasyPlatform = Field(
        description="Source platform: yahoo | espn",
    )
    league_id: str = Field(
        description="League ID on the source platform.",
    )
    season: int = Field(
        description="NFL season year (e.g. 2025).",
    )
    espn_swid: Optional[str] = Field(
        default=None,
        alias="swid",
        description="ESPN SWID cookie (private ESPN leagues only).",
    )
    espn_s2: Optional[str] = Field(
        default=None,
        description="ESPN espn_s2 cookie (private ESPN leagues only).",
    )

    model_config = {
        "populate_by_name": True,
        "json_schema_extra": {
            "examples": [
                {
                    "platform": "yahoo",
                    "league_id": "12345",
                    "season": 2025,
                },
            ],
        },
    }


# ------------------------------------------------------------------
# Response schemas
# ------------------------------------------------------------------


class FantasyImportAccepted(BaseModel):
    """Returned by ``POST /import`` (HTTP 202)."""

    job_id: str
    status: str = "accepted"
    platform: str
    league_id: str
    season: int


class ErrorOut(BaseModel):
    """Generic error response."""

    detail: str
