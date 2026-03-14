package nflstats

import "context"

// PlayerFilter holds optional query parameters for filtering players.
// All fields are pointers — nil means "don't filter on this field."
type PlayerFilter struct {
	Team     *string // exact match: ?team=KC
	Position *string // exact match: ?position=QB
	Source   *string // exact match: ?source=nflreadpy
	Search   *string // case-insensitive substring match on player_name: ?search=mahomes
	Sort     string  // column to ORDER BY
	Order    string  // ASC or DESC
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

// --------------------------------------------------------------------------
// Player Stats
// --------------------------------------------------------------------------

// StatFilter holds optional query parameters for filtering player stats.
type StatFilter struct {
	PlayerID   *string
	Team       *string
	Position   *string
	Season     *int
	Week       *int
	StatType   *string // actual, projected, fantasy
	SeasonType *string // REG, POST
	Source     *string // data source filter
	Search     *string // case-insensitive substring on player_name
	GroupBy    string  // "season" for aggregated season totals
	Sort       string
	Order      string
}

// StatStore reads player stat data.
type StatStore interface {
	ListStats(ctx context.Context, filter StatFilter, offset, limit int) ([]PlayerStat, int, error)
	ListSeasonStats(ctx context.Context, filter StatFilter, offset, limit int) ([]PlayerStat, int, error)
	GetLeaders(ctx context.Context, stat string, season, week int, position string, limit int) ([]PlayerStat, error)
	GetPlayerSummary(ctx context.Context, playerID int) (*PlayerSummary, error)
}

// --------------------------------------------------------------------------
// Games / Schedule
// --------------------------------------------------------------------------

// GameFilter holds optional query parameters for filtering games.
type GameFilter struct {
	Season *int
	Week   *int
	Team   *string // matches home_team OR away_team
	Sort   string
	Order  string
}

// GameStore reads game/schedule data.
type GameStore interface {
	ListGames(ctx context.Context, filter GameFilter, offset, limit int) ([]Game, int, error)
	GetGame(ctx context.Context, gameID string) (Game, error)
}

// --------------------------------------------------------------------------
// Fantasy Rankings
// --------------------------------------------------------------------------

// RankingFilter holds optional query parameters for filtering rankings.
type RankingFilter struct {
	RankType *string
	Pos      *string
	Team     *string
	Season   *int
	Week     *int
	Source   *string
	Search   *string
	Sort     string
	Order    string
}

// RankingStore reads fantasy ranking data.
type RankingStore interface {
	ListRankings(ctx context.Context, filter RankingFilter, offset, limit int) ([]FantasyRanking, int, error)
}
