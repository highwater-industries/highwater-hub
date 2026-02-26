package nflstats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PostgresGameStore implements GameStore.
type PostgresGameStore struct {
	db *sql.DB
}

// NewPostgresGameStore creates a game store backed by the given DB.
func NewPostgresGameStore(db *sql.DB) *PostgresGameStore {
	return &PostgresGameStore{db: db}
}

// --------------------------------------------------------------------------
// GetGame — single game by game_id
// --------------------------------------------------------------------------

func (s *PostgresGameStore) GetGame(ctx context.Context, gameID string) (Game, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, game_id, season, game_type, week, gameday, weekday,
		        gametime, away_team, home_team, away_score, home_score,
		        result, total, spread_line, total_line, overtime,
		        location, roof, surface, stadium, source, created_at
		 FROM games WHERE game_id = $1`, gameID,
	)
	return scanGameSingleRow(row)
}

// --------------------------------------------------------------------------
// ListGames — filtered, paginated
// --------------------------------------------------------------------------

func (s *PostgresGameStore) ListGames(ctx context.Context, f GameFilter, offset, limit int) ([]Game, int, error) {
	where, args := buildGameWhere(f)

	var total int
	countSQL := "SELECT COUNT(*) FROM games" + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count games: %w", err)
	}

	orderBy := buildGameOrderBy(f.Sort, f.Order)

	querySQL := fmt.Sprintf(
		`SELECT id, game_id, season, game_type, week, gameday, weekday,
		        gametime, away_team, home_team, away_score, home_score,
		        result, total, spread_line, total_line, overtime,
		        location, roof, surface, stadium, source, created_at
		 FROM games%s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderBy, len(args)+1, len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		g, err := scanGameRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan game: %w", err)
		}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate games: %w", err)
	}

	return games, total, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func buildGameWhere(f GameFilter) (string, []any) {
	var conditions []string
	var args []any

	if f.Season != nil {
		args = append(args, *f.Season)
		conditions = append(conditions, fmt.Sprintf("season = $%d", len(args)))
	}
	if f.Week != nil {
		args = append(args, *f.Week)
		conditions = append(conditions, fmt.Sprintf("week = $%d", len(args)))
	}
	if f.Team != nil {
		args = append(args, *f.Team)
		conditions = append(conditions, fmt.Sprintf("(home_team = $%d OR away_team = $%d)", len(args), len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanGameSingleRow(row *sql.Row) (Game, error) {
	var g Game
	var gameID, gameType, gameday, weekday, gametime sql.NullString
	var awayTeam, homeTeam, location, roof, surface, stadium, source sql.NullString
	var season, week, awayScore, homeScore, result, total sql.NullInt64
	var spreadLine, totalLine sql.NullFloat64
	var overtime sql.NullBool

	err := row.Scan(
		&g.ID, &gameID, &season, &gameType, &week, &gameday, &weekday,
		&gametime, &awayTeam, &homeTeam, &awayScore, &homeScore,
		&result, &total, &spreadLine, &totalLine, &overtime,
		&location, &roof, &surface, &stadium, &source, &g.CreatedAt,
	)
	if err != nil {
		return Game{}, fmt.Errorf("scan game: %w", err)
	}

	mapGameNullables(&g, gameID, season, gameType, week, gameday, weekday,
		gametime, awayTeam, homeTeam, awayScore, homeScore,
		result, total, spreadLine, totalLine, overtime,
		location, roof, surface, stadium, source)

	return g, nil
}

func scanGameRow(rows *sql.Rows) (Game, error) {
	var g Game
	var gameID, gameType, gameday, weekday, gametime sql.NullString
	var awayTeam, homeTeam, location, roof, surface, stadium, source sql.NullString
	var season, week, awayScore, homeScore, result, total sql.NullInt64
	var spreadLine, totalLine sql.NullFloat64
	var overtime sql.NullBool

	err := rows.Scan(
		&g.ID, &gameID, &season, &gameType, &week, &gameday, &weekday,
		&gametime, &awayTeam, &homeTeam, &awayScore, &homeScore,
		&result, &total, &spreadLine, &totalLine, &overtime,
		&location, &roof, &surface, &stadium, &source, &g.CreatedAt,
	)
	if err != nil {
		return Game{}, err
	}

	mapGameNullables(&g, gameID, season, gameType, week, gameday, weekday,
		gametime, awayTeam, homeTeam, awayScore, homeScore,
		result, total, spreadLine, totalLine, overtime,
		location, roof, surface, stadium, source)

	return g, nil
}

func mapGameNullables(g *Game,
	gameID sql.NullString,
	season sql.NullInt64,
	gameType sql.NullString,
	week sql.NullInt64,
	gameday, weekday, gametime sql.NullString,
	awayTeam, homeTeam sql.NullString,
	awayScore, homeScore, result, total sql.NullInt64,
	spreadLine, totalLine sql.NullFloat64,
	overtime sql.NullBool,
	location, roof, surface, stadium, source sql.NullString,
) {
	if gameID.Valid {
		g.GameID = &gameID.String
	}
	if gameType.Valid {
		g.GameType = &gameType.String
	}
	if gameday.Valid {
		g.Gameday = &gameday.String
	}
	if weekday.Valid {
		g.Weekday = &weekday.String
	}
	if gametime.Valid {
		g.Gametime = &gametime.String
	}
	if awayTeam.Valid {
		g.AwayTeam = &awayTeam.String
	}
	if homeTeam.Valid {
		g.HomeTeam = &homeTeam.String
	}
	if location.Valid {
		g.Location = &location.String
	}
	if roof.Valid {
		g.Roof = &roof.String
	}
	if surface.Valid {
		g.Surface = &surface.String
	}
	if stadium.Valid {
		g.Stadium = &stadium.String
	}
	if source.Valid {
		g.Source = &source.String
	}

	if season.Valid {
		v := int(season.Int64)
		g.Season = &v
	}
	if week.Valid {
		v := int(week.Int64)
		g.Week = &v
	}
	if awayScore.Valid {
		v := int(awayScore.Int64)
		g.AwayScore = &v
	}
	if homeScore.Valid {
		v := int(homeScore.Int64)
		g.HomeScore = &v
	}
	if result.Valid {
		v := int(result.Int64)
		g.Result = &v
	}
	if total.Valid {
		v := int(total.Int64)
		g.Total = &v
	}
	if spreadLine.Valid {
		g.SpreadLine = &spreadLine.Float64
	}
	if totalLine.Valid {
		g.TotalLine = &totalLine.Float64
	}
	if overtime.Valid {
		g.Overtime = &overtime.Bool
	}
}

var validGameSortColumns = map[string]bool{
	"season":     true,
	"week":       true,
	"gameday":    true,
	"away_team":  true,
	"home_team":  true,
	"away_score": true,
	"home_score": true,
}

func buildGameOrderBy(sort, order string) string {
	if sort != "" && validGameSortColumns[sort] {
		dir := "ASC"
		if order == "desc" {
			dir = "DESC"
		}
		return " ORDER BY " + sort + " " + dir + " NULLS LAST"
	}
	return " ORDER BY season DESC, week DESC"
}
