package nflstats

// Player represents an NFL player from the database.
type Player struct {
	ID             int            `json:"id"`
	PlayerID       *string        `json:"player_id,omitempty"`
	PlayerName     string         `json:"player_name"`
	Team           *string        `json:"team,omitempty"`
	PlayerPosition *string        `json:"player_position,omitempty"`
	Source         *string        `json:"source,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

// PlayerStat represents a row from the player_stats table.
type PlayerStat struct {
	ID                       int      `json:"id"`
	PlayerDbID               *int     `json:"player_db_id,omitempty"`
	PlayerID                 *string  `json:"player_id,omitempty"`
	PlayerName               string   `json:"player_name"`
	PlayerDisplayName        *string  `json:"player_display_name,omitempty"`
	Position                 *string  `json:"position,omitempty"`
	PositionGroup            *string  `json:"position_group,omitempty"`
	Team                     *string  `json:"team,omitempty"`
	Season                   int      `json:"season"`
	Week                     int      `json:"week"`
	StatType                 *string  `json:"stat_type,omitempty"`
	SeasonType               *string  `json:"season_type,omitempty"`
	OpponentTeam             *string  `json:"opponent_team,omitempty"`
	Completions              *int     `json:"completions,omitempty"`
	Attempts                 *int     `json:"attempts,omitempty"`
	PassingYards             *float64 `json:"passing_yards,omitempty"`
	PassingTds               *int     `json:"passing_tds,omitempty"`
	Interceptions            *int     `json:"interceptions,omitempty"`
	Sacks                    *float64 `json:"sacks,omitempty"`
	SackYards                *float64 `json:"sack_yards,omitempty"`
	PassingAirYards          *float64 `json:"passing_air_yards,omitempty"`
	PassingYardsAfterCatch   *float64 `json:"passing_yards_after_catch,omitempty"`
	Passing2ptConversions    *int     `json:"passing_2pt_conversions,omitempty"`
	Carries                  *int     `json:"carries,omitempty"`
	RushingYards             *float64 `json:"rushing_yards,omitempty"`
	RushingTds               *int     `json:"rushing_tds,omitempty"`
	RushingFumbles           *int     `json:"rushing_fumbles,omitempty"`
	RushingFumblesLost       *int     `json:"rushing_fumbles_lost,omitempty"`
	Rushing2ptConversions    *int     `json:"rushing_2pt_conversions,omitempty"`
	Receptions               *int     `json:"receptions,omitempty"`
	Targets                  *int     `json:"targets,omitempty"`
	ReceivingYards           *float64 `json:"receiving_yards,omitempty"`
	ReceivingTds             *int     `json:"receiving_tds,omitempty"`
	ReceivingFumbles         *int     `json:"receiving_fumbles,omitempty"`
	ReceivingFumblesLost     *int     `json:"receiving_fumbles_lost,omitempty"`
	ReceivingAirYards        *float64 `json:"receiving_air_yards,omitempty"`
	ReceivingYardsAfterCatch *float64 `json:"receiving_yards_after_catch,omitempty"`
	Receiving2ptConversions  *int     `json:"receiving_2pt_conversions,omitempty"`
	FantasyPoints            *float64 `json:"fantasy_points,omitempty"`
	FantasyPointsPPR         *float64 `json:"fantasy_points_ppr,omitempty"`
	SpecialTeamsTds          *int     `json:"special_teams_tds,omitempty"`
	Source                   *string  `json:"source,omitempty"`
	CreatedAt                string   `json:"created_at"`
}

// Game represents a row from the games table.
type Game struct {
	ID         int      `json:"id"`
	GameID     *string  `json:"game_id,omitempty"`
	Season     *int     `json:"season,omitempty"`
	GameType   *string  `json:"game_type,omitempty"`
	Week       *int     `json:"week,omitempty"`
	Gameday    *string  `json:"gameday,omitempty"`
	Weekday    *string  `json:"weekday,omitempty"`
	Gametime   *string  `json:"gametime,omitempty"`
	AwayTeam   *string  `json:"away_team,omitempty"`
	HomeTeam   *string  `json:"home_team,omitempty"`
	AwayScore  *int     `json:"away_score,omitempty"`
	HomeScore  *int     `json:"home_score,omitempty"`
	Result     *int     `json:"result,omitempty"`
	Total      *int     `json:"total,omitempty"`
	SpreadLine *float64 `json:"spread_line,omitempty"`
	TotalLine  *float64 `json:"total_line,omitempty"`
	Overtime   *bool    `json:"overtime,omitempty"`
	Location   *string  `json:"location,omitempty"`
	Roof       *string  `json:"roof,omitempty"`
	Surface    *string  `json:"surface,omitempty"`
	Stadium    *string  `json:"stadium,omitempty"`
	Source     *string  `json:"source,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

// --------------------------------------------------------------------------
// Player Summary (aggregated view for player detail page)
// --------------------------------------------------------------------------

// SeasonTotals holds aggregated stats for a single season.
type SeasonTotals struct {
	Season           int      `json:"season"`
	GamesPlayed      int      `json:"games_played"`
	Completions      *int     `json:"completions,omitempty"`
	Attempts         *int     `json:"attempts,omitempty"`
	PassingYards     *float64 `json:"passing_yards,omitempty"`
	PassingTds       *int     `json:"passing_tds,omitempty"`
	Interceptions    *int     `json:"interceptions,omitempty"`
	Carries          *int     `json:"carries,omitempty"`
	RushingYards     *float64 `json:"rushing_yards,omitempty"`
	RushingTds       *int     `json:"rushing_tds,omitempty"`
	Receptions       *int     `json:"receptions,omitempty"`
	Targets          *int     `json:"targets,omitempty"`
	ReceivingYards   *float64 `json:"receiving_yards,omitempty"`
	ReceivingTds     *int     `json:"receiving_tds,omitempty"`
	FantasyPoints    *float64 `json:"fantasy_points,omitempty"`
	FantasyPointsPPR *float64 `json:"fantasy_points_ppr,omitempty"`
}

// PlayerSummary is the combined response for the player detail page.
type PlayerSummary struct {
	Player       Player           `json:"player"`
	CareerTotals SeasonTotals     `json:"career_totals"`
	Seasons      []SeasonTotals   `json:"seasons"`
	RecentGames  []PlayerStat     `json:"recent_games"`
	Rankings     []FantasyRanking `json:"rankings"`
}

// FantasyRanking represents a row from the fantasy_rankings table.
type FantasyRanking struct {
	ID         int      `json:"id"`
	PlayerDbID *int     `json:"player_db_id,omitempty"`
	PlayerID   *string  `json:"player_id,omitempty"`
	PlayerName string   `json:"player_name"`
	Pos        *string  `json:"pos,omitempty"`
	Team       *string  `json:"team,omitempty"`
	Rank       *int     `json:"rank,omitempty"`
	ECR        *float64 `json:"ecr,omitempty"`
	SD         *float64 `json:"sd,omitempty"`
	Best       *int     `json:"best,omitempty"`
	Worst      *int     `json:"worst,omitempty"`
	Avg        *float64 `json:"avg,omitempty"`
	RankType   *string  `json:"rank_type,omitempty"`
	PageType   *string  `json:"page_type,omitempty"`
	Season     *int     `json:"season,omitempty"`
	Week       *int     `json:"week,omitempty"`
	Source     *string  `json:"source,omitempty"`
	CreatedAt  string   `json:"created_at"`
}
