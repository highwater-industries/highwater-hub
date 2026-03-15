"""ESPN Fantasy Football league data collector.

Fetches league metadata, teams, and rosters from the ESPN Fantasy API
using cookie-based authentication (SWID + espn_s2).

Registers as ``"espn_fantasy"`` in the :class:`CollectorFactory`.

Dependencies:
    pip install httpx
"""

from datetime import datetime, timezone
from typing import Any, Callable, Dict, List, Optional
import logging

import httpx

from .base import DataCollector
from .factory import CollectorFactory

logger = logging.getLogger(__name__)

# ESPN position-ID → human-readable code
_POSITION_MAP: Dict[int, str] = {
    1: "QB", 2: "RB", 3: "WR", 4: "TE",
    5: "K", 16: "D/ST", 0: "Unknown",
}

# ESPN pro-team-ID → NFL abbreviation
_TEAM_MAP: Dict[int, str] = {
    1: "ATL", 2: "BUF", 3: "CHI", 4: "CIN", 5: "CLE",
    6: "DAL", 7: "DEN", 8: "DET", 9: "GB", 10: "TEN",
    11: "IND", 12: "KC", 13: "LV", 14: "LAR", 15: "MIA",
    16: "MIN", 17: "NE", 18: "NO", 19: "NYG", 20: "NYJ",
    21: "PHI", 22: "ARI", 23: "PIT", 24: "LAC", 25: "SF",
    26: "SEA", 27: "TB", 28: "WAS", 29: "CAR", 30: "JAX",
    33: "BAL", 34: "HOU",
}

# ESPN lineup-slot-ID → roster-position label
_SLOT_MAP: Dict[int, str] = {
    0: "QB", 2: "RB", 4: "WR", 6: "TE",
    16: "D/ST", 17: "K", 20: "BENCH", 21: "IR",
    23: "FLEX", 24: "OP",
}


