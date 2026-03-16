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

    BASE_URL = "https://lm-api-reads.fantasy.espn.com/apis/v3/games/ffl"

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
        self._progress(1, 5, {"step": "Connecting to ESPN API"})

        url = (
            f"{self.BASE_URL}/seasons/{self.season}"
            f"/segments/0/leagues/{self.league_id}"
        )

        cookies = self._cookies()
        logger.info(
            "ESPN request: url=%s swid_len=%d espn_s2_len=%d",
            url,
            len(cookies.get("SWID", "")),
            len(cookies.get("espn_s2", "")),
        )

        headers = {
            "Accept": "application/json",
            "X-Fantasy-Source": "kona",
            "X-Fantasy-Platform": "kona-PROD-6daa498ce7a94f1ea2f8607e8b498924c1e58dfe",
        }

        async with httpx.AsyncClient(timeout=30.0, follow_redirects=False) as client:
            resp = await client.get(
                url,
                cookies=cookies,
                headers=headers,
                params={"view": ["mTeam", "mRoster", "mSettings", "mMatchupScore"]},
            )

            logger.info("ESPN response: status=%d url=%s", resp.status_code, resp.url)

            # ESPN returns 302 for private leagues with bad/missing auth
            if resp.status_code == 302:
                location = resp.headers.get("location", "")
                raise RuntimeError(
                    f"ESPN returned 302 redirect to {location}. "
                    "This usually means the league is private and the "
                    "SWID / espn_s2 cookies are missing, expired, or invalid. "
                    "Re-copy them from your browser."
                )

            if resp.status_code == 404:
                raise RuntimeError(
                    f"ESPN returned 404 for league {self.league_id} season {self.season}. "
                    "Check that the league ID and season are correct."
                )

            resp.raise_for_status()
            data: Dict[str, Any] = resp.json()

        self._progress(2, 5, {"step": "Processing league data"})
        league_info = self._extract_league(data)

        self._progress(3, 5, {"step": "Processing teams & rosters"})
        members = {
            m["id"]: m.get("displayName", "Unknown")
            for m in data.get("members", [])
        }
        teams = self._extract_teams(data, members)

        self._progress(4, 5, {"step": "Processing weekly matchups"})
        matchups = self._extract_matchups(data, teams)

        self._progress(5, 5, {"step": "Complete"})

        total_players = sum(len(t.get("roster", [])) for t in teams)
        logger.info(
            "Fetched ESPN league '%s': %d teams, %d roster entries, %d matchup rows",
            league_info.get("league_name", "?"),
            len(teams),
            total_players,
            len(matchups),
        )

        return {
            "league": league_info,
            "teams": teams,
            "matchups": matchups,
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

        # Derive scoring type from schedule settings
        sched = settings.get("scheduleSettings", {})
        # ESPN uses matchupPeriodCount > 0 for head-to-head leagues
        # and periodTypeId to distinguish (1=H2H, 0=roto-ish)
        scoring_settings = settings.get("scoringSettings", {})
        scoring_type_id = scoring_settings.get("scoringType")
        if sched.get("matchupPeriodCount", 0) > 0:
            scoring_type = "head_to_head"
        else:
            scoring_type = "roto"
        # Override if ESPN explicitly sets scoringType
        if scoring_type_id == "H2H_POINTS":
            scoring_type = "head_to_head"
        elif scoring_type_id == "H2H_CATEGORY":
            scoring_type = "head_to_head_category"
        elif scoring_type_id == "TOTAL_POINTS":
            scoring_type = "roto"

        return {
            "external_league_id": str(raw.get("id", self.league_id)),
            "league_name": settings.get("name", f"ESPN League {self.league_id}"),
            "platform": "espn",
            "season": self.season,
            "num_teams": len(raw.get("teams", [])),
            "scoring_type": scoring_type,
            "settings": {
                "playoff_team_count": sched.get("playoffTeamCount"),
                "matchup_period_count": sched.get("matchupPeriodCount"),
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

            # Streak info
            streak = record.get("streakLength", 0)
            streak_type = record.get("streakType", "")
            if isinstance(streak_type, str):
                streak_type = streak_type.lower().replace("win", "win").replace("loss", "loss")

            logo_url = t.get("logo", "")

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
                "standing_rank": None,  # computed below after sorting
                "playoff_seed": t.get("playoffSeed"),
                "logo_url": logo_url,
                "streak_type": streak_type,
                "streak_value": streak,
                "roster": roster,
            })

        # Compute standing_rank from record (wins desc, points_for desc)
        teams.sort(
            key=lambda x: (-x["wins"], x["losses"], -x["points_for"])
        )
        for rank, team in enumerate(teams, start=1):
            team["standing_rank"] = rank

        return teams

    def _extract_matchups(
        self,
        raw: Dict[str, Any],
        teams: List[Dict[str, Any]],
    ) -> List[Dict[str, Any]]:
        """Extract weekly matchup scores from the ESPN schedule data.

        The ``mMatchupScore`` view populates ``raw["schedule"]`` with
        matchup entries.  Each entry has ``matchupPeriodId`` (week),
        ``home`` / ``away`` team dicts with ``teamId`` and
        ``totalPoints``.
        """
        # team_id → team_name lookup
        team_lookup = {t["team_key"]: t["team_name"] for t in teams}

        schedule = raw.get("schedule", [])
        if not isinstance(schedule, list):
            return []

        # Determine playoff start from settings
        settings = raw.get("settings", {})
        sched_settings = settings.get("scheduleSettings", {})
        matchup_periods = sched_settings.get("matchupPeriodCount", 14)
        playoff_start = matchup_periods + 1  # approximate; regular season count

        matchups: List[Dict[str, Any]] = []
        for entry in schedule:
            if not isinstance(entry, dict):
                continue

            week = entry.get("matchupPeriodId", 0)
            matchup_id_raw = entry.get("id", 0)
            is_playoff = entry.get("playoffTierType", "NONE") != "NONE"

            home = entry.get("home", {}) or {}
            away = entry.get("away", {}) or {}

            home_id = str(home.get("teamId", ""))
            away_id = str(away.get("teamId", ""))

            home_pts = float(home.get("totalPoints", 0) or 0)
            away_pts = float(away.get("totalPoints", 0) or 0)

            # Skip unplayed matchups (both 0 usually means not yet played)
            if home_pts == 0 and away_pts == 0:
                # Check if there's a non-zero rosterForCurrentScoringPeriod
                # If truly 0-0, skip it
                if not home.get("rosterForCurrentScoringPeriod"):
                    continue

            # Determine results
            if home_pts > away_pts:
                home_result, away_result = "W", "L"
            elif away_pts > home_pts:
                home_result, away_result = "L", "W"
            else:
                home_result = away_result = "T"

            # Matchup index within the week
            matchup_idx = matchup_id_raw

            if home_id:
                matchups.append({
                    "week": week,
                    "matchup_id": matchup_idx,
                    "team_name": team_lookup.get(home_id, f"Team {home_id}"),
                    "external_team_id": home_id,
                    "points": home_pts,
                    "result": home_result,
                    "is_playoff": is_playoff,
                })

            if away_id:
                matchups.append({
                    "week": week,
                    "matchup_id": matchup_idx,
                    "team_name": team_lookup.get(away_id, f"Team {away_id}"),
                    "external_team_id": away_id,
                    "points": away_pts,
                    "result": away_result,
                    "is_playoff": is_playoff,
                })

        logger.info("Extracted %d matchup rows from ESPN schedule", len(matchups))
        return matchups


# Auto-register with the factory
CollectorFactory.register("espn_fantasy", ESPNFantasyCollector)
