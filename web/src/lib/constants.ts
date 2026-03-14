// Shared reference data for dropdowns across the app.

export const NFL_TEAMS = [
	{ abbr: 'ARI', name: 'Arizona Cardinals' },
	{ abbr: 'ATL', name: 'Atlanta Falcons' },
	{ abbr: 'BAL', name: 'Baltimore Ravens' },
	{ abbr: 'BUF', name: 'Buffalo Bills' },
	{ abbr: 'CAR', name: 'Carolina Panthers' },
	{ abbr: 'CHI', name: 'Chicago Bears' },
	{ abbr: 'CIN', name: 'Cincinnati Bengals' },
	{ abbr: 'CLE', name: 'Cleveland Browns' },
	{ abbr: 'DAL', name: 'Dallas Cowboys' },
	{ abbr: 'DEN', name: 'Denver Broncos' },
	{ abbr: 'DET', name: 'Detroit Lions' },
	{ abbr: 'GB', name: 'Green Bay Packers' },
	{ abbr: 'HOU', name: 'Houston Texans' },
	{ abbr: 'IND', name: 'Indianapolis Colts' },
	{ abbr: 'JAX', name: 'Jacksonville Jaguars' },
	{ abbr: 'KC', name: 'Kansas City Chiefs' },
	{ abbr: 'LA', name: 'Los Angeles Rams' },
	{ abbr: 'LAC', name: 'Los Angeles Chargers' },
	{ abbr: 'LV', name: 'Las Vegas Raiders' },
	{ abbr: 'MIA', name: 'Miami Dolphins' },
	{ abbr: 'MIN', name: 'Minnesota Vikings' },
	{ abbr: 'NE', name: 'New England Patriots' },
	{ abbr: 'NO', name: 'New Orleans Saints' },
	{ abbr: 'NYG', name: 'New York Giants' },
	{ abbr: 'NYJ', name: 'New York Jets' },
	{ abbr: 'PHI', name: 'Philadelphia Eagles' },
	{ abbr: 'PIT', name: 'Pittsburgh Steelers' },
	{ abbr: 'SEA', name: 'Seattle Seahawks' },
	{ abbr: 'SF', name: 'San Francisco 49ers' },
	{ abbr: 'TB', name: 'Tampa Bay Buccaneers' },
	{ abbr: 'TEN', name: 'Tennessee Titans' },
	{ abbr: 'WAS', name: 'Washington Commanders' },
];

export const POSITIONS = [
	{ abbr: 'QB', name: 'Quarterback' },
	{ abbr: 'RB', name: 'Running Back' },
	{ abbr: 'WR', name: 'Wide Receiver' },
	{ abbr: 'TE', name: 'Tight End' },
	{ abbr: 'OL', name: 'Offensive Line' },
	{ abbr: 'DL', name: 'Defensive Line' },
	{ abbr: 'LB', name: 'Linebacker' },
	{ abbr: 'DB', name: 'Defensive Back' },
	{ abbr: 'K', name: 'Kicker' },
	{ abbr: 'P', name: 'Punter' },
	{ abbr: 'LS', name: 'Long Snapper' },
];

// Generate a range of NFL seasons (2000–current year)
const currentYear = new Date().getFullYear();
export const SEASONS: number[] = Array.from(
	{ length: currentYear - 1999 },
	(_, i) => currentYear - i
);

export const SOURCES = ['nflreadpy'];

// Stat types for multi-source filtering
export const STAT_TYPES = [
	{ value: 'actual', label: 'Actual' },
	{ value: 'projected', label: 'Projected' },
	{ value: 'fantasy', label: 'Fantasy' },
];

// Collector types for the import form
export const COLLECTOR_TYPES = [
	{ value: 'nflreadpy', label: 'Rosters' },
	{ value: 'nflreadpy_stats', label: 'Player Stats' },
	{ value: 'nflreadpy_schedules', label: 'Schedules' },
	{ value: 'nflreadpy_ff_rankings', label: 'Fantasy Rankings' },
];

export const SUMMARY_LEVELS = [
	{ value: 'week', label: 'Weekly' },
	{ value: 'season', label: 'Season' },
];

export const RANK_TYPES = [
	{ value: 'draft', label: 'Draft' },
	{ value: 'week', label: 'Weekly' },
	{ value: 'all', label: 'All' },
];

// NFL weeks (1–18 regular season, 19–22 postseason)
export const NFL_WEEKS: number[] = Array.from({ length: 22 }, (_, i) => i + 1);

// Stat columns available for the leaders query
export const LEADER_STATS = [
	{ value: 'passing_yards', label: 'Pass Yards' },
	{ value: 'passing_tds', label: 'Pass TDs' },
	{ value: 'rushing_yards', label: 'Rush Yards' },
	{ value: 'rushing_tds', label: 'Rush TDs' },
	{ value: 'receiving_yards', label: 'Rec Yards' },
	{ value: 'receiving_tds', label: 'Rec TDs' },
	{ value: 'receptions', label: 'Receptions' },
	{ value: 'targets', label: 'Targets' },
	{ value: 'carries', label: 'Carries' },
	{ value: 'fantasy_points', label: 'Fantasy Pts' },
	{ value: 'fantasy_points_ppr', label: 'PPR Pts' },
	{ value: 'interceptions', label: 'INTs' },
	{ value: 'sacks', label: 'Sacks' },
	{ value: 'completions', label: 'Completions' },
	{ value: 'attempts', label: 'Attempts' },
];

// Batch import presets
export type ImportPreset = {
	label: string;
	desc: string;
	build: (season: number) => import('$lib/api').ImportOptions[];
};

export const IMPORT_PRESETS: ImportPreset[] = [
	{
		label: 'FULL SEASON',
		desc: 'Rosters + Weekly Stats + Schedules + Rankings',
		build: (season) => [
			{ collector_type: 'nflreadpy', seasons: [season], strategy: 'merge' },
			{ collector_type: 'nflreadpy_stats', seasons: [season], strategy: 'merge', summary_level: 'week' },
			{ collector_type: 'nflreadpy_schedules', seasons: [season], strategy: 'merge' },
			{ collector_type: 'nflreadpy_ff_rankings', seasons: [season], strategy: 'merge', rank_type: 'draft' },
		],
	},
	{
		label: 'ROSTERS + STATS',
		desc: 'Player roster and weekly stat lines',
		build: (season) => [
			{ collector_type: 'nflreadpy', seasons: [season], strategy: 'merge' },
			{ collector_type: 'nflreadpy_stats', seasons: [season], strategy: 'merge', summary_level: 'week' },
		],
	},
	{
		label: 'FANTASY PACKAGE',
		desc: 'Stats + Rankings for fantasy analysis',
		build: (season) => [
			{ collector_type: 'nflreadpy_stats', seasons: [season], strategy: 'merge', summary_level: 'week' },
			{ collector_type: 'nflreadpy_ff_rankings', seasons: [season], strategy: 'merge', rank_type: 'draft' },
		],
	},
];
