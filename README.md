# ffredux

A multi-service fantasy football platform built with Go, Python, and SvelteKit.

## Architecture

| Service | Language | Port | Description |
|---------|----------|------|-------------|
| Go Server | Go | 3141 | Primary API + embedded SvelteKit UI |
| Python Service | Python (FastAPI) | 3142 | Data service — NFL stats import via nflreadpy, async job processing |
| Celery Worker | Python (Celery) | — | Background task runner for long-running imports |
| Postgres | — | 5432 | Shared database for domain data, stats, and job state |
| RabbitMQ | — | 5672 / 15672 | Message broker for async task dispatch (Celery) |

The Go server is the primary API and serves the SvelteKit SPA (embedded in the binary via `embed.FS`). It calls the Python service over HTTP for data-heavy operations. Both services read/write to the same Postgres database. Long-running imports are dispatched asynchronously via Celery + RabbitMQ, with job status tracked in Postgres.

## Project Structure

```
ffredux/
├── cmd/
│   ├── server/              # Go HTTP server entry point
│   └── cli/                 # Go CLI tool entry point
├── internal/
│   ├── server/              # HTTP server, routes, middleware
│   ├── frontend/            # Embedded SvelteKit SPA (build output)
│   ├── jobs/                # Import job management (client, handlers, store)
│   ├── nflstats/            # NFL data queries (players, stats, games, rankings)
│   ├── user/                # User domain model
│   └── httputil/            # HTTP response helpers
├── web/                     # SvelteKit frontend source
│   ├── src/
│   │   ├── routes/          # Pages: Home Base, Players, Jobs
│   │   └── lib/
│   │       ├── api.ts       # Typed API client
│   │       └── constants.ts # Teams, positions, collector types
│   ├── svelte.config.js     # adapter-static (SPA mode)
│   └── vite.config.ts       # Dev proxy to Go server
├── python-service/
│   ├── app/
│   │   ├── main.py          # FastAPI entry point
│   │   ├── celery_app.py    # Celery configuration
│   │   ├── tasks/           # Async import tasks (dispatch + persistence)
│   │   ├── data_collectors/ # Data source adapters (rosters, stats, schedules, rankings)
│   │   ├── routes/          # API route handlers
│   │   ├── models/          # SQLAlchemy models
│   │   ├── schemas/         # Pydantic request/response schemas
│   │   └── database/        # DB engine and session factories
│   ├── pyproject.toml
│   └── Dockerfile
├── docker-compose.yml
├── Dockerfile               # Multi-stage: Node build → Go build → Alpine
└── README.md
```

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local Go development)
- uv (for local Python development)

### Run everything with Docker

```bash
# Build and start all services
docker compose up --build

# Rebuild only one service
docker compose up --build python-service
docker compose up --build go-server

# Stop all services
docker compose down

# Stop and wipe database
docker compose down -v

#Local Go development
go run ./cmd/server

#Local Python development
cd python-service
uv sync
uv run uvicorn app.main:app --reload --port 3142
```

## Services

### Go Server (port 3141)

Primary backend API and UI host. Serves the SvelteKit SPA and all API endpoints.

**API Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | SvelteKit UI (Home Base, Players, Jobs) |
| `GET` | `/api/health` | Health check |
| `GET` | `/api/nflstats/players` | List players (filterable, paginated) |
| `GET` | `/api/nflstats/players/{id}` | Get a single player |
| `GET` | `/api/nflstats/stats` | List player stats (filterable, paginated) |
| `GET` | `/api/nflstats/leaders` | Stat leaderboards (top N by any stat) |
| `GET` | `/api/nflstats/games` | List games/schedules (filterable, paginated) |
| `GET` | `/api/nflstats/games/{game_id}` | Get a single game |
| `GET` | `/api/nflstats/rankings` | List fantasy rankings (filterable, paginated) |
| `POST` | `/api/jobs/import` | Start an NFL stats import |
| `GET` | `/api/jobs/{job_id}` | Check import job status |
| `GET` | `/api/jobs` | List import history |

**Player query parameters:**

| Param | Example | Description |
|-------|---------|-------------|
| `team` | `KC` | Filter by NFL team abbreviation |
| `position` | `QB` | Filter by position |
| `search` | `mahomes` | Search by player name (case-insensitive) |
| `offset` | `0` | Pagination offset |
| `limit` | `20` | Results per page (max 100) |

