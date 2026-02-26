"""SQLAlchemy ORM models for the NFL stats service.

Mirrors the core tables from the old project (players, player_seasons,
collection_history) — enough to store the output of the NFLReadPyCollector.
"""

from datetime import UTC, datetime
from typing import Any, Dict, Optional

from sqlalchemy import (
    JSON,
    Boolean,
    DateTime,
    Float,
    Index,
    Integer,
    String,
    Text,
    UniqueConstraint,
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
    source: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
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
# Player Aliases — maps name variants to canonical player_id
# ------------------------------------------------------------------


class PlayerAlias(Base):
    """Maps name variations to canonical player_id.

    When a data source gives only a name (no gsis_id), the resolver
    checks this table to find the canonical player_id.  Aliases accumulate
    over time — the first import with a real player_id seeds the canonical name,
    and subsequent sources that use a different name spelling get entries added here.
    """

    __tablename__ = "player_aliases"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    player_id: Mapped[str] = mapped_column(
        String(64), nullable=False, index=True,
        comment="Canonical player_id (FK to players.player_id)",
    )
    alias_name: Mapped[str] = mapped_column(
        String(128), nullable=False,
        comment="Alternate spelling / format of the player name",
    )
    team: Mapped[Optional[str]] = mapped_column(
        String(8), nullable=True,
        comment="Team hint to disambiguate common names",
    )
    source: Mapped[Optional[str]] = mapped_column(
        String(64), nullable=True,
        comment="Data source that uses this name variant",
    )
    auto_matched: Mapped[bool] = mapped_column(
        Boolean, nullable=False, default=False,
        comment="True if this alias was auto-created by fuzzy matching",
    )
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    __table_args__ = (
        UniqueConstraint("alias_name", "source", "team", name="uq_alias_name_source_team"),
        Index("ix_player_aliases_name_source", "alias_name", "source"),
    )

    def __repr__(self) -> str:
        return f'<PlayerAlias "{self.alias_name}" → {self.player_id} ({self.source})>'


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
# Player Stats — weekly/seasonal performance stats
# ------------------------------------------------------------------


class PlayerStat(Base):
    """Weekly or seasonal player performance stats from load_player_stats().

    One row per player per season per week per stat_type per source.
    When summary_level='reg', week is stored as 0 to indicate a season aggregate.

    stat_type values:
      - actual    : real game performance stats (e.g. nflreadpy)
      - projected : pre-game projections (e.g. ESPN, FantasyPros)
      - fantasy   : scored fantasy-point sets (custom or platform-specific)
    """

    __tablename__ = "player_stats"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    player_id: Mapped[str] = mapped_column(String(64), nullable=False, index=True)
    player_name: Mapped[str] = mapped_column(String(128), nullable=False)
    player_display_name: Mapped[Optional[str]] = mapped_column(String(128), nullable=True)
    position: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    position_group: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    season: Mapped[int] = mapped_column(Integer, nullable=False)
    week: Mapped[int] = mapped_column(Integer, nullable=False, default=0)
    stat_type: Mapped[str] = mapped_column(
        String(32), nullable=False, default="actual",
        comment="actual | projected | fantasy",
    )
    season_type: Mapped[Optional[str]] = mapped_column(String(8), nullable=True, comment="REG / POST")
    opponent_team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)

    # Passing
    completions: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    attempts: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    passing_yards: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    passing_tds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    interceptions: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    sacks: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    sack_yards: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    passing_air_yards: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    passing_yards_after_catch: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    passing_2pt_conversions: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Rushing
    carries: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    rushing_yards: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    rushing_tds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    rushing_fumbles: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    rushing_fumbles_lost: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    rushing_2pt_conversions: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Receiving
    receptions: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    targets: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    receiving_yards: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    receiving_tds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    receiving_fumbles: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    receiving_fumbles_lost: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    receiving_air_yards: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    receiving_yards_after_catch: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    receiving_2pt_conversions: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Fantasy
    fantasy_points: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    fantasy_points_ppr: Mapped[Optional[float]] = mapped_column(Float, nullable=True)

    # Misc
    special_teams_tds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Extra fields — anything else goes in metadata
    metadata_: Mapped[Optional[Dict[str, Any]]] = mapped_column(
        "metadata", JSON, nullable=True, default=dict
    )
    source: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    __table_args__ = (
        Index("ix_player_stats_pid_season_week_type_src", "player_id", "season", "week", "stat_type", "source"),
        Index("ix_player_stats_team_season", "team", "season"),
        Index("ix_player_stats_position_season", "position", "season"),
        Index("ix_player_stats_stat_type", "stat_type"),
    )

    def __repr__(self) -> str:
        return f"<PlayerStat {self.player_name} {self.season}/W{self.week} {self.stat_type}/{self.source}>"


