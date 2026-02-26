"""Schedules collector using nflreadpy.

Fetches NFL game schedules + results via ``nfl.load_schedules()``.
Auto-registers as ``"nflreadpy_schedules"`` in the CollectorFactory.
"""

import logging
from typing import Any, Callable, Dict, List, Optional

import nflreadpy as nfl

from .base import DataCollector
from .factory import CollectorFactory

logger = logging.getLogger(__name__)


class SchedulesCollector(DataCollector):
    """Collector for NFL game schedules via nflreadpy."""

    def __init__(
        self,
        seasons: List[int],
        progress_callback: Optional[Callable] = None,
    ):
        super().__init__(name="Schedules (nflreadpy)")
        if not seasons:
            raise ValueError("seasons is required for SchedulesCollector")
        self.seasons = seasons
        self.progress_callback = progress_callback

    async def fetch(self) -> Dict[str, Any]:
        logger.info("Fetching schedules for seasons=%s", self.seasons)
        df = nfl.load_schedules(self.seasons)

        if hasattr(df, "to_dicts"):
            rows = df.to_dicts()
        else:
            rows = df.to_dict(orient="records")

        logger.info("Fetched %d games", len(rows))
        return {"seasons": self.seasons, "rows": rows, "total_rows": len(rows)}

    def validate(self, data: Dict[str, Any]) -> bool:
        if not isinstance(data, dict) or "rows" not in data:
            return False
        rows = data["rows"]
        if not isinstance(rows, list) or len(rows) == 0:
            logger.warning("No schedule rows returned")
            return False
        sample = rows[0]
        if "game_id" not in sample:
            logger.error("Missing game_id in schedule data")
            return False
        return True

    async def transform(self, data: Dict[str, Any]) -> Dict[str, Any]:
        games: list[dict] = []
        for row in data["rows"]:
            game = {
                "game_id": row.get("game_id"),
                "season": row.get("season"),
                "game_type": row.get("game_type"),
                "week": row.get("week"),
                "gameday": row.get("gameday"),
                "weekday": row.get("weekday"),
                "gametime": row.get("gametime"),
                "away_team": row.get("away_team"),
                "home_team": row.get("home_team"),
                "away_score": row.get("away_score"),
                "home_score": row.get("home_score"),
                "result": row.get("result"),
                "total": row.get("total"),
                "spread_line": row.get("spread_line"),
                "total_line": row.get("total_line"),
                "overtime": row.get("overtime"),
                "location": row.get("location"),
                "roof": row.get("roof"),
                "surface": row.get("surface"),
                "stadium": row.get("stadium_id") or row.get("stadium"),
                "source": "nflreadpy",
            }
            games.append(game)

        return {
            "source": "nflreadpy_schedules",
            "seasons": data["seasons"],
            "total_games": len(games),
            "games": games,
        }


CollectorFactory.register("nflreadpy_schedules", SchedulesCollector)
