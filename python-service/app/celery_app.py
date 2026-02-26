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
    imports=["app.tasks.import_task"],
)
