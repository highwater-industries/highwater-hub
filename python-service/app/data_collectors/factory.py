"""Factory / registry for data collectors.

Concrete collectors auto-register themselves at import time, so the
rest of the application never needs to hard-code a collector class.
"""

from typing import Dict, Type
import logging

from .base import DataCollector

logger = logging.getLogger(__name__)


class CollectorFactory:
    """Create data collector instances by source-type string."""

    _collectors: Dict[str, Type[DataCollector]] = {}

    @classmethod
    def register(cls, source_type: str, collector_class: Type[DataCollector]) -> None:
        """Register a new collector type."""
        if source_type in cls._collectors:
            logger.warning("Overwriting existing collector for type: %s", source_type)
        cls._collectors[source_type] = collector_class
        logger.info(
            "Registered collector: %s -> %s", source_type, collector_class.__name__
        )

    @classmethod
    def create(cls, source_type: str, **kwargs) -> DataCollector:
        """Instantiate a collector for *source_type*.

        Raises ``ValueError`` if the type has not been registered.
        """
        if source_type not in cls._collectors:
            available = ", ".join(cls._collectors.keys())
            raise ValueError(
                f"Unknown collector type: '{source_type}'. "
                f"Available types: {available or 'none registered'}"
            )
        collector_class = cls._collectors[source_type]
        return collector_class(**kwargs)

    @classmethod
    def get_available_types(cls) -> list[str]:
        """Return all registered collector type identifiers."""
        return list(cls._collectors.keys())

    @classmethod
    def unregister(cls, source_type: str) -> None:
        """Remove a collector registration."""
        if source_type in cls._collectors:
            del cls._collectors[source_type]
            logger.info("Unregistered collector: %s", source_type)
