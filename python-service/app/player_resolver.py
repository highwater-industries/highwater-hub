"""Player name → player_id resolver.

Provides a multi-phase lookup for mapping incoming player names to
canonical player_id values:

1. **Exact alias match** — check player_aliases for (alias_name, source)
   or (alias_name, team).
2. **Fuzzy match** — fall back to Levenshtein-style matching against the
   players table when no alias exists, then auto-create the alias for
   next time.

Usage inside import pipeline::

    resolver = PlayerResolver(session)
    pid = resolver.resolve(name="Pat Mahomes", team="KC", source="fantasypros")
    # Returns "00-0033873" if Patrick Mahomes is in the DB

Fantasy-import usage (with position for disambiguation)::

    resolver = PlayerResolver(session)
    pid = resolver.resolve(
        name="Josh Allen", team="BUF", position="QB", source="yahoo"
    )
"""

import logging
import re
import unicodedata
from dataclasses import dataclass
from typing import Dict, List, Optional, Sequence

from sqlalchemy import func, select
from sqlalchemy.orm import Session

from app.models.models import PlayerAlias, PlayerDB

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Name / team / position normalisation helpers
# ---------------------------------------------------------------------------

# Common suffixes/prefixes to strip for normalisation
_STRIP_RE = re.compile(
    r"\b(jr\.?|sr\.?|ii|iii|iv|v)\b", re.IGNORECASE
)
_WHITESPACE_RE = re.compile(r"\s+")
_PUNCT_TO_SPACE = re.compile(r"[\.'`-]+")

# Team abbreviation → canonical code used by nflreadpy.
# Yahoo and ESPN sometimes use different abbreviations.
_TEAM_MAP: Dict[str, str] = {
    "WSH": "WAS",
    "JAX": "JAC",
    "LA":  "LAR",
    "LV":  "LVR",
    "OAK": "LVR",
    "SD":  "LAC",
    "STL": "LAR",
    "SAINTS": "NO",
    "SAINT":  "NO",
}

# Position label → canonical code.
_POSITION_MAP: Dict[str, str] = {
    "D/ST":    "DST",
    "DEF":     "DST",
    "DEFENSE": "DST",
    "PK":      "K",
}

# Positions that represent team defenses (not individual players).
_DEFENSE_POSITIONS = frozenset({"DST", "DEF", "D/ST", "DEFENSE"})


def _normalise(name: str) -> str:
    """Lowercase, strip accents/suffixes (Jr, II, III...), collapse whitespace."""
    name = name.lower().strip()
    # Strip Unicode accents (é → e, etc.)
    name = unicodedata.normalize("NFKD", name)
    name = "".join(c for c in name if not unicodedata.combining(c))
    name = _STRIP_RE.sub("", name)
    name = _WHITESPACE_RE.sub(" ", name).strip()
    # Remove periods from initials: "T.J." -> "tj"
    name = name.replace(".", "")
    return name


def normalize_team(team: Optional[str]) -> str:
    """Normalize a team abbreviation to the canonical code used in the DB.

    Maps known variants (WSH→WAS, JAX→JAC, OAK→LVR, etc.) and uppercases.
    Returns empty string for None/empty input.
    """
    if not team:
        return ""
    code = str(team).strip().upper()
    return _TEAM_MAP.get(code, code)


def normalize_position(position: Optional[str]) -> str:
    """Normalize a position label to the canonical code used in the DB.

    Maps D/ST→DST, DEF→DST, PK→K, etc.  Returns empty string for None.
    """
    if not position:
        return ""
    code = str(position).strip().upper()
    return _POSITION_MAP.get(code, code)


def is_team_defense(position: Optional[str]) -> bool:
    """Return True if *position* represents a team defense (not an individual player).

    Team defenses are not stored in the players table, so fantasy import
    pipelines should skip them rather than trying to resolve.
    """
    if not position:
        return False
    return str(position).strip().upper() in _DEFENSE_POSITIONS


