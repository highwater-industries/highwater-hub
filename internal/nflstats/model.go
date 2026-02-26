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