**Stats query parameters:**

| Param | Example | Description |
|-------|---------|-------------|
| `player_id` | `00-0022531` | Filter by NFL player ID |
| `team` | `KC` | Filter by team |
| `position` | `QB` | Filter by position |
| `season` | `2024` | Filter by season |
| `week` | `1` | Filter by week |
| `search` | `mahomes` | Search by player name |

**Leaders query parameters:**

| Param | Example | Description |
|-------|---------|-------------|
| `stat` | `passing_yards` | *(required)* Stat column to rank by |
| `season` | `2024` | *(required)* Season to query |
| `week` | `1` | Optional week filter (0 = all weeks) |
| `position` | `QB` | Optional position filter |
| `limit` | `25` | Number of results (default 25, max 100) |

Valid stat columns: `passing_yards`, `passing_tds`, `rushing_yards`, `rushing_tds`, `receiving_yards`, `receiving_tds`, `receptions`, `targets`, `carries`, `fantasy_points`, `fantasy_points_ppr`, `interceptions`, `sacks`, `completions`, `attempts`.

**Games query parameters:**

| Param | Example | Description |
|-------|---------|-------------|
| `season` | `2024` | Filter by season |
| `week` | `1` | Filter by week |
| `team` | `KC` | Filter by team (matches home or away) |

**Rankings query parameters:**

| Param | Example | Description |
|-------|---------|-------------|
| `rank_type` | `draft` | Filter by ranking type |
| `pos` | `QB` | Filter by position |
| `team` | `KC` | Filter by team |
| `search` | `mahomes` | Search by player name |

### Python Service (port 3142)

Data import service. Called by the Go server to dispatch async imports.

- `GET /health` — Health check
- `POST /api/v1/nflstats/import` — Start an NFL stats import job
- `GET /api/v1/nflstats/jobs/{job_id}` — Check job status

### RabbitMQ Management (port 15672)

Web UI for monitoring queues and messages. Default credentials: `guest` / `guest`.

## NFL Stats Import Pipeline