class ESPNFantasyCollector(DataCollector):
    """Collect league, team, and roster data from ESPN Fantasy Football.

    For **private** leagues the user must supply two browser cookies:

    * ``swid`` — the ESPN SWID cookie
    * ``espn_s2`` — the ESPN espn_s2 cookie

    Public leagues can be fetched without any credentials.
    """

    BASE_URL = "https://fantasy.espn.com/apis/v3/games/ffl"

    def __init__(
        self,
        league_id: str,
        season: int,
        swid: Optional[str] = None,
        espn_s2: Optional[str] = None,
        progress_callback: Optional[Callable[[int, int, Dict[str, Any]], None]] = None,
        **_kwargs: Any,
    ):
        super().__init__(name="ESPN Fantasy")
        self.league_id = league_id
        self.season = season
        self.swid = swid
        self.espn_s2 = espn_s2
        self.progress_callback = progress_callback

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _cookies(self) -> Dict[str, str]:
        cookies: Dict[str, str] = {}
        if self.swid:
            cookies["SWID"] = self.swid
        if self.espn_s2:
            cookies["espn_s2"] = self.espn_s2
        return cookies

    def _progress(self, current: int, total: int, info: Dict[str, Any]) -> None:
        if self.progress_callback:
            self.progress_callback(current, total, info)

    @staticmethod
    def _position(pos_id: int) -> str:
        return _POSITION_MAP.get(pos_id, "Unknown")

    @staticmethod
    def _nfl_team(team_id: int) -> str:
        return _TEAM_MAP.get(team_id, "FA")

    @staticmethod
    def _slot(slot_id: int) -> str:
        return _SLOT_MAP.get(slot_id, "BENCH")

    # ------------------------------------------------------------------
    # DataCollector interface
    # ------------------------------------------------------------------

    async def fetch(self) -> Dict[str, Any]:
        """Hit the ESPN Fantasy API and return structured league data."""
        self._progress(1, 4, {"step": "Connecting to ESPN API"})

        url = (
            f"{self.BASE_URL}/seasons/{self.season}"
            f"/segments/0/leagues/{self.league_id}"
        )

        async with httpx.AsyncClient(timeout=30.0) as client:
            resp = await client.get(
                url,
                cookies=self._cookies(),
                params={"view": ["mTeam", "mRoster", "mSettings"]},
            )
            resp.raise_for_status()
            data: Dict[str, Any] = resp.json()

        self._progress(2, 4, {"step": "Processing league data"})
        league_info = self._extract_league(data)

        self._progress(3, 4, {"step": "Processing teams & rosters"})
        members = {
            m["id"]: m.get("displayName", "Unknown")
            for m in data.get("members", [])
        }
        teams = self._extract_teams(data, members)

        self._progress(4, 4, {"step": "Complete"})

        total_players = sum(len(t.get("roster", [])) for t in teams)
        logger.info(
            "Fetched ESPN league '%s': %d teams, %d roster entries",
            league_info.get("league_name", "?"),
            len(teams),
            total_players,
        )

        return {
            "league": league_info,
            "teams": teams,
            "platform": "espn",
            "season": self.season,
            "total_players": total_players,
        }

    def validate(self, data: Dict[str, Any]) -> bool:
        if not isinstance(data, dict):
            return False
        if "league" not in data or "teams" not in data:
            return False
        return True

    async def transform(self, data: Dict[str, Any]) -> Dict[str, Any]:
        return data

    # ------------------------------------------------------------------
    # Extraction helpers
    # ------------------------------------------------------------------

    def _extract_league(self, raw: Dict[str, Any]) -> Dict[str, Any]:
        settings = raw.get("settings", {})
        return {
            "external_league_id": str(raw.get("id", self.league_id)),
            "league_name": settings.get("name", f"ESPN League {self.league_id}"),
            "platform": "espn",
            "season": self.season,
            "num_teams": len(raw.get("teams", [])),
            "scoring_type": None,
            "settings": {
                "schedule_type": settings.get("scheduleSettings", {}).get("type"),
                "playoff_team_count": settings.get("scheduleSettings", {}).get("playoffTeamCount"),
            },
        }

    def _extract_teams(
        self,
        raw: Dict[str, Any],
        members: Dict[str, str],
    ) -> List[Dict[str, Any]]:
        teams: List[Dict[str, Any]] = []

        for t in raw.get("teams", []):
            team_id = str(t["id"])
            team_name = (
                t.get("name")
                or (t.get("location", "") + " " + t.get("nickname", "")).strip()
                or f"Team {team_id}"
            )

            owner_id = t.get("primaryOwner")
            owner_name = members.get(owner_id, "Unknown") if owner_id else "Unknown"

            record = t.get("record", {}).get("overall", {})

            roster: List[Dict[str, Any]] = []
            for entry in t.get("roster", {}).get("entries", []):
                player_raw = entry.get("playerPoolEntry", {}).get("player", {})
                pid = str(player_raw.get("id", ""))
                if not pid:
                    continue

                roster.append({
                    "player_name": player_raw.get("fullName", "Unknown"),
                    "player_position": self._position(
                        player_raw.get("defaultPositionId", 0)
                    ),
                    "team": self._nfl_team(player_raw.get("proTeamId", 0)),
                    "roster_position": self._slot(entry.get("lineupSlotId", 20)),
                    "external_player_id": pid,
                })

            teams.append({
                "team_key": team_id,
                "external_team_id": team_id,
                "team_name": team_name,
                "owner_name": owner_name,
                "wins": record.get("wins", 0),
                "losses": record.get("losses", 0),
                "ties": record.get("ties", 0),
                "points_for": record.get("pointsFor", 0.0),
                "points_against": record.get("pointsAgainst", 0.0),
                "standing_rank": t.get("playoffSeed"),
                "playoff_seed": t.get("playoffSeed"),
                "roster": roster,
            })

        return teams


# Auto-register with the factory
CollectorFactory.register("espn_fantasy", ESPNFantasyCollector)
