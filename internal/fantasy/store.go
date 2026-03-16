package fantasy

import "context"

// Store reads fantasy league data from the database (read-only from Go's perspective).
type Store interface {
	// ListLeagues returns paginated leagues with optional platform/season filter.
	ListLeagues(ctx context.Context, filter LeagueFilter, offset, limit int) ([]League, int, error)

	// GetLeague returns a single league by ID.
	GetLeague(ctx context.Context, id int) (League, error)

	// ListTeams returns all teams for a given league.
	ListTeams(ctx context.Context, leagueID int) ([]Team, error)

	// GetTeam returns a single team by ID.
	GetTeam(ctx context.Context, teamID int) (Team, error)

	// ListRoster returns all roster entries for a given team.
	ListRoster(ctx context.Context, teamID int) ([]RosterEntry, error)

	// ListMatchups returns all weekly matchup rows for a given league.
	ListMatchups(ctx context.Context, leagueID int) ([]Matchup, error)
}
