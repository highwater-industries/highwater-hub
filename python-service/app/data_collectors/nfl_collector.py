"""NFL data collector using nflreadpy.

Fetches NFL player roster data season-by-season and transforms it into
the internal player format.  Auto-registers as ``"nflreadpy"`` in the
:class:`CollectorFactory`.
"""

import asyncio
import logging
from typing import Any, Callable, Dict, List, Optional

import nflreadpy as nfl

from .base import DataCollector
from .factory import CollectorFactory

logger = logging.getLogger(__name__)


class NFLReadPyCollector(DataCollector):
    """Collector for NFL roster / player data via the *nflreadpy* package."""

    def __init__(
        self,
        seasons: List[int],
        progress_callback: Optional[Callable[[int, int, Dict[str, Any]], None]] = None,
    ):
        """
        Args:
            seasons: NFL seasons to fetch (e.g. ``[2023, 2024]``).
            progress_callback: Optional ``(current, total, info)`` function
                called after each season is fetched.
        """
        super().__init__(name="NFL Data (nflreadpy)")
        if not seasons:
            raise ValueError(
                "seasons is required for NFLReadPyCollector; "
                "provide one or more seasons explicitly"
            )
        self.seasons = seasons
        self.progress_callback = progress_callback

    # ------------------------------------------------------------------
    # Pipeline steps
    # ------------------------------------------------------------------

    async def fetch(self) -> Dict[str, Any]:
        """Fetch NFL roster data for each requested season."""
        logger.info(
            "Fetching NFL data for %d seasons: %s", len(self.seasons), self.seasons
        )

        all_rosters: list[dict] = []
        total_seasons = len(self.seasons)

        for idx, season in enumerate(self.seasons, start=1):
            try:
                logger.info("Fetching season %d (%d/%d)…", season, idx, total_seasons)
                roster_df = nfl.load_rosters([season])
                # Support both pandas (.to_dict) and polars (.to_dicts)
                if hasattr(roster_df, "to_dicts"):
                    roster_data = roster_df.to_dicts()
                else:
                    roster_data = roster_df.to_dict(orient="records")
                all_rosters.extend(roster_data)
                logger.info(
                    "Fetched %d players for season %d", len(roster_data), season
                )

                if self.progress_callback:
                    result = self.progress_callback(
                        idx,
                        total_seasons,
                        {
                            "season": season,
                            "players_fetched": len(roster_data),
                            "total_players_so_far": len(all_rosters),
                        },
                    )
                    if asyncio.iscoroutine(result):
                        await result

            except Exception as e:
                logger.error("Error fetching season %d: %s", season, e)
                # Continue with remaining seasons

        logger.info(
            "Completed fetching %d total players from %d seasons",
            len(all_rosters),
            total_seasons,
        )
        return {
            "seasons": self.seasons,
            "rosters": all_rosters,
            "total_players": len(all_rosters),
        }

    def validate(self, data: Dict[str, Any]) -> bool:
        """Check that the payload looks like valid roster data."""
        if not isinstance(data, dict):
            logger.error("Data is not a dictionary")
            return False
        if "rosters" not in data or not isinstance(data["rosters"], list):
            logger.error("Missing or invalid 'rosters' field")
            return False
        if len(data["rosters"]) == 0:
            logger.warning("No roster data returned")
            return False

        sample = data["rosters"][0]
        name_present = any(
            sample.get(key)
            for key in [
                "player_name",
                "full_name",
                "first_name",
                "last_name",
                "football_name",
            ]
        )
        if not name_present:
            logger.error("Roster entry missing any player name field")
            return False
        if "team" not in sample or "position" not in sample:
            logger.error("Roster entry missing team or position")
            return False

        return True

    async def transform(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Normalise each roster entry into the internal player dict."""
        players: list[dict] = []

        for entry in data["rosters"]:
            player_name = (
                entry.get("player_name")
                or entry.get("full_name")
                or " ".join(
                    filter(None, [entry.get("first_name"), entry.get("last_name")])
                ).strip()
                or entry.get("football_name")
            )
            player_id = (
                entry.get("player_id")
                or entry.get("gsis_id")
                or entry.get("esb_id")
                or entry.get("smart_id")
            )

            metadata = {
                "season": entry.get("season"),
                "week": entry.get("week"),
                "jersey_number": entry.get("jersey_number"),
                "status": entry.get("status"),
                "status_description": entry.get("status_description_abbr"),
                "depth_chart_position": entry.get("depth_chart_position"),
                "height": entry.get("height"),
                "weight": entry.get("weight"),
                "birth_date": entry.get("birth_date"),
                "college": entry.get("college"),
                "entry_year": entry.get("entry_year"),
                "rookie_year": entry.get("rookie_year"),
                "years_exp": entry.get("years_exp"),
                "first_name": entry.get("first_name"),
                "last_name": entry.get("last_name"),
                "football_name": entry.get("football_name"),
                "gsis_id": entry.get("gsis_id"),
                "gsis_it_id": entry.get("gsis_it_id"),
                "esb_id": entry.get("esb_id"),
                "pfr_id": entry.get("pfr_id"),
                "smart_id": entry.get("smart_id"),
                "espn_id": entry.get("espn_id"),
                "sleeper_id": entry.get("sleeper_id"),
                "yahoo_id": entry.get("yahoo_id"),
                "headshot_url": entry.get("headshot_url"),
                "draft_club": entry.get("draft_club"),
                "draft_number": entry.get("draft_number"),
            }
            # Strip None values from metadata
            metadata = {k: v for k, v in metadata.items() if v is not None}

            players.append(
                {
                    "player_id": player_id,
                    "player_name": player_name,
                    "player_position": entry.get("position"),
                    "team": entry.get("team"),
                    "source": "nflreadpy",
                    "metadata": metadata,
                }
            )

        return {
            "source": "nflreadpy",
            "seasons": data["seasons"],
            "total_players": len(players),
            "players": players,
        }


# Auto-register so ``CollectorFactory.create("nflreadpy", seasons=[2024])`` works.
CollectorFactory.register("nflreadpy", NFLReadPyCollector)
