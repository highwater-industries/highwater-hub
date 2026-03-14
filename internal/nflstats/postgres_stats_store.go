package nflstats

import (
	"context"
	"database/sql"
	"encoding/json"
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
	countSQL := "SELECT COUNT(*) FROM player_stats ps" + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count player_stats: %w", err)
	}

	// Query
	orderBy := buildStatOrderBy(f.Sort, f.Order)

	querySQL := fmt.Sprintf(
		`SELECT ps.id, p.id AS player_db_id,
		        ps.player_id, ps.player_name, ps.player_display_name, ps.position,
		        ps.position_group, ps.team, ps.season, ps.week, ps.stat_type, ps.season_type, ps.opponent_team,
		        ps.completions, ps.attempts, ps.passing_yards, ps.passing_tds, ps.interceptions,
		        ps.sacks, ps.sack_yards, ps.passing_air_yards, ps.passing_yards_after_catch,
		        ps.passing_2pt_conversions,
		        ps.carries, ps.rushing_yards, ps.rushing_tds, ps.rushing_fumbles,
		        ps.rushing_fumbles_lost, ps.rushing_2pt_conversions,
		        ps.receptions, ps.targets, ps.receiving_yards, ps.receiving_tds,
		        ps.receiving_fumbles, ps.receiving_fumbles_lost,
		        ps.receiving_air_yards, ps.receiving_yards_after_catch,
		        ps.receiving_2pt_conversions,
		        ps.fantasy_points, ps.fantasy_points_ppr, ps.special_teams_tds,
		        ps.source, ps.created_at
		 FROM player_stats ps
		 LEFT JOIN players p ON ps.player_id = p.player_id%s
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

	stats := make([]PlayerStat, 0)
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
	conditions = append(conditions, fmt.Sprintf("ps.season = $%d", len(args)))

	if week > 0 {
		args = append(args, week)
		conditions = append(conditions, fmt.Sprintf("ps.week = $%d", len(args)))
	}
	if position != "" {
		args = append(args, position)
		conditions = append(conditions, fmt.Sprintf("ps.position = $%d", len(args)))
	}

	// Default to stat_type = 'actual' for leader queries
	args = append(args, "actual")
	conditions = append(conditions, fmt.Sprintf("ps.stat_type = $%d", len(args)))

	where := " WHERE " + strings.Join(conditions, " AND ")

	querySQL := fmt.Sprintf(
		`SELECT ps.id, p.id AS player_db_id,
		        ps.player_id, ps.player_name, ps.player_display_name, ps.position,
		        ps.position_group, ps.team, ps.season, ps.week, ps.stat_type, ps.season_type, ps.opponent_team,
		        ps.completions, ps.attempts, ps.passing_yards, ps.passing_tds, ps.interceptions,
		        ps.sacks, ps.sack_yards, ps.passing_air_yards, ps.passing_yards_after_catch,
		        ps.passing_2pt_conversions,
		        ps.carries, ps.rushing_yards, ps.rushing_tds, ps.rushing_fumbles,
		        ps.rushing_fumbles_lost, ps.rushing_2pt_conversions,
		        ps.receptions, ps.targets, ps.receiving_yards, ps.receiving_tds,
		        ps.receiving_fumbles, ps.receiving_fumbles_lost,
		        ps.receiving_air_yards, ps.receiving_yards_after_catch,
		        ps.receiving_2pt_conversions,
		        ps.fantasy_points, ps.fantasy_points_ppr, ps.special_teams_tds,
		        ps.source, ps.created_at
		 FROM player_stats ps
		 LEFT JOIN players p ON ps.player_id = p.player_id%s
		 ORDER BY ps.%s DESC NULLS LAST
		 LIMIT $%d`,
		where, stat, len(args)+1,
	)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query leaders: %w", err)
	}
	defer rows.Close()

	stats := make([]PlayerStat, 0)
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
		conditions = append(conditions, fmt.Sprintf("ps.player_id = $%d", len(args)))
	}
	if f.Team != nil {
		args = append(args, *f.Team)
		conditions = append(conditions, fmt.Sprintf("ps.team = $%d", len(args)))
	}
	if f.Position != nil {
		args = append(args, *f.Position)
		conditions = append(conditions, fmt.Sprintf("ps.position = $%d", len(args)))
	}
	if f.Season != nil {
		args = append(args, *f.Season)
		conditions = append(conditions, fmt.Sprintf("ps.season = $%d", len(args)))
	}
	if f.Week != nil {
		args = append(args, *f.Week)
		conditions = append(conditions, fmt.Sprintf("ps.week = $%d", len(args)))
	}
	if f.StatType != nil {
		args = append(args, *f.StatType)
		conditions = append(conditions, fmt.Sprintf("ps.stat_type = $%d", len(args)))
	}
	if f.Source != nil {
		args = append(args, *f.Source)
		conditions = append(conditions, fmt.Sprintf("ps.source = $%d", len(args)))
	}
	if f.Search != nil {
		args = append(args, "%"+*f.Search+"%")
		conditions = append(conditions, fmt.Sprintf("ps.player_name ILIKE $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanStatRow(rows *sql.Rows) (PlayerStat, error) {
	var st PlayerStat
	var playerDbID sql.NullInt64
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
		&st.ID, &playerDbID, &playerID, &st.PlayerName, &displayName, &pos,
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
	if playerDbID.Valid {
		v := int(playerDbID.Int64)
		st.PlayerDbID = &v
	}
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
		return fmt.Sprintf("ORDER BY ps.%s %s NULLS LAST", sort, dir)
	}
	return "ORDER BY ps.season DESC, ps.week DESC, ps.fantasy_points_ppr DESC NULLS LAST"
}

// --------------------------------------------------------------------------
// GetPlayerSummary — aggregated career + season + recent game log
// --------------------------------------------------------------------------

func (s *PostgresStatStore) GetPlayerSummary(ctx context.Context, playerDBID int) (*PlayerSummary, error) {
	// 1. Fetch the player record
	row := s.db.QueryRowContext(ctx,
		`SELECT id, player_id, player_name, team, player_position,
		        source, metadata, created_at, updated_at
		 FROM players WHERE id = $1`, playerDBID,
	)
	player, err := scanPlayerFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	// We need the external player_id to query stats (stats use the NFL ID string)
	if player.PlayerID == nil {
		// Player exists but has no NFL ID — return empty summary
		return &PlayerSummary{
			Player:      player,
			Seasons:     []SeasonTotals{},
			RecentGames: []PlayerStat{},
			Rankings:    []FantasyRanking{},
		}, nil
	}
	nflID := *player.PlayerID

	// 2. Season-by-season aggregation
	seasons, err := s.querySeasonTotals(ctx, nflID)
	if err != nil {
		return nil, fmt.Errorf("season totals: %w", err)
	}

	// 3. Career totals (aggregate of all seasons)
	career, err := s.queryCareerTotals(ctx, nflID)
	if err != nil {
		return nil, fmt.Errorf("career totals: %w", err)
	}

	// 4. Recent game log (last 10 weekly stat lines)
	recentGames, err := s.queryRecentGames(ctx, nflID, 10)
	if err != nil {
		return nil, fmt.Errorf("recent games: %w", err)
	}

	// 5. Fantasy rankings for this player
	rankings, err := s.queryPlayerRankings(ctx, nflID)
	if err != nil {
		return nil, fmt.Errorf("player rankings: %w", err)
	}

	return &PlayerSummary{
		Player:       player,
		CareerTotals: career,
		Seasons:      seasons,
		RecentGames:  recentGames,
		Rankings:     rankings,
	}, nil
}

// scanPlayerFromRow scans a player from *sql.Row (used within stat store).
func scanPlayerFromRow(row *sql.Row) (Player, error) {
	var p Player
	var playerID, team, pos, source sql.NullString
	var metaJSON sql.NullString

	err := row.Scan(
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
		var m map[string]any
		if jsonErr := json.Unmarshal([]byte(metaJSON.String), &m); jsonErr == nil {
			p.Metadata = m
		}
	}

	return p, nil
}

func (s *PostgresStatStore) querySeasonTotals(ctx context.Context, nflID string) ([]SeasonTotals, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT season,
		       COUNT(*)                     AS games_played,
		       SUM(completions)             AS completions,
		       SUM(attempts)                AS attempts,
		       SUM(passing_yards)           AS passing_yards,
		       SUM(passing_tds)             AS passing_tds,
		       SUM(interceptions)           AS interceptions,
		       SUM(carries)                 AS carries,
		       SUM(rushing_yards)           AS rushing_yards,
		       SUM(rushing_tds)             AS rushing_tds,
		       SUM(receptions)              AS receptions,
		       SUM(targets)                 AS targets,
		       SUM(receiving_yards)         AS receiving_yards,
		       SUM(receiving_tds)           AS receiving_tds,
		       SUM(fantasy_points)          AS fantasy_points,
		       SUM(fantasy_points_ppr)      AS fantasy_points_ppr
		FROM player_stats
		WHERE player_id = $1 AND stat_type = 'actual'
		GROUP BY season
		ORDER BY season DESC
	`, nflID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SeasonTotals, 0)
	for rows.Next() {
		st, err := scanSeasonTotals(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

func (s *PostgresStatStore) queryCareerTotals(ctx context.Context, nflID string) (SeasonTotals, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT 0 AS season,
		       COUNT(*)                     AS games_played,
		       SUM(completions)             AS completions,
		       SUM(attempts)                AS attempts,
		       SUM(passing_yards)           AS passing_yards,
		       SUM(passing_tds)             AS passing_tds,
		       SUM(interceptions)           AS interceptions,
		       SUM(carries)                 AS carries,
		       SUM(rushing_yards)           AS rushing_yards,
		       SUM(rushing_tds)             AS rushing_tds,
		       SUM(receptions)              AS receptions,
		       SUM(targets)                 AS targets,
		       SUM(receiving_yards)         AS receiving_yards,
		       SUM(receiving_tds)           AS receiving_tds,
		       SUM(fantasy_points)          AS fantasy_points,
		       SUM(fantasy_points_ppr)      AS fantasy_points_ppr
		FROM player_stats
		WHERE player_id = $1 AND stat_type = 'actual'
	`, nflID)

	return scanSeasonTotalsSingle(row)
}

func (s *PostgresStatStore) queryRecentGames(ctx context.Context, nflID string, limit int) ([]PlayerStat, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, player_id, player_name, player_display_name, position,
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
		FROM player_stats
		WHERE player_id = $1 AND stat_type = 'actual'
		ORDER BY season DESC, week DESC
		LIMIT %d
	`, limit), nflID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]PlayerStat, 0)
	for rows.Next() {
		st, err := scanStatRowBasic(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

func (s *PostgresStatStore) queryPlayerRankings(ctx context.Context, nflID string) ([]FantasyRanking, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT fr.id, p.id AS player_db_id, fr.player_id, fr.player_name, fr.pos, fr.team, fr.rank, fr.ecr,
		       fr.sd, fr.best, fr.worst, fr.avg, fr.rank_type, fr.page_type,
		       fr.season, fr.week, fr.source, fr.created_at
		FROM fantasy_rankings fr
		LEFT JOIN players p ON fr.player_id = p.player_id
		WHERE fr.player_id = $1
		ORDER BY fr.season DESC, fr.week DESC NULLS LAST, fr.rank ASC NULLS LAST
	`, nflID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]FantasyRanking, 0)
	for rows.Next() {
		r, err := scanRankingRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// scanStatRowBasic scans a stat row WITHOUT the player_db_id JOIN column.
// Used by internal queries (e.g. queryRecentGames) that don't join players.
func scanStatRowBasic(rows *sql.Rows) (PlayerStat, error) {
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

// scanSeasonTotals scans a SeasonTotals row from *sql.Rows.
func scanSeasonTotals(rows *sql.Rows) (SeasonTotals, error) {
	var st SeasonTotals
	var completions, attempts, passingTds, interceptions sql.NullInt64
	var carries, rushingTds, receptions, targets, receivingTds sql.NullInt64
	var passingYards, rushingYards, receivingYards sql.NullFloat64
	var fantasyPts, fantasyPtsPPR sql.NullFloat64

	err := rows.Scan(
		&st.Season, &st.GamesPlayed,
		&completions, &attempts, &passingYards, &passingTds, &interceptions,
		&carries, &rushingYards, &rushingTds,
		&receptions, &targets, &receivingYards, &receivingTds,
		&fantasyPts, &fantasyPtsPPR,
	)
	if err != nil {
		return SeasonTotals{}, err
	}

	setIntPtr(&st.Completions, completions)
	setIntPtr(&st.Attempts, attempts)
	setIntPtr(&st.PassingTds, passingTds)
	setIntPtr(&st.Interceptions, interceptions)
	setIntPtr(&st.Carries, carries)
	setIntPtr(&st.RushingTds, rushingTds)
	setIntPtr(&st.Receptions, receptions)
	setIntPtr(&st.Targets, targets)
	setIntPtr(&st.ReceivingTds, receivingTds)
	setFloat64Ptr(&st.PassingYards, passingYards)
	setFloat64Ptr(&st.RushingYards, rushingYards)
	setFloat64Ptr(&st.ReceivingYards, receivingYards)
	setFloat64Ptr(&st.FantasyPoints, fantasyPts)
	setFloat64Ptr(&st.FantasyPointsPPR, fantasyPtsPPR)

	return st, nil
}

// scanSeasonTotalsSingle scans a single-row aggregation result.
func scanSeasonTotalsSingle(row *sql.Row) (SeasonTotals, error) {
	var st SeasonTotals
	var completions, attempts, passingTds, interceptions sql.NullInt64
	var carries, rushingTds, receptions, targets, receivingTds sql.NullInt64
	var passingYards, rushingYards, receivingYards sql.NullFloat64
	var fantasyPts, fantasyPtsPPR sql.NullFloat64

	err := row.Scan(
		&st.Season, &st.GamesPlayed,
		&completions, &attempts, &passingYards, &passingTds, &interceptions,
		&carries, &rushingYards, &rushingTds,
		&receptions, &targets, &receivingYards, &receivingTds,
		&fantasyPts, &fantasyPtsPPR,
	)
	if err != nil {
		return SeasonTotals{}, err
	}

	setIntPtr(&st.Completions, completions)
	setIntPtr(&st.Attempts, attempts)
	setIntPtr(&st.PassingTds, passingTds)
	setIntPtr(&st.Interceptions, interceptions)
	setIntPtr(&st.Carries, carries)
	setIntPtr(&st.RushingTds, rushingTds)
	setIntPtr(&st.Receptions, receptions)
	setIntPtr(&st.Targets, targets)
	setIntPtr(&st.ReceivingTds, receivingTds)
	setFloat64Ptr(&st.PassingYards, passingYards)
	setFloat64Ptr(&st.RushingYards, rushingYards)
	setFloat64Ptr(&st.ReceivingYards, receivingYards)
	setFloat64Ptr(&st.FantasyPoints, fantasyPts)
	setFloat64Ptr(&st.FantasyPointsPPR, fantasyPtsPPR)

	return st, nil
}
