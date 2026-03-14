package nflstats

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// PostgresStore implements Store by querying the players table.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new store backed by the given database connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// --------------------------------------------------------------------------
// Single-row lookups
// --------------------------------------------------------------------------

// Get returns a single player by internal database ID.
func (s *PostgresStore) Get(ctx context.Context, id int) (Player, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, player_id, player_name, team, player_position,
                source, metadata, created_at, updated_at
         FROM players WHERE id = $1`, id,
	)
	return scanPlayer(row)
}

// GetByPlayerID returns a single player by external NFL ID (e.g. "00-0022531").
func (s *PostgresStore) GetByPlayerID(ctx context.Context, playerID string) (Player, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, player_id, player_name, team, player_position,
                source, metadata, created_at, updated_at
         FROM players WHERE player_id = $1`, playerID,
	)
	return scanPlayer(row)
}

// --------------------------------------------------------------------------
// Filtered list
// --------------------------------------------------------------------------

// List returns a filtered, paginated list of players.
func (s *PostgresStore) List(ctx context.Context, filter PlayerFilter, offset, limit int) ([]Player, int, error) {
	// Build WHERE clause dynamically from non-nil filter fields
	where, args := buildWhere(filter)

	// 1. Count total matching rows
	var total int
	countSQL := "SELECT COUNT(*) FROM players" + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count players: %w", err)
	}

	// 2. Fetch the page
	orderBy := buildPlayerOrderBy(filter.Sort, filter.Order)

	querySQL := fmt.Sprintf(
		`SELECT id, player_id, player_name, team, player_position,
                source, metadata, created_at, updated_at
         FROM players%s
         %s
         LIMIT $%d OFFSET $%d`,
		where, orderBy, len(args)+1, len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query players: %w", err)
	}
	defer rows.Close()

	players := make([]Player, 0)
	for rows.Next() {
		p, err := scanPlayerRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan player: %w", err)
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate players: %w", err)
	}

	return players, total, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// buildWhere constructs a WHERE clause and parameter list from a PlayerFilter.
// Returns an empty string and nil args if no filters are set.
func buildWhere(f PlayerFilter) (string, []any) {
	var conditions []string
	var args []any

	if f.Team != nil {
		args = append(args, *f.Team)
		conditions = append(conditions, fmt.Sprintf("team = $%d", len(args)))
	}
	if f.Position != nil {
		args = append(args, *f.Position)
		conditions = append(conditions, fmt.Sprintf("player_position = $%d", len(args)))
	}
	if f.Source != nil {
		args = append(args, *f.Source)
		conditions = append(conditions, fmt.Sprintf("source = $%d", len(args)))
	}
	if f.Search != nil {
		args = append(args, "%"+*f.Search+"%")
		conditions = append(conditions, fmt.Sprintf("player_name ILIKE $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// scanPlayer scans a single *sql.Row into a Player.
func scanPlayer(row *sql.Row) (Player, error) {
	var p Player
	var playerID, team, pos, source sql.NullString
	var metaJSON sql.NullString

	err := row.Scan(
		&p.ID, &playerID, &p.PlayerName, &team, &pos,
		&source, &metaJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return Player{}, fmt.Errorf("scan player: %w", err)
	}

	if playerID.Valid {
		p.PlayerID = &playerID.String
	}
	if team.Valid {
		p.Team = &team.String
	}
	if pos.Valid {
		p.PlayerPosition = &pos.String
	}
	if source.Valid {
		p.Source = &source.String
	}
	if metaJSON.Valid {
		json.Unmarshal([]byte(metaJSON.String), &p.Metadata)
	}

	return p, nil
}

// scanPlayerRow scans a single row from *sql.Rows into a Player.
// Same logic as scanPlayer but works with the multi-row iterator.
func scanPlayerRow(rows *sql.Rows) (Player, error) {
	var p Player
	var playerID, team, pos, source sql.NullString
	var metaJSON sql.NullString

	err := rows.Scan(
		&p.ID, &playerID, &p.PlayerName, &team, &pos,
		&source, &metaJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return Player{}, err
	}

	if playerID.Valid {
		p.PlayerID = &playerID.String
	}
	if team.Valid {
		p.Team = &team.String
	}
	if pos.Valid {
		p.PlayerPosition = &pos.String
	}
	if source.Valid {
		p.Source = &source.String
	}
	if metaJSON.Valid {
		json.Unmarshal([]byte(metaJSON.String), &p.Metadata)
	}

	return p, nil
}

var validPlayerSortColumns = map[string]bool{
	"player_name":     true,
	"team":            true,
	"player_position": true,
}

func buildPlayerOrderBy(sort, order string) string {
	if sort != "" && validPlayerSortColumns[sort] {
		dir := "ASC"
		if order == "desc" {
			dir = "DESC"
		}
		return " ORDER BY " + sort + " " + dir + " NULLS LAST"
	}
	return " ORDER BY player_name ASC"
}
