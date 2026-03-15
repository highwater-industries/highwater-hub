"""Data collector framework.

Provides the base DataCollector class, CollectorFactory registry,
and concrete collector implementations.
"""

from .base import DataCollector
from .factory import CollectorFactory
from .nfl_collector import NFLReadPyCollector
from .stats_collector import PlayerStatsCollector
from .schedules_collector import SchedulesCollector
from .ff_rankings_collector import FFRankingsCollector
from .yahoo_fantasy_collector import YahooFantasyCollector
from .espn_fantasy_collector import ESPNFantasyCollector

__all__ = [
    "DataCollector",
    "CollectorFactory",
    "NFLReadPyCollector",
    "PlayerStatsCollector",
    "SchedulesCollector",
    "FFRankingsCollector",
    "YahooFantasyCollector",
    "ESPNFantasyCollector",
]
