package fantasy

// League represents a fantasy football league imported from an external platform.
type League struct {
	ID               int            `json:"id"`
	ExternalLeagueID string         `json:"external_league_id"`
	LeagueName       string         `json:"league_name"`
	Platform         string         `json:"platform"`
	Season           int            `json:"season"`
	NumTeams         *int           `json:"num_teams,omitempty"`
	ScoringType      *string        `json:"scoring_type,omitempty"`
	Settings         map[string]any `json:"settings,omitempty"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
}

// Team represents a team within a fantasy league.
type Team struct {
	ID               int     `json:"id"`
	LeagueID         int     `json:"league_id"`
	ExternalTeamID   *string `json:"external_team_id,omitempty"`
	TeamName         string  `json:"team_name"`
	OwnerName        *string `json:"owner_name,omitempty"`
	Wins             int     `json:"wins"`
	Losses           int     `json:"losses"`
	Ties             int     `json:"ties"`
	PointsFor        float64 `json:"points_for"`
	PointsAgainst    float64 `json:"points_against"`
	StandingRank     *int    `json:"standing_rank,omitempty"`
	PlayoffSeed      *int    `json:"playoff_seed,omitempty"`
	LogoURL          *string `json:"logo_url,omitempty"`
	StreakType       *string `json:"streak_type,omitempty"`
	StreakValue      int     `json:"streak_value"`
	WaiverPriority   int     `json:"waiver_priority"`
	NumberOfMoves    int     `json:"number_of_moves"`
	NumberOfTrades   int     `json:"number_of_trades"`
	ClinchedPlayoffs bool    `json:"clinched_playoffs"`
	DraftGrade       *string `json:"draft_grade,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// RosterEntry represents a player slot on a fantasy team.
type RosterEntry struct {
	ID               int     `json:"id"`
	TeamID           int     `json:"team_id"`
	PlayerID         *string `json:"player_id,omitempty"`
	PlayerName       string  `json:"player_name"`
	PlayerPosition   string  `json:"player_position"`
	NFLTeam          *string `json:"nfl_team,omitempty"`
	RosterPosition   *string `json:"roster_position,omitempty"`
	ExternalPlayerID *string `json:"external_player_id,omitempty"`
	Matched          bool    `json:"matched"`
	CreatedAt        string  `json:"created_at"`
}

// LeagueFilter holds optional filter criteria for listing leagues.
type LeagueFilter struct {
	Platform string // yahoo, espn, sleeper
	Season   int    // 0 = all
}

// ImportRequest is the body sent from the Go server to the Python service
// to start a fantasy league import.
type ImportRequest struct {
	Platform string `json:"platform"`
	LeagueID string `json:"league_id"`
	Season   int    `json:"season"`
	SWID     string `json:"swid,omitempty"`
	EspnS2   string `json:"espn_s2,omitempty"`
}

// ImportAccepted is the response from the Python service when an import is dispatched.
type ImportAccepted struct {
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	Platform string `json:"platform"`
	LeagueID string `json:"league_id"`
	Season   int    `json:"season"`
}

// LeagueDetail is a league with its teams and per-team roster counts.
type LeagueDetail struct {
	League League `json:"league"`
	Teams  []Team `json:"teams"`
}

// TeamDetail is a team with its full roster.
type TeamDetail struct {
	Team   Team          `json:"team"`
	Roster []RosterEntry `json:"roster"`
}

// Matchup represents one team's result in a weekly head-to-head matchup.
type Matchup struct {
	ID             int     `json:"id"`
	LeagueID       int     `json:"league_id"`
	Week           int     `json:"week"`
	MatchupID      int     `json:"matchup_id"`
	TeamName       string  `json:"team_name"`
	ExternalTeamID *string `json:"external_team_id,omitempty"`
	Points         float64 `json:"points"`
	Result         *string `json:"result,omitempty"`
	IsPlayoff      bool    `json:"is_playoff"`
	CreatedAt      string  `json:"created_at"`
}
