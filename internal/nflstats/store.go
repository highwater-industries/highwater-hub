package nflstats

import "context"

// PlayerFilter holds optional query parameters for filtering players.
// All fields are pointers — nil means "don't filter on this field."
type PlayerFilter struct {
	Team     *string // exact match: ?team=KC
	Position *string // exact match: ?position=QB
	Source   *string // exact match: ?source=nflreadpy
	Search   *string // case-insensitive substring match on player_name: ?search=mahomes
}

// Store reads player data from the database.
type Store interface {
	// Get returns a single player by their internal database ID.
	Get(ctx context.Context, id int) (Player, error)

	// GetByPlayerID returns a single player by their external NFL ID (e.g. "00-0022531").
	GetByPlayerID(ctx context.Context, playerID string) (Player, error)

	// List returns a filtered, paginated list of players and the total count
	// matching the filter (for pagination metadata).
	List(ctx context.Context, filter PlayerFilter, offset, limit int) ([]Player, int, error)
}