@dataclass
class ResolveResult:
    """Result of resolving a single player from a fantasy roster."""

    player_id: Optional[str]
    matched: bool
    player_name: str          # as-reported by the platform
    position: str             # normalised
    team: str                 # normalised
    skipped_defense: bool = False


class PlayerResolver:
    """Resolves display names to canonical player_ids.

    Create one per import session — it caches results in memory so the
    same name isn't looked up twice.
    """

    def __init__(self, session: Session) -> None:
        self._session = session
        # In-memory cache: (normalised_name, source|None, team|None) → player_id | None
        self._cache: dict[tuple[str, str | None, str | None], str | None] = {}
        # Pre-load all aliases into cache
        self._load_aliases()

    def _load_aliases(self) -> None:
        """Bulk-load all aliases into the in-memory cache."""
        rows = self._session.execute(select(PlayerAlias)).scalars().all()
        for alias in rows:
            key = (_normalise(alias.alias_name), alias.source, alias.team)
            self._cache[key] = alias.player_id
        logger.debug("Loaded %d player aliases into resolver cache", len(rows))

    def resolve(
        self,
        name: str,
        team: str | None = None,
        source: str | None = None,
        position: str | None = None,
    ) -> Optional[str]:
        """Try to resolve *name* to a canonical player_id.

        Args:
            name: Player display name as reported by the source.
            team: NFL team abbreviation (will be normalised internally).
            source: Data source identifier (e.g. ``"yahoo"``, ``"espn"``).
            position: Player position (e.g. ``"QB"``, ``"D/ST"``).  Used to
                disambiguate when multiple players share the same name.

        Lookup order:
        1. Memory cache (exact normalised name + source + team)
        2. Memory cache (exact normalised name + source, any team)
        3. Memory cache (exact normalised name, any source/team)
        4. DB: exact player_name match in players table (narrow by team/position)
        5. DB: normalised substring match against players table
        6. Give up → return None
        """
        # Normalise inputs
        norm = _normalise(name)
        team = normalize_team(team) if team else team
        pos = normalize_position(position) if position else None

        # --- Phase 1: cache lookups (most specific → least) ---
        for try_src, try_team in [
            (source, team),
            (source, None),
            (None, None),
        ]:
            key = (norm, try_src, try_team)
            if key in self._cache:
                return self._cache[key]

        # --- Phase 2: DB lookup on players table ---
        # 2a. Case-insensitive exact match on player_name
        #     Use scalars().all() to handle duplicate names (e.g. "Josh Allen" QB & DE)
        candidates = self._session.execute(
            select(PlayerDB).where(
                func.lower(PlayerDB.player_name) == name.lower().strip()
            )
        ).scalars().all()

        player = None
        if len(candidates) == 1:
            player = candidates[0]
        elif len(candidates) > 1:
            # Multiple matches — narrow by position first, then team
            if pos:
                pos_matches = [p for p in candidates if p.player_position == pos]
                if len(pos_matches) == 1:
                    player = pos_matches[0]
                elif len(pos_matches) > 1 and team:
                    team_matches = [p for p in pos_matches if p.team == team]
                    if len(team_matches) == 1:
                        player = team_matches[0]
            if player is None and team:
                team_matches = [p for p in candidates if p.team == team]
                if len(team_matches) == 1:
                    player = team_matches[0]

        # 2b. Try normalised/ILIKE match (strips Jr/II/III and periods)
        if player is None:
            # Search by ILIKE with the normalised name
            all_candidates = self._session.execute(
                select(PlayerDB).where(
                    PlayerDB.player_name.ilike(f"%{norm}%")
                )
            ).scalars().all()

            # Filter by position first if available
            if pos:
                pos_matches = [p for p in all_candidates if p.player_position == pos]
                if pos_matches:
                    all_candidates = pos_matches

            if team:
                team_matches = [p for p in all_candidates if p.team == team]
                if len(team_matches) == 1:
                    player = team_matches[0]
                elif len(all_candidates) == 1:
                    player = all_candidates[0]
            elif len(all_candidates) == 1:
                player = all_candidates[0]

        if player is None or player.player_id is None:
            # Cache the miss so we don't retry
            self._cache[(norm, source, team)] = None
            return None

        pid = player.player_id

        # --- Phase 3: Create alias for next time ---
        # Only create alias if name is different from the canonical name
        canon_norm = _normalise(player.player_name)
        if norm != canon_norm:
            try:
                alias = PlayerAlias(
                    player_id=pid,
                    alias_name=name.strip(),
                    team=team,
                    source=source,
                    auto_matched=True,
                )
                self._session.add(alias)
                self._session.flush()
                logger.info(
                    "Auto-created alias: '%s' → %s (%s/%s)",
                    name.strip(), pid, source, team,
                )
            except Exception:
                # Unique constraint violation — alias already exists
                self._session.rollback()
                logger.debug("Alias already exists for '%s'", name.strip())

        # Cache the hit
        self._cache[(norm, source, team)] = pid
        return pid

    def seed_from_roster(self) -> int:
        """Seed aliases from the players table — canonical name → player_id.

        This ensures every player in the roster has at least one alias entry.
        Returns the number of aliases created.
        """
        created = 0
        players = self._session.execute(
            select(PlayerDB).where(PlayerDB.player_id.isnot(None))
        ).scalars().all()

        existing_pids = set(
            self._session.execute(
                select(PlayerAlias.player_id)
            ).scalars().all()
        )

        for p in players:
            if p.player_id in existing_pids:
                continue
            try:
                alias = PlayerAlias(
                    player_id=p.player_id,
                    alias_name=p.player_name,
                    team=p.team,
                    source=p.source,
                    auto_matched=False,
                )
                self._session.add(alias)
                created += 1
            except Exception:
                self._session.rollback()

        if created:
            self._session.commit()
            logger.info("Seeded %d canonical aliases from roster", created)

        return created

    # ------------------------------------------------------------------
    # Fantasy roster helpers
    # ------------------------------------------------------------------

    def resolve_roster(
        self,
        players: Sequence[Dict[str, str]],
        source: str,
    ) -> List[ResolveResult]:
        """Resolve a list of fantasy roster entries in bulk.

        Each dict in *players* should have at minimum ``player_name`` and
        ``player_position``, and optionally ``team``.

        Team defenses are automatically detected and skipped (they aren't
        in the master player list).

        Args:
            players: Sequence of dicts with player_name, player_position,
                and optionally team.
            source: Platform source identifier (``"yahoo"`` / ``"espn"``).

        Returns:
            List of :class:`ResolveResult` in the same order as *players*.
        """
        results: List[ResolveResult] = []

        for entry in players:
            raw_name = entry.get("player_name", "")
            raw_pos = entry.get("player_position", "") or entry.get("position", "")
            raw_team = entry.get("team", "")

            norm_pos = normalize_position(raw_pos)
            norm_team = normalize_team(raw_team)

            # Skip team defenses — they don't exist in the player DB
            if is_team_defense(raw_pos):
                results.append(ResolveResult(
                    player_id=None,
                    matched=False,
                    player_name=raw_name,
                    position=norm_pos,
                    team=norm_team,
                    skipped_defense=True,
                ))
                logger.debug("Skipping team defense: %s", raw_name)
                continue

            pid = self.resolve(
                name=raw_name,
                team=raw_team,    # resolve() normalises internally
                source=source,
                position=raw_pos,  # resolve() normalises internally
            )

            results.append(ResolveResult(
                player_id=pid,
                matched=pid is not None,
                player_name=raw_name,
                position=norm_pos,
                team=norm_team,
            ))

            if pid is None:
                logger.warning(
                    "Unmatched player from %s: %s (%s, %s)",
                    source, raw_name, norm_pos, norm_team,
                )

        matched = sum(1 for r in results if r.matched)
        skipped = sum(1 for r in results if r.skipped_defense)
        unmatched = len(results) - matched - skipped
        logger.info(
            "Roster resolve: %d matched, %d unmatched, %d defenses skipped "
            "(source=%s)",
            matched, unmatched, skipped, source,
        )

        return results
