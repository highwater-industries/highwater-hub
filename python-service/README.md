# NFL Stats Python Service

FastAPI service for NFL statistics data operations. Called by the Go backend.

## Setup (local dev)

```bash
uv sync
uv run uvicorn app.main:app --reload --port 8000
