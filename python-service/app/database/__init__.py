"""Database configuration — async & sync SQLAlchemy engines and session factories.

The async engine/session is used by the FastAPI web process.
The sync engine/session is used by Celery worker processes (which run
ordinary blocking code).
"""

import os

from sqlalchemy import create_engine
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import Session, sessionmaker

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgresql+asyncpg://appuser:apppass@localhost:5432/nflstats",
)

# ---------------------------------------------------------------------------
# Async engine (for FastAPI / uvicorn)
# ---------------------------------------------------------------------------
_async_url = DATABASE_URL
for _old in ("postgresql://", "postgres://", "postgresql+psycopg2://"):
    _async_url = _async_url.replace(_old, "postgresql+asyncpg://", 1)

engine = create_async_engine(
    _async_url,
    pool_size=5,
    max_overflow=10,
    echo=False,
)

async_session = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,
)

# ---------------------------------------------------------------------------
# Sync engine (for Celery workers)
# ---------------------------------------------------------------------------
_sync_url = DATABASE_URL
for _old in ("postgresql+asyncpg://", "postgres://"):
    _sync_url = _sync_url.replace(_old, "postgresql://", 1)

sync_engine = create_engine(
    _sync_url,
    pool_size=5,
    max_overflow=10,
    echo=False,
)

SyncSession = sessionmaker(
    sync_engine,
    class_=Session,
    expire_on_commit=False,
)


async def create_tables() -> None:
    """Create all ORM tables if they don't already exist (async)."""
    from app.models.models import Base  # local import to avoid circular deps

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


def create_tables_sync() -> None:
    """Create all ORM tables if they don't already exist (sync)."""
    from app.models.models import Base

    Base.metadata.create_all(sync_engine)
