"""
NFL Stats Python Service

FastAPI application providing NFL statistics data operations.
Called by the Go backend server.
"""
from fastapi import FastAPI

from app.routes import nflstats

app = FastAPI(
    title="NFL Stats Service",
    description="Python service for NFL statistics data operations",
    version="0.1.0",
)

app.include_router(nflstats.router, prefix="/api/v1/nflstats", tags=["nflstats"])


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "python-nflstats"}
