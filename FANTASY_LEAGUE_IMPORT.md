# Fantasy League Import — Feature Plan

> Tracking document for porting Yahoo/ESPN fantasy league imports from the old
> all-Python project (`oldffmgr/ff/`) into the new Go+Python architecture.

## Status: Phase 1 — Implementation Complete (pending docker build + e2e test)

---

## Architecture Overview

```
SvelteKit UI (/leagues page)
    │
    ▼
Go API  POST /api/fantasy/import
    │   GET  /api/fantasy/leagues
    │   GET  /api/fantasy/leagues/:id
    │   GET  /api/fantasy/leagues/:id/teams
    │   GET  /api/fantasy/leagues/:id/teams/:teamId/roster
    │
    ├── jobs.Client → POST python-service /api/v1/fantasy/import
    │                  (triggers Celery task)
    │
    ▼
Go reads from Postgres         Python writes to Postgres
(fantasy_leagues,              (via Celery worker running
 fantasy_teams,                 yahoo/espn collectors)
 fantasy_rosters,
 fantasy_league_players)
```

### Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Route approach | **B — New `/api/v1/fantasy/` route** | Fantasy import params (OAuth, cookies, league_id) differ significantly from nflstats |
| Job tracking | **Shared `collection_history` table** | Reuse existing `/api/jobs/*` list/summary/abort endpoints |
| Yahoo auth (v1) | **File-based `oauth2.json`** | Keep it behind-the-scenes; no OAuth UI flow yet |
| ESPN auth | **User pastes SWID + espn_s2** | Session cookies provided via import form, not stored in DB |
| Player matching | **Reuse existing `PlayerResolver`** | Already has 3-phase lookup (cache → alias → fuzzy DB match) |
| New player creation | **Never from fantasy imports** | NFL collector owns the master player list; unmatched players are logged |
| DEF/D/ST handling | **Skipped** | Team defenses are not in the player database |

---

## Phase 1 — Core Import Pipeline + Basic UI

### 1.1 Database Tables (Python side — SQLAlchemy models)

Add to `python-service/app/models/models.py`:

#### `fantasy_leagues`
| Column | Type | Notes |
|---|---|---|
| `id` | int PK | auto |
| `external_league_id` | varchar(100) | Yahoo/ESPN league ID |
| `league_name` | varchar(255) | |
| `platform` | varchar(50) | "Yahoo" or "ESPN" |
| `season` | int | NFL season year |
| `num_teams` | int | nullable |
| `scoring_type` | varchar(50) | nullable (e.g. "head_to_head") |
| `settings` | JSONB | Full settings blob from platform |
| `created_at` | timestamptz | |
| `updated_at` | timestamptz | |

Unique constraint: `(external_league_id, platform, season)`

#### `fantasy_teams`
| Column | Type | Notes |
|---|---|---|
| `id` | int PK | auto |
| `league_id` | int FK → fantasy_leagues.id | ON DELETE CASCADE |
| `external_team_id` | varchar(100) | nullable, platform team key |
| `team_name` | varchar(255) | |
| `owner_name` | varchar(255) | nullable |
| `wins` | int | default 0 |
| `losses` | int | default 0 |
| `ties` | int | default 0 |
| `points_for` | float | default 0.0 |
| `points_against` | float | default 0.0 |
| `standing_rank` | int | nullable |
| `playoff_seed` | int | nullable |
| `created_at` | timestamptz | |
| `updated_at` | timestamptz | |

#### `fantasy_rosters`
| Column | Type | Notes |
|---|---|---|
| `id` | int PK | auto |
| `team_id` | int FK → fantasy_teams.id | ON DELETE CASCADE |
| `player_id` | varchar(64) | nullable FK-ish to players.player_id (nullable for unmatched) |
| `player_name` | varchar(255) | as-reported by platform (always stored) |
| `player_position` | varchar(50) | |
| `nfl_team` | varchar(8) | as-reported by platform |
| `roster_position` | varchar(50) | e.g. QB, RB, BN, FLEX, IR |
| `external_player_id` | varchar(100) | Yahoo/ESPN player ID |
| `matched` | bool | whether player_id was resolved |
| `created_at` | timestamptz | |

Unique constraint: `(team_id, external_player_id)` or `(team_id, player_name, player_position)`

#### `fantasy_league_players` (Phase 1 — optional, lower priority)
All players in league context (owned + free agents) with ownership stats.
May defer to Phase 2 if scope needs trimming.

#### `fantasy_transactions` (Phase 2)
Transaction history — deferred.

### 1.2 Python Collectors

