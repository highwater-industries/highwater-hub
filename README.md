# ffredux

A multi-service fantasy football platform built with Go, Python, and JavaScript.

## Architecture

| Service | Language | Port | Description |
|---------|----------|------|-------------|
| Go Server | Go | 3141 | Backend API server — domain logic, routing, auth |
| Python Service | Python (FastAPI) | 3142 | Data service — NFL stats import via nflmypy, async job processing |
| Postgres | — | 5432 | Shared database for domain data, stats, and job state |
| RabbitMQ | — | 5672 / 15672 | Message broker for async task dispatch (Celery) |

The Go server is the primary API, calling the Python service over HTTP for data-heavy operations. Both services read/write to the same Postgres database. Long-running imports are dispatched asynchronously via Celery + RabbitMQ, with job status tracked in Postgres.

A JavaScript frontend (separate repo) consumes the Go API.

## Project Structure
fredux/
├── cmd/
│ ├── server/ # Go HTTP server entry point
│ └── cli/ # Go CLI tool entry point
├── internal/
│ ├── server/ # HTTP server, routes, middleware
│ ├── user/ # User domain model
│ └── httputil/ # HTTP response helpers
├── python-service/
│ ├── app/
│ │ ├── main.py # FastAPI entry point
│ │ ├── routes/ # API route handlers
│ │ ├── models/ # SQLAlchemy models
│ │ └── schemas/ # Pydantic request/response schemas
│ ├── pyproject.toml
│ └── Dockerfile
├── docker-compose.yml
├── Dockerfile # Go server Dockerfile
└── README.model

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
Go Server (port 3141)
Primary backend API. Handles domain logic, user management, and proxies data requests to the Python service.

Python Service (port 3142)
GET /health — Health check
POST /api/v1/nflstats/import — Start an NFL stats import job
GET /api/v1/nflstats/jobs/{job_id} — Check job status
RabbitMQ Management (port 15672)
Web UI for monitoring queues and messages. Default credentials: guest / guest.
