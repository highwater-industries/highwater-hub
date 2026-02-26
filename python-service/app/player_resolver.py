"""Player name → player_id resolver.

Provides a two-phase lookup for mapping incoming player names to
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
"""

import logging
import re
from typing import Optional

from sqlalchemy import func, select
from sqlalchemy.orm import Session

from app.models.models import PlayerAlias, PlayerDB

logger = logging.getLogger(__name__)


# Common suffixes/prefixes to strip for normalisation
_STRIP_RE = re.compile(
    r"\b(jr\.?|sr\.?|ii|iii|iv|v)\b", re.IGNORECASE
)
_WHITESPACE_RE = re.compile(r"\s+")


def _normalise(name: str) -> str:
    """Lowercase, strip suffixes (Jr, II, III...), collapse whitespace."""
    name = name.lower().strip()
    name = _STRIP_RE.sub("", name)
    name = _WHITESPACE_RE.sub(" ", name).strip()
    # Remove periods from initials: "T.J." -> "tj"
    name = name.replace(".", "")
    return name


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
    ) -> Optional[str]:
        """Try to resolve *name* to a canonical player_id.

        Lookup order:
        1. Memory cache (exact normalised name + source + team)
        2. Memory cache (exact normalised name + source, any team)
        3. Memory cache (exact normalised name, any source/team)
        4. DB: exact player_name match in players table
        5. DB: normalised substring match against players table
        6. Give up → return None
        """
        norm = _normalise(name)

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
        player = self._session.execute(
            select(PlayerDB).where(
                func.lower(PlayerDB.player_name) == name.lower().strip()
            )
        ).scalar_one_or_none()

        # 2b. If no exact match and team is known, try with team filter
        if player is None and team:
            player = self._session.execute(
                select(PlayerDB).where(
                    func.lower(PlayerDB.player_name) == name.lower().strip(),
                    PlayerDB.team == team,
                )
            ).scalar_one_or_none()

        # 2c. Try normalised match (strips Jr/II/III and periods)
        if player is None:
            # Search by ILIKE with the normalised name
            all_candidates = self._session.execute(
                select(PlayerDB).where(
                    PlayerDB.player_name.ilike(f"%{norm}%")
                )
            ).scalars().all()

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