#### `python-service/app/data_collectors/yahoo_fantasy_collector.py`
- Port from `oldffmgr/ff/data_collectors/yahoo_fantasy_collector.py`
- Uses `yahoo_oauth` + `yahoo_fantasy_api` libraries
- Reads `oauth2.json` from filesystem (mounted via docker volume or baked in)
- Fetches: league details → teams → rosters (per-team)
- For v1, skip free agents and transactions (keep the code paths but don't import)
- Returns structured dict: `{ league: {}, teams: [{ roster: [] }] }`
- Register as `"yahoo_fantasy"` in `CollectorFactory`

#### `python-service/app/data_collectors/espn_fantasy_collector.py`
- Port from `oldffmgr/ff/data_collectors/espn_fantasy_collector.py`
- Uses `httpx` with cookie-based auth (SWID + espn_s2)
- Hits ESPN API: `https://fantasy.espn.com/apis/v3/games/ffl/seasons/{season}/segments/0/leagues/{league_id}`
- With views: `mTeam`, `mRoster`, `mSettings`
- Maps ESPN numeric IDs → positions and NFL teams
- Returns same structured dict format as Yahoo
- Register as `"espn_fantasy"` in `CollectorFactory`

### 1.3 Python Import Task

#### `python-service/app/tasks/fantasy_import_task.py`
New Celery task: `run_fantasy_import`
- Creates `CollectionHistory` row (status=running)
- Instantiates Yahoo or ESPN collector based on `collector_type`
- Calls collector `.fetch()`
- Persists league → teams → rosters using `PlayerResolver` for matching
- Updates `CollectionHistory` with results (records_fetched, inserted, etc.)
- Handles errors gracefully — partial imports are OK (some players unmatched)

#### `python-service/app/routes/fantasy.py`
New FastAPI router mounted at `/api/v1/fantasy`:
- `POST /import` — accepts `{ platform, league_id, season, espn_swid?, espn_s2? }`, dispatches Celery task
- `GET /jobs/{job_id}` — reuse pattern from nflstats (or just reuse the existing jobs endpoint)

### 1.4 Python Dependencies

Add to `pyproject.toml`:
- `yahoo-oauth` — Yahoo OAuth2 client
- `yahoo-fantasy-api` — Yahoo Fantasy API wrapper
- `httpx` — async HTTP client (for ESPN API calls)

### 1.5 Go API Layer

#### `internal/fantasy/model.go`
Go structs for API responses:
```go
type League struct {
    ID               int             `json:"id"`
    ExternalLeagueID string          `json:"external_league_id"`
    LeagueName       string          `json:"league_name"`
    Platform         string          `json:"platform"`
    Season           int             `json:"season"`
    NumTeams         *int            `json:"num_teams,omitempty"`
    ScoringType      *string         `json:"scoring_type,omitempty"`
    Settings         json.RawMessage `json:"settings,omitempty"`
    CreatedAt        time.Time       `json:"created_at"`
    UpdatedAt        time.Time       `json:"updated_at"`
}

type Team struct {
    ID             int      `json:"id"`
    LeagueID       int      `json:"league_id"`
    ExternalTeamID *string  `json:"external_team_id,omitempty"`
    TeamName       string   `json:"team_name"`
    OwnerName      *string  `json:"owner_name,omitempty"`
    Wins           int      `json:"wins"`
    Losses         int      `json:"losses"`
    Ties           int      `json:"ties"`
    PointsFor      float64  `json:"points_for"`
    PointsAgainst  float64  `json:"points_against"`
    StandingRank   *int     `json:"standing_rank,omitempty"`
    PlayoffSeed    *int     `json:"playoff_seed,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}

type RosterEntry struct {
    ID               int     `json:"id"`
    TeamID           int     `json:"team_id"`
    PlayerID         *string `json:"player_id,omitempty"`
    PlayerName       string  `json:"player_name"`
    PlayerPosition   string  `json:"player_position"`
    NFLTeam          string  `json:"nfl_team"`
    RosterPosition   string  `json:"roster_position"`
    ExternalPlayerID *string `json:"external_player_id,omitempty"`
    Matched          bool    `json:"matched"`
}

type ImportRequest struct {
    Platform string  `json:"platform"`     // "yahoo" or "espn"
    LeagueID string  `json:"league_id"`
    Season   int     `json:"season"`
    ESPNSWID *string `json:"espn_swid,omitempty"`
    ESPNS2   *string `json:"espn_s2,omitempty"`
}
```

#### `internal/fantasy/store.go`
Read-only store interface:
```go
type Store interface {
    ListLeagues(ctx context.Context, filter LeagueFilter) ([]League, int, error)
    GetLeague(ctx context.Context, id int) (*League, error)
    ListTeams(ctx context.Context, leagueID int) ([]Team, error)
    GetRoster(ctx context.Context, teamID int) ([]RosterEntry, error)
}
```

#### `internal/fantasy/postgres_store.go`
PostgreSQL implementation reading from `fantasy_*` tables.

#### `internal/fantasy/handler.go`
HTTP handlers:
- `HandleImportLeague` — POST /api/fantasy/import → calls Python service via `jobs.Client`
- `HandleListLeagues` — GET /api/fantasy/leagues
- `HandleGetLeague` — GET /api/fantasy/leagues/:id
- `HandleListTeams` — GET /api/fantasy/leagues/:id/teams
- `HandleGetRoster` — GET /api/fantasy/leagues/:id/teams/:teamId/roster

#### `internal/fantasy/client.go`
Extension of the jobs client pattern — calls Python's `/api/v1/fantasy/import`.

#### Route registration in `internal/server/routes.go`
```go
// Fantasy league management
r.Post("/api/fantasy/import", fantasyHandler.HandleImportLeague)
r.Get("/api/fantasy/leagues", fantasyHandler.HandleListLeagues)
r.Get("/api/fantasy/leagues/{id}", fantasyHandler.HandleGetLeague)
r.Get("/api/fantasy/leagues/{id}/teams", fantasyHandler.HandleListTeams)
r.Get("/api/fantasy/leagues/{id}/teams/{teamId}/roster", fantasyHandler.HandleGetRoster)
```

### 1.6 SvelteKit UI

#### `/leagues` page
- **Import panel** at top: platform select (Yahoo/ESPN), league ID input, season input
  - ESPN mode shows additional fields: SWID, espn_s2
  - Submit triggers POST /api/fantasy/import → shows job ID with link to jobs page
- **League list** below: cards showing imported leagues
  - Each card: league name, platform badge, season, team count, import date
  - Click → expands to show teams as a standings table (rank, name, owner, W-L-T, PF, PA)
  - Click team → shows roster as a table (player, position, NFL team, roster slot, match status)
  - Unmatched players highlighted in amber/warning color

#### Navigation
- Add "Leagues" to the sidebar/nav alongside existing Players, Stats, Games, etc.

---

## Phase 2 — Future Enhancements

### Yahoo Projections Collector
- `yahoo_projections` collector type
- Fetches projected stats from Yahoo Fantasy API
- Stores into `player_stats` table with `stat_type='projected'` and `source='yahoo'`
- Separate job type, separate import trigger

### Full OAuth Browser Flow (Maybe)
- `/api/fantasy/auth/yahoo/start` → redirect to Yahoo OAuth consent
- `/api/fantasy/auth/yahoo/callback` → exchange code for tokens
- Store encrypted tokens in DB (like old project's `OAuthToken` model)
- UI: "Connect Yahoo Account" button
- **May choose to keep this server-side only** — TBD

### Fantasy League Players (Free Agents)
- Import waiver wire / free agent lists with ownership %
- `fantasy_league_players` table with ownership_status, percent_owned, etc.

### Transaction History
- Import add/drop/trade history
- `fantasy_transactions` table with JSONB players_data

### Richer UI
- Roster comparison views
- Player value / trade analyzer
- Waiver wire recommendations
- Draft board integration

---

## File Map — What Gets Created/Modified

### New Files
```
python-service/app/data_collectors/yahoo_fantasy_collector.py
python-service/app/data_collectors/espn_fantasy_collector.py
python-service/app/tasks/fantasy_import_task.py
python-service/app/routes/fantasy.py
python-service/app/schemas/fantasy.py
internal/fantasy/model.go
internal/fantasy/store.go
internal/fantasy/postgres_store.go
internal/fantasy/handler.go
internal/fantasy/client.go
web/src/routes/leagues/+page.svelte
web/src/routes/leagues/[id]/+page.svelte
```

### Modified Files
```
python-service/app/models/models.py          — add FantasyLeague, FantasyTeam, FantasyRoster models
python-service/app/player_resolver.py        — add normalize_team/position, resolve_roster batch, ResolveResult
python-service/app/data_collectors/__init__.py — import/export new collectors
python-service/app/main.py                   — mount fantasy router
python-service/pyproject.toml                — add yahoo-oauth, yahoo-fantasy-api, httpx deps
internal/server/routes.go                    — register fantasy API routes
internal/server/server.go                    — add FantasyStore + FantasyClient to Config
cmd/server/main.go                           — instantiate fantasy store + client
web/src/lib/api.ts                           — fantasy types + API functions
web/src/routes/menu.ts                       — add Fantasy section + Leagues link
```

---

## Player Matching Flow

```
Platform data (Yahoo/ESPN)
    │
    │  player_name = "Pat Mahomes"
    │  team = "KC", position = "QB"
    │
    ▼
PlayerResolver.resolve(name, team, source)
    │
    ├─ 1. Cache hit? → return player_id
    ├─ 2. Alias table match? → return player_id
    ├─ 3. DB exact name match → disambiguate by team → return player_id
    ├─ 4. DB fuzzy/ILIKE match → return player_id, auto-create alias
    └─ 5. No match → return None, log warning
           │
           ▼
     Store in fantasy_rosters with:
       player_id = NULL
       matched = false
       player_name = "Pat Mahomes" (always stored)
```

Unmatched players are visible in the UI (amber highlight) and can be
manually resolved later by adding aliases or re-running after the NFL
collector populates the master list.

---

## Notes

- Yahoo's `oauth2.json` file needs to be accessible to the Celery worker container
  (mount via docker volume or copy during build)
- ESPN cookies are ephemeral — user must re-provide them if they expire
- The `collection_history.collector_type` values will be `"yahoo_fantasy"` and
  `"espn_fantasy"` — these show up in the existing Jobs page automatically
- Fantasy rosters always store the platform-reported player name regardless of
  match status, so we never lose data
