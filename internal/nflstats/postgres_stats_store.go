package nflstats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PostgresStatStore implements StatStore.
type PostgresStatStore struct {
	db *sql.DB
}

// NewPostgresStatStore creates a stat store backed by the given DB.
func NewPostgresStatStore(db *sql.DB) *PostgresStatStore {
	return &PostgresStatStore{db: db}
}

// --------------------------------------------------------------------------
// ListStats — filtered, paginated
// --------------------------------------------------------------------------

func (s *PostgresStatStore) ListStats(ctx context.Context, f StatFilter, offset, limit int) ([]PlayerStat, int, error) {
	where, args := buildStatWhere(f)

	// Count
	var total int
	countSQL := "SELECT COUNT(*) FROM player_stats" + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count player_stats: %w", err)
	}

	// Query
	orderBy := buildStatOrderBy(f.Sort, f.Order)

	querySQL := fmt.Sprintf(
		`SELECT id, player_id, player_name, player_display_name, position,
		        position_group, team, season, week, stat_type, season_type, opponent_team,
		        completions, attempts, passing_yards, passing_tds, interceptions,
		        sacks, sack_yards, passing_air_yards, passing_yards_after_catch,
		        passing_2pt_conversions,
		        carries, rushing_yards, rushing_tds, rushing_fumbles,
		        rushing_fumbles_lost, rushing_2pt_conversions,
		        receptions, targets, receiving_yards, receiving_tds,
		        receiving_fumbles, receiving_fumbles_lost,
		        receiving_air_yards, receiving_yards_after_catch,
		        receiving_2pt_conversions,
		        fantasy_points, fantasy_points_ppr, special_teams_tds,
		        source, created_at
		 FROM player_stats%s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderBy, len(args)+1, len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query player_stats: %w", err)
	}
	defer rows.Close()

	var stats []PlayerStat
	for rows.Next() {
		st, err := scanStatRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan stat: %w", err)
		}
		stats = append(stats, st)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stats: %w", err)
	}

	return stats, total, nil
}

// --------------------------------------------------------------------------
// GetLeaders — top N by a stat column
// --------------------------------------------------------------------------

// validStatColumns is a whitelist of columns allowed for leader queries.
var validStatColumns = map[string]bool{
	"passing_yards": true, "passing_tds": true, "rushing_yards": true,
	"rushing_tds": true, "receiving_yards": true, "receiving_tds": true,
	"receptions": true, "targets": true, "carries": true,
	"fantasy_points": true, "fantasy_points_ppr": true,
	"interceptions": true, "sacks": true, "completions": true, "attempts": true,
}

func (s *PostgresStatStore) GetLeaders(ctx context.Context, stat string, season, week int, position string, limit int) ([]PlayerStat, error) {
	if !validStatColumns[stat] {
		return nil, fmt.Errorf("invalid stat column: %s", stat)
	}

	var conditions []string
	var args []any

	args = append(args, season)
	conditions = append(conditions, fmt.Sprintf("season = $%d", len(args)))

	if week > 0 {
		args = append(args, week)
		conditions = append(conditions, fmt.Sprintf("week = $%d", len(args)))
	}
	if position != "" {
		args = append(args, position)
		conditions = append(conditions, fmt.Sprintf("position = $%d", len(args)))
	}

	// Default to stat_type = 'actual' for leader queries
	args = append(args, "actual")
	conditions = append(conditions, fmt.Sprintf("stat_type = $%d", len(args)))

	where := " WHERE " + strings.Join(conditions, " AND ")

	querySQL := fmt.Sprintf(
		`SELECT id, player_id, player_name, player_display_name, position,
		        position_group, team, season, week, stat_type, season_type, opponent_team,
		        completions, attempts, passing_yards, passing_tds, interceptions,
		        sacks, sack_yards, passing_air_yards, passing_yards_after_catch,
		        passing_2pt_conversions,
		        carries, rushing_yards, rushing_tds, rushing_fumbles,
		        rushing_fumbles_lost, rushing_2pt_conversions,
		        receptions, targets, receiving_yards, receiving_tds,
		        receiving_fumbles, receiving_fumbles_lost,
		        receiving_air_yards, receiving_yards_after_catch,
		        receiving_2pt_conversions,
		        fantasy_points, fantasy_points_ppr, special_teams_tds,
		        source, created_at
		 FROM player_stats%s
		 ORDER BY %s DESC NULLS LAST
		 LIMIT $%d`,
		where, stat, len(args)+1,
	)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query leaders: %w", err)
	}
	defer rows.Close()

	var stats []PlayerStat
	for rows.Next() {
		st, err := scanStatRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan leader: %w", err)
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func buildStatWhere(f StatFilter) (string, []any) {
	var conditions []string
	var args []any

	if f.PlayerID != nil {
		args = append(args, *f.PlayerID)
		conditions = append(conditions, fmt.Sprintf("player_id = $%d", len(args)))
	}
	if f.Team != nil {
		args = append(args, *f.Team)
		conditions = append(conditions, fmt.Sprintf("team = $%d", len(args)))
	}
	if f.Position != nil {
		args = append(args, *f.Position)
		conditions = append(conditions, fmt.Sprintf("position = $%d", len(args)))
	}
	if f.Season != nil {
		args = append(args, *f.Season)
		conditions = append(conditions, fmt.Sprintf("season = $%d", len(args)))
	}
	if f.Week != nil {
		args = append(args, *f.Week)
		conditions = append(conditions, fmt.Sprintf("week = $%d", len(args)))
	}
	if f.StatType != nil {
		args = append(args, *f.StatType)
		conditions = append(conditions, fmt.Sprintf("stat_type = $%d", len(args)))
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

func scanStatRow(rows *sql.Rows) (PlayerStat, error) {
	var st PlayerStat
	var playerID, displayName, pos, posGroup, team sql.NullString
	var statType, seasonType, opponentTeam, source sql.NullString
	var completions, attempts, passingTds, interceptions sql.NullInt64
	var passing2pt, carries, rushingTds, rushingFumbles, rushingFumblesLost sql.NullInt64
	var rushing2pt, receptions, targets, receivingTds sql.NullInt64
	var receivingFumbles, receivingFumblesLost, receiving2pt, specialTeamsTds sql.NullInt64
	var passingYards, sacks, sackYards, passingAirYards, passYAC sql.NullFloat64
	var rushingYards, receivingYards, recAirYards, recYAC sql.NullFloat64
	var fantasyPts, fantasyPtsPPR sql.NullFloat64

	err := rows.Scan(
		&st.ID, &playerID, &st.PlayerName, &displayName, &pos,
		&posGroup, &team, &st.Season, &st.Week, &statType, &seasonType, &opponentTeam,
		&completions, &attempts, &passingYards, &passingTds, &interceptions,
		&sacks, &sackYards, &passingAirYards, &passYAC,
		&passing2pt,
		&carries, &rushingYards, &rushingTds, &rushingFumbles,
		&rushingFumblesLost, &rushing2pt,
		&receptions, &targets, &receivingYards, &receivingTds,
		&receivingFumbles, &receivingFumblesLost,
		&recAirYards, &recYAC,
		&receiving2pt,
		&fantasyPts, &fantasyPtsPPR, &specialTeamsTds,
		&source, &st.CreatedAt,
	)
	if err != nil {
		return PlayerStat{}, err
	}

	// Map nullable fields
	if playerID.Valid {
		st.PlayerID = &playerID.String
	}
	if displayName.Valid {
		st.PlayerDisplayName = &displayName.String
	}
	if pos.Valid {
		st.Position = &pos.String
	}
	if posGroup.Valid {
		st.PositionGroup = &posGroup.String
	}
	if team.Valid {
		st.Team = &team.String
	}
	if statType.Valid {
		st.StatType = &statType.String
	}
	if seasonType.Valid {
		st.SeasonType = &seasonType.String
	}
	if opponentTeam.Valid {
		st.OpponentTeam = &opponentTeam.String
	}
	if source.Valid {
		st.Source = &source.String
	}

	// Ints
	setIntPtr(&st.Completions, completions)
	setIntPtr(&st.Attempts, attempts)
	setIntPtr(&st.PassingTds, passingTds)
	setIntPtr(&st.Interceptions, interceptions)
	setIntPtr(&st.Passing2ptConversions, passing2pt)
	setIntPtr(&st.Carries, carries)
	setIntPtr(&st.RushingTds, rushingTds)
	setIntPtr(&st.RushingFumbles, rushingFumbles)
	setIntPtr(&st.RushingFumblesLost, rushingFumblesLost)
	setIntPtr(&st.Rushing2ptConversions, rushing2pt)
	setIntPtr(&st.Receptions, receptions)
	setIntPtr(&st.Targets, targets)
	setIntPtr(&st.ReceivingTds, receivingTds)
	setIntPtr(&st.ReceivingFumbles, receivingFumbles)
	setIntPtr(&st.ReceivingFumblesLost, receivingFumblesLost)
	setIntPtr(&st.Receiving2ptConversions, receiving2pt)
	setIntPtr(&st.SpecialTeamsTds, specialTeamsTds)

	// Floats
	setFloat64Ptr(&st.PassingYards, passingYards)
	setFloat64Ptr(&st.Sacks, sacks)
	setFloat64Ptr(&st.SackYards, sackYards)
	setFloat64Ptr(&st.PassingAirYards, passingAirYards)
	setFloat64Ptr(&st.PassingYardsAfterCatch, passYAC)
	setFloat64Ptr(&st.RushingYards, rushingYards)
	setFloat64Ptr(&st.ReceivingYards, receivingYards)
	setFloat64Ptr(&st.ReceivingAirYards, recAirYards)
	setFloat64Ptr(&st.ReceivingYardsAfterCatch, recYAC)
	setFloat64Ptr(&st.FantasyPoints, fantasyPts)
	setFloat64Ptr(&st.FantasyPointsPPR, fantasyPtsPPR)

	return st, nil
}

// setIntPtr converts a sql.NullInt64 to *int.
func setIntPtr(dst **int, src sql.NullInt64) {
	if src.Valid {
		v := int(src.Int64)
		*dst = &v
	}
}

// setFloat64Ptr converts a sql.NullFloat64 to *float64.
func setFloat64Ptr(dst **float64, src sql.NullFloat64) {
	if src.Valid {
		*dst = &src.Float64
	}
}

// Valid sort columns for the player_stats table.
var validStatSortColumns = map[string]bool{
	"player_name": true, "team": true, "position": true,
	"season": true, "week": true,
	"passing_yards": true, "passing_tds": true,
	"rushing_yards": true, "rushing_tds": true,
	"receiving_yards": true, "receiving_tds": true,
	"receptions": true, "targets": true, "carries": true,
	"fantasy_points": true, "fantasy_points_ppr": true,
	"completions": true, "attempts": true,
	"interceptions": true, "sacks": true,
}

func buildStatOrderBy(sort, order string) string {
	if sort != "" && validStatSortColumns[sort] {
		dir := "DESC"
		if order == "asc" || order == "ASC" {
			dir = "ASC"
		}
		return fmt.Sprintf("ORDER BY %s %s NULLS LAST", sort, dir)
	}
	return "ORDER BY season DESC, week DESC, fantasy_points_ppr DESC NULLS LAST"
}
