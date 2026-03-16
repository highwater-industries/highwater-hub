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
        """Fetch league details, teams, rosters, and matchups from Yahoo."""
        self._connect()

        self._progress(1, 5, {"step": "Fetching league details"})
        league_data = self._fetch_league()

        self._progress(2, 5, {"step": "Fetching teams"})
        teams_data = self._fetch_teams()

        self._progress(3, 5, {"step": "Fetching rosters"})
        for team in teams_data:
            team["roster"] = self._fetch_roster(team["team_key"])

        self._progress(4, 5, {"step": "Fetching weekly matchups"})
        matchups_data = self._fetch_matchups(teams_data)

        self._progress(5, 5, {"step": "Complete"})

        total_players = sum(len(t.get("roster", [])) for t in teams_data)
        logger.info(
            "Fetched Yahoo league '%s': %d teams, %d roster entries, %d matchup rows",
            league_data.get("league_name", "?"),
            len(teams_data),
            total_players,
            len(matchups_data),
        )

        return {
            "league": league_data,
            "teams": teams_data,
            "matchups": matchups_data,
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

    def _fetch_matchups(self, teams_data: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Return weekly matchup scores for all completed weeks.

        Uses ``self._league.matchups(week=N)`` which returns the raw Yahoo
        scoreboard JSON for a given week.
        """
        # Determine how many weeks to fetch
        end_week = int(self._league.end_week())
        current_week = int(self._league.current_week())
        max_week = min(current_week, end_week)

        matchups: List[Dict[str, Any]] = []
        for week in range(1, max_week + 1):
            try:
                raw = self._league.matchups(week=week)
            except Exception as exc:
                logger.warning("Yahoo matchups week %d failed: %s", week, exc)
                continue

            try:
                scoreboard = raw["fantasy_content"]["league"][1]["scoreboard"]
                matchups_obj = scoreboard["0"]["matchups"]
            except (KeyError, IndexError, TypeError):
                logger.warning("Unexpected scoreboard structure for week %d", week)
                continue

            # matchups_obj is keyed "0", "1", …, "count"
            count = int(matchups_obj.get("count", 0))
            for m_idx in range(count):
                matchup = matchups_obj.get(str(m_idx), {}).get("matchup", {})
                if not matchup:
                    continue

                is_playoff = matchup.get("is_playoffs", "0") == "1"
                is_consolation = matchup.get("is_consolation", "0") == "1"
                winner_key = matchup.get("winner_team_key", "")
                is_tied = matchup.get("is_tied", 0)

                # Teams are in matchup["0"]["teams"]["0"] and ["1"]
                teams_container = matchup.get("0", {}).get("teams", {})

                team_rows = []
                for t_idx in range(2):
                    team_data = teams_container.get(str(t_idx), {}).get("team")
                    if not team_data or not isinstance(team_data, list):
                        continue

                    # team_data[0] is a list of info dicts, team_data[1] has points
                    info_list = team_data[0] if isinstance(team_data[0], list) else []
                    stats = team_data[1] if len(team_data) > 1 else {}

                    team_key = ""
                    team_id = ""
                    team_name = "Unknown"
                    for item in info_list:
                        if isinstance(item, dict):
                            if "team_key" in item:
                                team_key = item["team_key"]
                            if "team_id" in item:
                                team_id = item["team_id"]
                            if "name" in item:
                                team_name = item["name"]

                    points = 0.0
                    if isinstance(stats, dict):
                        tp = stats.get("team_points", {})
                        points = float(tp.get("total", 0) or 0)

                    team_rows.append({
                        "team_key": team_key,
                        "team_id": team_id,
                        "team_name": team_name,
                        "points": points,
                    })

                # Determine results
                for i, row in enumerate(team_rows):
                    if is_tied:
                        result = "T"
                    elif row["team_key"] == winner_key:
                        result = "W"
                    elif winner_key:
                        result = "L"
                    else:
                        result = None

                    matchups.append({
                        "week": week,
                        "matchup_id": m_idx,
                        "team_name": row["team_name"],
                        "external_team_id": row["team_key"],
                        "points": row["points"],
                        "result": result,
                        "is_playoff": is_playoff or is_consolation,
                    })

        logger.info(
            "Fetched %d matchup rows for %d weeks",
            len(matchups), max_week,
        )
        return matchups


# Auto-register with the factory
CollectorFactory.register("yahoo_fantasy", YahooFantasyCollector)