# ------------------------------------------------------------------
# Games — schedule and results
# ------------------------------------------------------------------


class Game(Base):
    """NFL game schedule and results from load_schedules()."""

    __tablename__ = "games"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    game_id: Mapped[str] = mapped_column(
        String(64), unique=True, nullable=False, index=True,
        comment="nflverse game_id e.g. 2024_01_KC_BAL"
    )
    season: Mapped[int] = mapped_column(Integer, nullable=False)
    game_type: Mapped[Optional[str]] = mapped_column(String(8), nullable=True, comment="REG / WC / DIV / CON / SB")
    week: Mapped[int] = mapped_column(Integer, nullable=False)
    gameday: Mapped[Optional[str]] = mapped_column(String(16), nullable=True)
    weekday: Mapped[Optional[str]] = mapped_column(String(16), nullable=True)
    gametime: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)

    away_team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    home_team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    away_score: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    home_score: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    result: Mapped[Optional[int]] = mapped_column(Integer, nullable=True, comment="home_score - away_score")
    total: Mapped[Optional[int]] = mapped_column(Integer, nullable=True, comment="total points scored")
    spread_line: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    total_line: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    overtime: Mapped[Optional[int]] = mapped_column(Integer, nullable=True, comment="1 if OT")

    location: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
    roof: Mapped[Optional[str]] = mapped_column(String(16), nullable=True)
    surface: Mapped[Optional[str]] = mapped_column(String(32), nullable=True)
    stadium: Mapped[Optional[str]] = mapped_column(String(128), nullable=True)

    metadata_: Mapped[Optional[Dict[str, Any]]] = mapped_column(
        "metadata", JSON, nullable=True, default=dict
    )
    source: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    __table_args__ = (
        Index("ix_games_season_week", "season", "week"),
        Index("ix_games_teams", "home_team", "away_team"),
    )

    def __repr__(self) -> str:
        return f"<Game {self.game_id}>"


# ------------------------------------------------------------------
# Fantasy Rankings — third-party projections
# ------------------------------------------------------------------


class FantasyRanking(Base):
    """Fantasy rankings/projections from FantasyPros via load_ff_rankings().

    season + week allow storing rankings for different points in the year.
    source allows storing rankings from different providers side-by-side.
    """

    __tablename__ = "fantasy_rankings"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    player_id: Mapped[Optional[str]] = mapped_column(String(64), nullable=True, index=True)
    player_name: Mapped[str] = mapped_column(String(128), nullable=False)
    pos: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    team: Mapped[Optional[str]] = mapped_column(String(8), nullable=True)
    rank: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    ecr: Mapped[Optional[float]] = mapped_column(Float, nullable=True, comment="Expert consensus ranking")
    sd: Mapped[Optional[float]] = mapped_column(Float, nullable=True, comment="Standard deviation")
    best: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    worst: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    avg: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    rank_type: Mapped[Optional[str]] = mapped_column(String(16), nullable=True, comment="draft / week / all")
    page_type: Mapped[Optional[str]] = mapped_column(String(32), nullable=True)
    season: Mapped[int] = mapped_column(Integer, nullable=False, default=0, comment="NFL season year, 0=unspecified")
    week: Mapped[int] = mapped_column(Integer, nullable=False, default=0, comment="0=preseason/draft, 1-18=weekly")

    metadata_: Mapped[Optional[Dict[str, Any]]] = mapped_column(
        "metadata", JSON, nullable=True, default=dict
    )
    source: Mapped[Optional[str]] = mapped_column(String(64), nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now()
    )

    __table_args__ = (
        Index("ix_fantasy_rankings_pos_rank", "pos", "rank"),
        Index("ix_fantasy_rankings_season_week", "season", "week"),
        Index("ix_fantasy_rankings_pid_type_season_src", "player_id", "rank_type", "season", "week", "source"),
    )

    def __repr__(self) -> str:
        return f"<FantasyRanking {self.player_name} rank={self.rank} {self.rank_type}/{self.source}>"


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
