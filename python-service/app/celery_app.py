"""Celery application configuration.

The broker URL comes from the ``RABBITMQ_URL`` environment variable
(set by docker-compose).  Results are stored in the same broker by
default — swap to ``rpc://`` or a Postgres result backend later if
you want persistent result storage.
"""

import os

from celery import Celery

RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

celery_app = Celery(
    "nflstats",
    broker=RABBITMQ_URL,
    # Use rpc:// so callers can poll for results via AsyncResult
    backend="rpc://",
)

# -------------------------------------------------------------------
# Task timeout settings (seconds).  Easy to tune via env vars.
# -------------------------------------------------------------------
TASK_SOFT_LIMIT = int(os.getenv("CELERY_TASK_SOFT_TIME_LIMIT", 1800))   # 30 min
TASK_HARD_LIMIT = int(os.getenv("CELERY_TASK_TIME_LIMIT", 2100))        # 35 min

celery_app.conf.update(
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="UTC",
    enable_utc=True,
    # Keep results around for 24 h so the /jobs endpoint can read them
    result_expires=86400,
    # Avoid prefetching many tasks — import jobs are heavy
    worker_prefetch_multiplier=1,
    # Auto-discover tasks in app.tasks
    imports=["app.tasks.import_task", "app.tasks.fantasy_import_task"],
    # Task timeouts — soft raises SoftTimeLimitExceeded (graceful),
    # hard kills the worker process (last resort).
    # Override per-env with CELERY_TASK_SOFT_TIME_LIMIT / CELERY_TASK_TIME_LIMIT.
    task_soft_time_limit=TASK_SOFT_LIMIT,
    task_time_limit=TASK_HARD_LIMIT,
)
