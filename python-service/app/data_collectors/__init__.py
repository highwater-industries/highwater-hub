"""Data collector framework.

Provides the base DataCollector class, CollectorFactory registry,
and concrete collector implementations.
"""

from .base import DataCollector
from .factory import CollectorFactory
from .nfl_collector import NFLReadPyCollector

__all__ = ["DataCollector", "CollectorFactory", "NFLReadPyCollector"]
