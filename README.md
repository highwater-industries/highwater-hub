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
│   ├── nflstats/            # NFL player queries (handlers, store, filters)
│   ├── user/                # User domain model
│   └── httputil/            # HTTP response helpers
├── web/                     # SvelteKit frontend source
│   ├── src/
│   │   ├── routes/          # Pages: Dashboard, Players, Jobs
│   │   └── lib/api.ts       # Typed API client
│   ├── svelte.config.js     # adapter-static (SPA mode)
│   └── vite.config.ts       # Dev proxy to Go server
├── python-service/
│   ├── app/
│   │   ├── main.py          # FastAPI entry point
│   │   ├── celery_app.py    # Celery configuration
│   │   ├── tasks/           # Async import tasks
│   │   ├── data_collectors/ # Data source adapters (nflreadpy)
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
| `GET` | `/` | SvelteKit UI (Dashboard, Players, Jobs) |
| `GET` | `/api/health` | Health check |
| `GET` | `/api/nflstats/players` | List players (filterable, paginated) |
| `GET` | `/api/nflstats/players/{id}` | Get a single player |
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

### Python Service (port 3142)

Data import service. Called by the Go server to dispatch async imports.

- `GET /health` — Health check
- `POST /api/v1/nflstats/import` — Start an NFL stats import job
- `GET /api/v1/nflstats/jobs/{job_id}` — Check job status

### RabbitMQ Management (port 15672)

Web UI for monitoring queues and messages. Default credentials: `guest` / `guest`.

## NFL Stats Import Pipeline

The Python service provides an async import pipeline for NFL player data. Imports are dispatched as Celery tasks via RabbitMQ so the API responds immediately while the worker processes data in the background.

### Starting an import

```bash
curl -X POST http://localhost:3142/api/v1/nflstats/import \
  -H "Content-Type: application/json" \
  -d '{"seasons": [2024]}'
```

Request body fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `seasons` | `int[]` | *(required)* | NFL seasons to import (e.g. `[2023, 2024]`) |
| `collector_type` | `string` | `"nflreadpy"` | Data source collector to use |
| `strategy` | `string` | `"merge"` | How to reconcile with existing data (see below) |

Response (HTTP 202):
```json
{
  "job_id": "abc123-...",
  "status": "accepted",
  "collector_type": "nflreadpy",
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
curl http://localhost:3142/api/v1/nflstats/jobs/{job_id}
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
    "collector_type": "nflreadpy",
    "seasons": [2024],
    "strategy": "merge",
    "total_players": 1800,
    "records_inserted": 1500,
    "records_updated": 300,
    "records_skipped": 0,
    "players_sample": [ "..." ]
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
| `collection_history` | Audit log of every import run (counts, status, timing, params) |

### Running the Celery worker locally

```bash
cd python-service
uv sync
uv run celery -A app.celery_app:celery_app worker --loglevel=info
```

This requires a running RabbitMQ instance (default: `amqp://guest:guest@localhost:5672/`) and Postgres.

## Frontend Development

The UI is a SvelteKit SPA in the `web/` directory, built with `adapter-static` and embedded into the Go binary at compile time.

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
