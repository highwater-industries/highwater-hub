"""Fantasy rankings collector using nflreadpy.

Fetches FantasyPros rankings via ``nfl.load_ff_rankings()``.
Auto-registers as ``"nflreadpy_ff_rankings"`` in the CollectorFactory.
"""

import logging
from typing import Any, Callable, Dict, List, Optional

import nflreadpy as nfl

from .base import DataCollector
from .factory import CollectorFactory

logger = logging.getLogger(__name__)


class FFRankingsCollector(DataCollector):
    """Collector for FantasyPros rankings via nflreadpy."""

    def __init__(
        self,
        seasons: Optional[List[int]] = None,
        rank_type: str = "draft",
        progress_callback: Optional[Callable] = None,
    ):
        super().__init__(name="FF Rankings (nflreadpy)")
        self.seasons = seasons or []
        self.rank_type = rank_type
        self.progress_callback = progress_callback

    async def fetch(self) -> Dict[str, Any]:
        logger.info("Fetching FF rankings type=%s", self.rank_type)
        df = nfl.load_ff_rankings(type=self.rank_type)

        if hasattr(df, "to_dicts"):
            rows = df.to_dicts()
        else:
            rows = df.to_dict(orient="records")

        logger.info("Fetched %d ranking rows", len(rows))
        return {"rank_type": self.rank_type, "rows": rows, "total_rows": len(rows)}

    def validate(self, data: Dict[str, Any]) -> bool:
        if not isinstance(data, dict) or "rows" not in data:
            return False
        rows = data["rows"]
        if not isinstance(rows, list) or len(rows) == 0:
            logger.warning("No ranking rows returned")
            return False
        return True

    async def transform(self, data: Dict[str, Any]) -> Dict[str, Any]:
        rankings: list[dict] = []
        for row in data["rows"]:
            player_name = row.get("player_name") or row.get("player") or ""
            ranking = {
                "player_id": row.get("player_id") or row.get("fantasypros_id"),
                "player_name": player_name,
                "pos": row.get("pos") or row.get("position"),
                "team": row.get("team"),
                "rank": row.get("rank") or row.get("rk"),
                "ecr": row.get("ecr"),
                "sd": row.get("sd"),
                "best": row.get("best"),
                "worst": row.get("worst"),
                "avg": row.get("avg"),
                "rank_type": data["rank_type"],
                "page_type": row.get("page_type"),
                "source": "nflreadpy",
            }
            rankings.append(ranking)

        return {
            "source": "nflreadpy_ff_rankings",
            "rank_type": data["rank_type"],
            "total_rankings": len(rankings),
            "rankings": rankings,
        }


CollectorFactory.register("nflreadpy_ff_rankings", FFRankingsCollector)
