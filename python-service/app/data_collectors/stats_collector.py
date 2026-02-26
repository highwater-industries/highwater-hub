"""Player stats collector using nflreadpy.

Fetches weekly/seasonal player performance stats (passing, rushing,
receiving, fantasy points) via ``nfl.load_player_stats()``.
Auto-registers as ``"nflreadpy_stats"`` in the CollectorFactory.
"""

import asyncio
import logging
from typing import Any, Callable, Dict, List, Optional

import nflreadpy as nfl

from .base import DataCollector
from .factory import CollectorFactory

logger = logging.getLogger(__name__)


class PlayerStatsCollector(DataCollector):
    """Collector for weekly player stats via nflreadpy."""

    def __init__(
        self,
        seasons: List[int],
        summary_level: str = "week",
        progress_callback: Optional[Callable] = None,
    ):
        super().__init__(name="Player Stats (nflreadpy)")
        if not seasons:
            raise ValueError("seasons is required for PlayerStatsCollector")
        self.seasons = seasons
        self.summary_level = summary_level
        self.progress_callback = progress_callback

    async def fetch(self) -> Dict[str, Any]:
        logger.info(
            "Fetching player stats for seasons=%s summary_level=%s",
            self.seasons, self.summary_level,
        )
        df = nfl.load_player_stats(self.seasons, summary_level=self.summary_level)

        if hasattr(df, "to_dicts"):
            rows = df.to_dicts()
        else:
            rows = df.to_dict(orient="records")

        logger.info("Fetched %d stat rows", len(rows))
        return {
            "seasons": self.seasons,
            "summary_level": self.summary_level,
            "rows": rows,
            "total_rows": len(rows),
        }

    def validate(self, data: Dict[str, Any]) -> bool:
        if not isinstance(data, dict) or "rows" not in data:
            return False
        rows = data["rows"]
        if not isinstance(rows, list) or len(rows) == 0:
            logger.warning("No stat rows returned")
            return False
        sample = rows[0]
        if "player_id" not in sample:
            logger.error("Missing player_id in stats data")
            return False
        return True

    async def transform(self, data: Dict[str, Any]) -> Dict[str, Any]:
        stats: list[dict] = []
        for row in data["rows"]:
            player_name = (
                row.get("player_name")
                or row.get("player_display_name")
                or ""
            )
            stat = {
                "player_id": row.get("player_id"),
                "player_name": player_name,
                "player_display_name": row.get("player_display_name"),
                "position": row.get("position"),
                "position_group": row.get("position_group"),
                "team": row.get("recent_team") or row.get("team"),
                "season": row.get("season"),
                "week": row.get("week", 0),
                "season_type": row.get("season_type"),
                "opponent_team": row.get("opponent_team"),
                # Passing
                "completions": row.get("completions"),
                "attempts": row.get("attempts"),
                "passing_yards": row.get("passing_yards"),
                "passing_tds": row.get("passing_tds"),
                "interceptions": row.get("interceptions"),
                "sacks": row.get("sacks"),
                "sack_yards": row.get("sack_yards"),
                "passing_air_yards": row.get("passing_air_yards"),
                "passing_yards_after_catch": row.get("passing_yards_after_catch"),
                "passing_2pt_conversions": row.get("passing_2pt_conversions"),
                # Rushing
                "carries": row.get("carries"),
                "rushing_yards": row.get("rushing_yards"),
                "rushing_tds": row.get("rushing_tds"),
                "rushing_fumbles": row.get("rushing_fumbles"),
                "rushing_fumbles_lost": row.get("rushing_fumbles_lost"),
                "rushing_2pt_conversions": row.get("rushing_2pt_conversions"),
                # Receiving
                "receptions": row.get("receptions"),
                "targets": row.get("targets"),
                "receiving_yards": row.get("receiving_yards"),
                "receiving_tds": row.get("receiving_tds"),
                "receiving_fumbles": row.get("receiving_fumbles"),
                "receiving_fumbles_lost": row.get("receiving_fumbles_lost"),
                "receiving_air_yards": row.get("receiving_air_yards"),
                "receiving_yards_after_catch": row.get("receiving_yards_after_catch"),
                "receiving_2pt_conversions": row.get("receiving_2pt_conversions"),
                # Fantasy
                "fantasy_points": row.get("fantasy_points"),
                "fantasy_points_ppr": row.get("fantasy_points_ppr"),
                # Misc
                "special_teams_tds": row.get("special_teams_tds"),
                "source": "nflreadpy",
            }
            stats.append(stat)

        return {
            "source": "nflreadpy_stats",
            "seasons": data["seasons"],
            "summary_level": data["summary_level"],
            "total_rows": len(stats),
            "stats": stats,
        }


CollectorFactory.register("nflreadpy_stats", PlayerStatsCollector)