The Python service provides an async import pipeline for NFL data powered by [nflreadpy](https://github.com/nflverse/nflreadpy). Imports are dispatched as Celery tasks via RabbitMQ so the API responds immediately while the worker processes data in the background.

### Data types

| Collector Type | Data | nflreadpy Function | Description |
|---------------|------|-------------------|-------------|
| `nflreadpy` | Rosters | `load_rosters()` | Master player list with team, position, jersey number |
| `nflreadpy_stats` | Player Stats | `load_player_stats()` | Passing, rushing, receiving, and fantasy stats (weekly or seasonal) |
| `nflreadpy_schedules` | Schedules | `load_schedules()` | Game results, scores, spread, stadium info |
| `nflreadpy_ff_rankings` | Fantasy Rankings | `load_ff_rankings()` | FantasyPros ECR rankings (draft, weekly, or all) |

### Starting an import

```bash
# Import rosters
curl -X POST http://localhost:3141/api/jobs/import \
  -H "Content-Type: application/json" \
  -d '{"collector_type": "nflreadpy", "seasons": [2024]}'

# Import weekly player stats
curl -X POST http://localhost:3141/api/jobs/import \
  -H "Content-Type: application/json" \
  -d '{"collector_type": "nflreadpy_stats", "seasons": [2024], "summary_level": "week"}'

# Import game schedules
curl -X POST http://localhost:3141/api/jobs/import \
  -H "Content-Type: application/json" \
  -d '{"collector_type": "nflreadpy_schedules", "seasons": [2024]}'

# Import fantasy draft rankings
curl -X POST http://localhost:3141/api/jobs/import \
  -H "Content-Type: application/json" \
  -d '{"collector_type": "nflreadpy_ff_rankings", "seasons": [2024], "rank_type": "draft"}'
```

Request body fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `collector_type` | `string` | `"nflreadpy"` | Data source to import (see table above) |
| `seasons` | `int[]` | *(required)* | NFL seasons to import (e.g. `[2023, 2024]`) |
| `strategy` | `string` | `"merge"` | How to reconcile with existing data (see below) |
| `summary_level` | `string` | `"week"` | For `nflreadpy_stats`: `"week"` or `"season"` |
| `rank_type` | `string` | `"draft"` | For `nflreadpy_ff_rankings`: `"draft"`, `"week"`, or `"all"` |

Response (HTTP 202):
```json
{
  "job_id": "abc123-...",
  "status": "accepted",
  "collector_type": "nflreadpy_stats",
  "seasons": [2024],
  "strategy": "merge"
}
```

### Collection strategies

| Strategy | Behaviour |
|----------|-----------|
| `merge` | Upsert — update existing players (matched by `player_id`), insert new ones |
| `replace` | Delete all existing rows from the source, then insert fresh data |
| `append` | Insert everything with no deduplication |
| `dry_run` | Run the collector and validate data but skip database writes |

### Polling job status

```bash
curl http://localhost:3141/api/jobs/{job_id}
```

Response varies by state:

**Pending** (queued, not started):
```json
{ "job_id": "abc123-...", "status": "pending" }
```

**In progress**:
```json
{
  "job_id": "abc123-...",
  "status": "progress",
  "progress": 0.5,
  "meta": {
    "current_season": 2024,
    "seasons_completed": 1,
    "seasons_total": 2,
    "total_players_so_far": 1800
  }
}
```

**Completed**:
```json
{
  "job_id": "abc123-...",
  "status": "completed",
  "result": {
    "status": "completed",
    "collector_type": "nflreadpy_stats",
    "seasons": [2024],
    "strategy": "merge",
    "total_records": 8500,
    "records_inserted": 8200,
    "records_updated": 300,
    "records_skipped": 0,
    "sample": [ "..." ]
  }
}
```

**Failed**:
```json
{ "job_id": "abc123-...", "status": "failed", "error": "..." }
```

### Database tables

Tables are auto-created on startup by both the FastAPI server and the Celery worker. No manual migration is needed.

| Table | Description |
|-------|-------------|
| `players` | Master player list — one row per unique NFL player, keyed by `player_id` |
| `player_seasons` | Per-season/week roster snapshots linked to `player_id` |
| `player_stats` | Weekly/seasonal performance stats (passing, rushing, receiving, fantasy points) |
| `games` | Game schedules and results (scores, spread, stadium, overtime) |
| `fantasy_rankings` | FantasyPros consensus rankings (ECR, standard deviation, best/worst) |
| `collection_history` | Audit log of every import run (counts, status, timing, params) |

### Running the Celery worker locally

```bash
cd python-service
uv sync
uv run celery -A app.celery_app:celery_app worker --loglevel=info
```

This requires a running RabbitMQ instance (default: `amqp://guest:guest@localhost:5672/`) and Postgres.

## Frontend Development

The UI is a SvelteKit SPA in the `web/` directory with a retro pixel "Highwater Hub" theme. It's built with `adapter-static` and embedded into the Go binary at compile time.

### Pages

| Route | Page | Description |
|-------|------|-------------|
| `/` | Home Base | Service launcher grid — links to Plex, Radarr, Sonarr, and other services |
| `/players` | Players | Filterable, paginated roster browser with team/position/search dropdowns |
| `/jobs` | Jobs | Import dashboard — stats cards, data type selector, job history table |

The Jobs page supports importing four data types (Rosters, Player Stats, Schedules, Fantasy Rankings) with type-specific options like summary level and ranking type.

### Dev mode (hot reload)

```bash
cd web
npm install
npm run dev
```

This starts Vite on `http://localhost:5173` with a proxy that forwards `/api/*` requests to the Go server on `:3141`. Make sure the Go server (or Docker stack) is running.

### Production build

The Dockerfile handles everything automatically — the multi-stage build compiles the frontend, copies it into the Go binary, and outputs a single Alpine image.

To build manually:

```bash
cd web && npm run build          # outputs to web/build/
cp -r web/build internal/frontend/dist  # copy for go:embed
cd .. && go build ./cmd/server   # binary now includes the SPA
```

### Tech stack

- **SvelteKit** with TypeScript
- **adapter-static** — SPA mode, all routes fall back to `index.html`
- **Vite** — dev server with API proxy
- **Go embed.FS** — static files baked into the binary at compile time
