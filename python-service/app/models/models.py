"""SQLAlchemy ORM models for the NFL stats service.

Mirrors the core tables from the old project (players, player_seasons,
collection_history) — enough to store the output of the NFLReadPyCollector.
"""

from datetime import UTC, datetime
from typing import Any, Dict, Optional

from sqlalchemy import (
    JSON,
    DateTime,
    Float,
    Index,
    Integer,
    String,
    Text,
    func,
)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    """Shared declarative base for all models."""


# ------------------------------------------------------------------
# Players — master player list
# ------------------------------------------------------------------


class PlayerDB(Base):
    """Master player record.  One row per unique NFL player."""

    __tablename__ = "players"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    player_id: Mapped[Optional[str]] = mapped_column(
        String(64), unique=True, index=True, nullable=True, comment="Stable external ID (gsis_id / esb_id / smart_id)"
    )
    player_name: Mapped[str] = mapped_column(String(128), nullable=False)
    team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    player_position: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    source: Mapped[Optional[str]] = mapped_column(String(32), nullable=True)
    metadata_: Mapped[Optional[Dict[str, Any]]] = mapped_column(
        "metadata", JSON, nullable=True, default=dict
    )
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now()
    )

    __table_args__ = (
        Index("ix_players_name_team", "player_name", "team"),
    )

    def __repr__(self) -> str:
        return f"<Player {self.player_name} ({self.team} {self.player_position})>"


# ------------------------------------------------------------------
# Player Seasons — per-season/week roster snapshots
# ------------------------------------------------------------------


class PlayerSeason(Base):
    """Per-season (optionally per-week) snapshot of a player's roster slot."""

    __tablename__ = "player_seasons"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    player_id: Mapped[str] = mapped_column(
        String(64), nullable=False, index=True
    )
    season: Mapped[int] = mapped_column(Integer, nullable=False)
    week: Mapped[int] = mapped_column(Integer, nullable=False, default=0, comment="0 = full-season aggregate")
    team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    position: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    metadata_: Mapped[Optional[Dict[str, Any]]] = mapped_column(
        "metadata", JSON, nullable=True, default=dict
    )
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    __table_args__ = (
        Index("ix_player_seasons_pid_season", "player_id", "season"),
    )

    def __repr__(self) -> str:
        return f"<PlayerSeason player_id={self.player_id} {self.season}/W{self.week}>"


# ------------------------------------------------------------------
# Collection History — tracks every import run
# ------------------------------------------------------------------


class CollectionHistory(Base):
    """Audit record for a single data-collection run."""

    __tablename__ = "collection_history"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    collector_type: Mapped[str] = mapped_column(String(32), nullable=False)
    status: Mapped[str] = mapped_column(
        String(16), nullable=False, default="running", comment="running | completed | failed"
    )
    records_fetched: Mapped[int] = mapped_column(Integer, default=0)
    records_inserted: Mapped[int] = mapped_column(Integer, default=0)
    records_updated: Mapped[int] = mapped_column(Integer, default=0)
    records_skipped: Mapped[int] = mapped_column(Integer, default=0)
    error_message: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    started_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )
    finished_at: Mapped[Optional[datetime]] = mapped_column(
        DateTime(timezone=True), nullable=True
    )
    params: Mapped[Optional[Dict[str, Any]]] = mapped_column(
        JSON, nullable=True, default=dict
    )
    progress: Mapped[Optional[float]] = mapped_column(
        Float, nullable=True, comment="0.0 – 1.0 progress fraction"
    )

    def __repr__(self) -> str:
        return f"<CollectionHistory {self.collector_type} {self.status}>"
