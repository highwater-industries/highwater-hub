"""Yahoo Fantasy Football league data collector.

Fetches league metadata, teams, and rosters from the Yahoo Fantasy API
using OAuth2 credentials stored in an ``oauth2.json`` file.

Registers as ``"yahoo_fantasy"`` in the :class:`CollectorFactory`.

Dependencies:
    pip install yahoo-oauth yahoo-fantasy-api
"""

from typing import Any, Callable, Dict, List, Optional

import logging

from .base import DataCollector
from .factory import CollectorFactory

logger = logging.getLogger(__name__)

# Defer import so the service still boots when the libs aren't installed.
try:
    from yahoo_oauth import OAuth2
    import yahoo_fantasy_api as yfa  # type: ignore[import-untyped]

    YAHOO_AVAILABLE = True
except ImportError:
    YAHOO_AVAILABLE = False


class YahooFantasyCollector(DataCollector):
    """Collect league, team, and roster data from Yahoo Fantasy Football.

    Requires an ``oauth2.json`` file in the working directory (or a path
    supplied via *oauth_json_path*) containing Yahoo API credentials::

        {
            "consumer_key": "<your-key>",
            "consumer_secret": "<your-secret>"
        }
    """

    def __init__(
        self,
        league_id: str,
        season: int,
        oauth_json_path: str = "oauth2.json",
        progress_callback: Optional[Callable[[int, int, Dict[str, Any]], None]] = None,
        **_kwargs: Any,
    ):
        super().__init__(name="Yahoo Fantasy")
        if not YAHOO_AVAILABLE:
            raise ImportError(
                "yahoo_fantasy_api is not installed. "
                "Install with: pip install yahoo-oauth yahoo-fantasy-api"
            )

        self.league_id = league_id
        self.season = season
        self.oauth_json_path = oauth_json_path
        self.progress_callback = progress_callback
        self._oauth: Any = None
        self._game: Any = None
        self._league: Any = None

    # ------------------------------------------------------------------
    # Connection helpers
    # ------------------------------------------------------------------

    def _connect(self) -> None:
        """Initialise OAuth and the Yahoo Fantasy API objects."""
        if self._league is not None:
            return

        self._oauth = OAuth2(None, None, from_file=self.oauth_json_path)
        if not self._oauth.token_is_valid():
            self._oauth.refresh_access_token()

        self._game = yfa.Game(self._oauth, "nfl")

        # Accept formats:
        #   "158244"           → current-season shorthand "nfl.l.158244"
        #   "nfl.l.158244"     → current-season shorthand (as-is)
        #   "461.l.158244"     → historical game-key-prefixed (as-is)
        if "." in self.league_id:
            league_key = self.league_id           # already qualified
        else:
            league_key = f"nfl.l.{self.league_id}"
        self._league = self._game.to_league(league_key)
        logger.info("Connected to Yahoo league: %s", league_key)

    def _progress(self, current: int, total: int, info: Dict[str, Any]) -> None:
        if self.progress_callback:
            self.progress_callback(current, total, info)

    # ------------------------------------------------------------------
    # DataCollector interface
    # ------------------------------------------------------------------

    async def fetch(self) -> Dict[str, Any]:
        """Fetch league details, teams, and rosters from Yahoo."""
        self._connect()

        self._progress(1, 4, {"step": "Fetching league details"})
        league_data = self._fetch_league()

        self._progress(2, 4, {"step": "Fetching teams"})
        teams_data = self._fetch_teams()

        self._progress(3, 4, {"step": "Fetching rosters"})
        for team in teams_data:
            team["roster"] = self._fetch_roster(team["team_key"])

        self._progress(4, 4, {"step": "Complete"})

        total_players = sum(len(t.get("roster", [])) for t in teams_data)
        logger.info(
            "Fetched Yahoo league '%s': %d teams, %d roster entries",
            league_data.get("league_name", "?"),
            len(teams_data),
            total_players,
        )

        return {
            "league": league_data,
            "teams": teams_data,
            "platform": "yahoo",
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
        """No additional transformation needed — data is already structured."""
        return data

    # ------------------------------------------------------------------
    # Fetch helpers
    # ------------------------------------------------------------------

    def _fetch_league(self) -> Dict[str, Any]:
        """Return league metadata."""
        settings_raw = self._league.settings()
        settings: dict = settings_raw if isinstance(settings_raw, dict) else {}

        metadata: Dict[str, Any] = {}
        for attr_name in ("league_info", "meta"):
            fn = getattr(self._league, attr_name, None)
            if callable(fn):
                try:
                    result = fn()
                    if isinstance(result, dict):
                        metadata = result
                        break
                except Exception:
                    pass

        league_name = (
            metadata.get("name")
            or settings.get("name")
            or settings.get("league_name")
            or f"Yahoo League {self.league_id}"
        )

        # Extract bare league number from any key format
        # e.g. "390.l.113462" → "113462", "nfl.l.158244" → "158244"
        bare_id = self.league_id.rsplit(".", 1)[-1] if "." in self.league_id else self.league_id

        return {
            "external_league_id": bare_id,
            "league_name": league_name,
            "platform": "yahoo",
            "season": self.season,
            "num_teams": metadata.get("num_teams") or settings.get("num_teams"),
            "scoring_type": {
                "head": "head_to_head",
                "roto": "roto",
            }.get(settings.get("scoring_type", ""), settings.get("scoring_type")),
            "settings": {
                "roster_positions": settings.get("roster_positions", []),
                "stat_categories": settings.get("stat_categories", {}),
                "draft_status": metadata.get("draft_status") or settings.get("draft_status"),
                "current_week": metadata.get("current_week") or settings.get("current_week"),
            },
        }

    def _fetch_teams(self) -> List[Dict[str, Any]]:
        """Return all teams with standing data.

        Yahoo's ``teams()`` returns basic info (name, managers, logo)
        while ``standings()`` has records, points, rank, and streak.
        We merge both by ``team_key``.
        """
        teams_raw = self._league.teams()
        teams: dict = teams_raw if isinstance(teams_raw, dict) else {}

        # Build standings lookup keyed by team_key
        standings_map: Dict[str, dict] = {}
        try:
            standings_raw = self._league.standings()
            for entry in (standings_raw if isinstance(standings_raw, list) else []):
                if isinstance(entry, dict) and "team_key" in entry:
                    standings_map[entry["team_key"]] = entry
        except Exception:
            logger.warning("Could not fetch Yahoo standings — records/points will be 0")

        result: List[Dict[str, Any]] = []
        for team_key, info in teams.items():
            if not isinstance(info, dict):
                continue

            # --- Owner name ---
            # teams() returns managers as a list of {manager: {nickname, ...}}
            owner_name = "Unknown"
            managers = info.get("managers")
            if isinstance(managers, list) and managers:
                first = managers[0]
                if isinstance(first, dict):
                    mgr = first.get("manager", first)
                    owner_name = mgr.get("nickname", "Unknown") if isinstance(mgr, dict) else "Unknown"
            elif isinstance(info.get("manager"), dict):
                owner_name = info["manager"].get("nickname", "Unknown")

            # --- Standings from standings() ---
            st = standings_map.get(team_key, {})
            outcomes = st.get("outcome_totals", {})
            if not isinstance(outcomes, dict):
                outcomes = {}

            # points_for/against can be top-level strings or nested dicts
            pf_raw = st.get("points_for", 0)
            pa_raw = st.get("points_against", 0)
            points_for = float(pf_raw) if not isinstance(pf_raw, dict) else float(pf_raw.get("total", 0))
            points_against = float(pa_raw) if not isinstance(pa_raw, dict) else float(pa_raw.get("total", 0))

            # Streak info
            streak = st.get("streak", {})
            streak_type = streak.get("type", "") if isinstance(streak, dict) else ""
            streak_value = int(streak.get("value", 0)) if isinstance(streak, dict) else 0

            # Team logo
            logo_url = ""
            logos = info.get("team_logos", [])
            if isinstance(logos, list) and logos:
                logo_entry = logos[0] if isinstance(logos[0], dict) else {}
                tl = logo_entry.get("team_logo", {})
                logo_url = tl.get("url", "") if isinstance(tl, dict) else ""

            result.append({
                "team_key": team_key,
                "external_team_id": team_key,
                "team_name": info.get("name", "Unknown Team"),
                "owner_name": owner_name,
                "wins": int(outcomes.get("wins", 0) or 0),
                "losses": int(outcomes.get("losses", 0) or 0),
                "ties": int(outcomes.get("ties", 0) or 0),
                "points_for": points_for,
                "points_against": points_against,
                "standing_rank": int(st["rank"]) if st.get("rank") else None,
                "playoff_seed": (
                    int(st["playoff_seed"])
                    if st.get("playoff_seed")
                    else None
                ),
                "streak_type": streak_type,
                "streak_value": streak_value,
                "logo_url": logo_url,
                "waiver_priority": int(info.get("waiver_priority", 0) or 0),
                "number_of_moves": int(info.get("number_of_moves", 0) or 0),
                "number_of_trades": int(info.get("number_of_trades", 0) or 0),
                "clinched_playoffs": bool(info.get("clinched_playoffs")),
                "draft_grade": info.get("draft_grade", ""),
            })

        return result

    def _fetch_roster(self, team_key: str) -> List[Dict[str, Any]]:
        """Return roster entries for a single team."""
        team_obj = self._league.to_team(team_key)
        roster_raw = team_obj.roster()
        roster: list = roster_raw if isinstance(roster_raw, list) else []

        players: List[Dict[str, Any]] = []
        for p in roster:
            if not isinstance(p, dict):
                continue

            sel_pos = p.get("selected_position", {})
            sel_pos = sel_pos if isinstance(sel_pos, dict) else {}

            eligible = p.get("eligible_positions")
            position = (
                p.get("display_position")
                or p.get("primary_position")
                or (eligible[0] if isinstance(eligible, list) and eligible else "")
                or p.get("position_type", "")
            )

            players.append({
                "player_name": p.get("name", "Unknown"),
                "player_position": position,
                "team": p.get("editorial_team_abbr", ""),
                "roster_position": sel_pos.get("position", "BN"),
                "external_player_id": str(p.get("player_id", "")),
                "status": p.get("status", ""),
            })

        return players


# Auto-register with the factory
CollectorFactory.register("yahoo_fantasy", YahooFantasyCollector)
