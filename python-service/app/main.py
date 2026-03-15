"""
NFL Stats Python Service

FastAPI application providing NFL statistics data operations.
Called by the Go backend server.
"""
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.database import create_tables
from app.routes import nflstats
from app.routes import fantasy

# Import the data_collectors package so collector classes auto-register
# with the CollectorFactory at startup.
import app.data_collectors  # noqa: F401

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown lifecycle."""
    logger.info("Creating database tables (if they don't exist)…")
    await create_tables()
    logger.info("Database tables ready.")
    yield


app = FastAPI(
    title="NFL Stats Service",
    description="Python service for NFL statistics data operations",
    version="0.1.0",
    lifespan=lifespan,
)

app.include_router(nflstats.router, prefix="/api/v1/nflstats", tags=["nflstats"])
app.include_router(fantasy.router, prefix="/api/v1/fantasy", tags=["fantasy"])


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "python-nflstats"}
