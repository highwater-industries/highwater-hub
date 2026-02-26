"""Abstract base class for all data collectors.

Every collector follows a three-step pipeline:
  fetch  -> pull raw data from an external source
  validate -> sanity-check the raw payload
  transform -> convert to the internal format

The high-level `collect()` method runs all three steps in order and
handles errors/logging uniformly.
"""

from abc import ABC, abstractmethod
from datetime import UTC, datetime
from typing import Any, Dict, Optional
import logging

logger = logging.getLogger(__name__)


class DataCollector(ABC):
    """Base class for all data collectors.

    Each collector is responsible for:
    1. Fetching data from an external source
    2. Validating the fetched data
    3. Transforming data to internal format
    4. Handling errors gracefully
    """

    def __init__(self, name: str):
        self.name = name
        self.last_fetch_time: Optional[datetime] = None
        self.last_error: Optional[str] = None

    # ------------------------------------------------------------------
    # Abstract interface — subclasses must implement
    # ------------------------------------------------------------------

    @abstractmethod
    async def fetch(self) -> Dict[str, Any]:
        """Fetch raw data from the external source."""
        ...

    @abstractmethod
    def validate(self, data: Dict[str, Any]) -> bool:
        """Return True if *data* passes structural validation."""
        ...

    @abstractmethod
    async def transform(self, data: Dict[str, Any]) -> Any:
        """Convert raw data to the internal format."""
        ...

    # ------------------------------------------------------------------
    # Pipeline
    # ------------------------------------------------------------------

    async def collect(self) -> Optional[Any]:
        """Execute the full collection pipeline: fetch → validate → transform.

        Returns transformed data on success, ``None`` on any failure.
        """
        try:
            logger.info("Starting data collection from %s", self.name)

            raw_data = await self.fetch()
            self.last_fetch_time = datetime.now(UTC)

            if not self.validate(raw_data):
                error_msg = f"Validation failed for {self.name}"
                logger.error(error_msg)
                self.last_error = error_msg
                return None

            transformed = await self.transform(raw_data)

            logger.info("Successfully collected data from %s", self.name)
            self.last_error = None
            return transformed

        except Exception as e:
            error_msg = f"Error collecting data from {self.name}: {e}"
            logger.error(error_msg, exc_info=True)
            self.last_error = error_msg
            return None

    # ------------------------------------------------------------------
    # Status
    # ------------------------------------------------------------------

    def get_status(self) -> Dict[str, Any]:
        """Return a serialisable status dict for this collector."""
        return {
            "name": self.name,
            "last_fetch_time": (
                self.last_fetch_time.isoformat() if self.last_fetch_time else None
            ),
            "last_error": self.last_error,
            "status": "healthy" if not self.last_error else "error",
        }
