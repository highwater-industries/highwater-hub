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
