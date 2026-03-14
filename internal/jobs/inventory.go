package jobs

import (
	"context"
	"database/sql"
	"fmt"
)

// InventoryRow represents one row in the data inventory overview.
type InventoryRow struct {
	Source          string  `json:"source"`
	Table           string  `json:"table"`
	Season          *int    `json:"season,omitempty"`
	StatType        *string `json:"stat_type,omitempty"`
	SeasonType      *string `json:"season_type,omitempty"`
	RankType        *string `json:"rank_type,omitempty"`
	Rows            int     `json:"rows"`
	DistinctPlayers int     `json:"distinct_players"`
	MinWeek         *int    `json:"min_week,omitempty"`
	MaxWeek         *int    `json:"max_week,omitempty"`
	WeekCount       *int    `json:"week_count,omitempty"`
	LastUpdated     string  `json:"last_updated"`
}

// InventoryResponse is the full data inventory payload.
type InventoryResponse struct {
	Stats    []InventoryRow  `json:"stats"`
	Players  []InventoryRow  `json:"players"`
	Games    []InventoryRow  `json:"games"`
	Rankings []InventoryRow  `json:"rankings"`
	Totals   InventoryTotals `json:"totals"`
}

// InventoryTotals holds grand totals across all tables.
type InventoryTotals struct {
	Players  int `json:"players"`
	Stats    int `json:"stats"`
	Games    int `json:"games"`
	Rankings int `json:"rankings"`
}

// AuditResult holds all the checks for one audit run.
type AuditResult struct {
	Duplicates      []AuditDuplicate      `json:"duplicates"`
	Completeness    []AuditCompleteness   `json:"completeness"`
	PlayerCoverage  []AuditPlayerCoverage `json:"player_coverage"`
	RankingCoverage *AuditRankingCoverage `json:"ranking_coverage,omitempty"`
}

// AuditDuplicate reports a (season, week) with duplicate rows.
type AuditDuplicate struct {
	Season     int    `json:"season"`
	Week       int    `json:"week"`
	StatType   string `json:"stat_type"`
	Source     string `json:"source"`
	Duplicates int    `json:"duplicates"`
}

// AuditCompleteness reports expected vs actual week counts per season.
type AuditCompleteness struct {
	Season        int `json:"season"`
	ExpectedWeeks int `json:"expected_weeks"`
	ActualWeeks   int `json:"actual_weeks"`
	MissingWeeks  int `json:"missing_weeks"`
}

// AuditPlayerCoverage reports how many rostered players have no stats.
type AuditPlayerCoverage struct {
	Season           int `json:"season"`
	RosteredPlayers  int `json:"rostered_players"`
	PlayersWithStats int `json:"players_with_stats"`
	MissingStats     int `json:"missing_stats"`
}

// AuditRankingCoverage reports player_id resolution rate in fantasy_rankings.
type AuditRankingCoverage struct {
	TotalRankings     int     `json:"total_rankings"`
	ResolvedPlayers   int     `json:"resolved_players"`
	UnresolvedPlayers int     `json:"unresolved_players"`
	ResolutionPct     float64 `json:"resolution_pct"`
}

// InventoryStore extends Store with inventory/audit queries.
type InventoryStore interface {
	GetInventory(ctx context.Context, filter InventoryFilter) (*InventoryResponse, error)
	RunAudit(ctx context.Context, table string, season int) (*AuditResult, error)
}

// PostgresInventoryStore implements InventoryStore.
type PostgresInventoryStore struct {
	db *sql.DB
}

// NewPostgresInventoryStore creates an inventory store.
func NewPostgresInventoryStore(db *sql.DB) *PostgresInventoryStore {
	return &PostgresInventoryStore{db: db}
}

// GetInventory returns a summary of all data currently in the database.
func (s *PostgresInventoryStore) GetInventory(ctx context.Context, filter InventoryFilter) (*InventoryResponse, error) {
	resp := &InventoryResponse{
		Stats:    make([]InventoryRow, 0),
		Players:  make([]InventoryRow, 0),
		Games:    make([]InventoryRow, 0),
		Rankings: make([]InventoryRow, 0),
	}

	// --- player_stats breakdown ---
	{
		where := "WHERE 1=1"
		args := []any{}
		if filter.Source != "" {
			args = append(args, filter.Source)
			where += fmt.Sprintf(" AND source = $%d", len(args))
		}
		if filter.Season > 0 {
			args = append(args, filter.Season)
			where += fmt.Sprintf(" AND season = $%d", len(args))
		}
		if filter.StatType != "" {
			args = append(args, filter.StatType)
			where += fmt.Sprintf(" AND stat_type = $%d", len(args))
		}
		if filter.SeasonType != "" {
			args = append(args, filter.SeasonType)
			where += fmt.Sprintf(" AND season_type = $%d", len(args))
		}
		rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(source, 'unknown'), season, stat_type, season_type,
			       COUNT(*) AS rows,
			       COUNT(DISTINCT player_id) AS distinct_players,
			       MIN(week) AS min_week,
			       MAX(week) AS max_week,
			       COUNT(DISTINCT week) AS week_count,
			       MAX(created_at)::text AS last_updated
			FROM player_stats %s
			GROUP BY source, season, stat_type, season_type
			ORDER BY season DESC, stat_type, season_type
		`, where), args...)
		if err != nil {
			return nil, fmt.Errorf("inventory player_stats: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var r InventoryRow
			var statType, seasonType sql.NullString
			var minWeek, maxWeek, weekCount sql.NullInt64
			r.Table = "player_stats"
			err := rows.Scan(&r.Source, &r.Season, &statType, &seasonType,
				&r.Rows, &r.DistinctPlayers, &minWeek, &maxWeek, &weekCount, &r.LastUpdated)
			if err != nil {
				return nil, fmt.Errorf("scan stats inventory: %w", err)
			}
			if statType.Valid {
				r.StatType = &statType.String
			}
			if seasonType.Valid {
				r.SeasonType = &seasonType.String
			}
			if minWeek.Valid {
				v := int(minWeek.Int64)
				r.MinWeek = &v
			}
			if maxWeek.Valid {
				v := int(maxWeek.Int64)
				r.MaxWeek = &v
			}
			if weekCount.Valid {
				v := int(weekCount.Int64)
				r.WeekCount = &v
			}
			resp.Stats = append(resp.Stats, r)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate stats inventory: %w", err)
		}
	}

	// --- players breakdown ---
	{
		where := "WHERE 1=1"
		args := []any{}
		if filter.Source != "" {
			args = append(args, filter.Source)
			where += fmt.Sprintf(" AND source = $%d", len(args))
		}
		pRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(source, 'unknown'),
			       COUNT(*) AS rows,
			       COUNT(DISTINCT player_id) AS distinct_players,
			       MAX(updated_at)::text AS last_updated
			FROM players %s
			GROUP BY source
			ORDER BY source
		`, where), args...)
		if err != nil {
			return nil, fmt.Errorf("inventory players: %w", err)
		}
		defer pRows.Close()
		for pRows.Next() {
			var r InventoryRow
			r.Table = "players"
			err := pRows.Scan(&r.Source, &r.Rows, &r.DistinctPlayers, &r.LastUpdated)
			if err != nil {
				return nil, fmt.Errorf("scan players inventory: %w", err)
			}
			resp.Players = append(resp.Players, r)
		}
	}

	// --- games breakdown ---
	{
		where := "WHERE 1=1"
		args := []any{}
		if filter.Source != "" {
			args = append(args, filter.Source)
			where += fmt.Sprintf(" AND source = $%d", len(args))
		}
		if filter.Season > 0 {
			args = append(args, filter.Season)
			where += fmt.Sprintf(" AND season = $%d", len(args))
		}
		gRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(source, 'unknown'), season,
			       COUNT(*) AS rows,
			       0 AS distinct_players,
			       MIN(week) AS min_week,
			       MAX(week) AS max_week,
			       COUNT(DISTINCT week) AS week_count,
			       MAX(created_at)::text AS last_updated
			FROM games %s
			GROUP BY source, season
			ORDER BY season DESC
		`, where), args...)
		if err != nil {
			return nil, fmt.Errorf("inventory games: %w", err)
		}
		defer gRows.Close()
		for gRows.Next() {
			var r InventoryRow
			var minWeek, maxWeek, weekCount sql.NullInt64
			r.Table = "games"
			err := gRows.Scan(&r.Source, &r.Season, &r.Rows, &r.DistinctPlayers,
				&minWeek, &maxWeek, &weekCount, &r.LastUpdated)
			if err != nil {
				return nil, fmt.Errorf("scan games inventory: %w", err)
			}
			if minWeek.Valid {
				v := int(minWeek.Int64)
				r.MinWeek = &v
			}
			if maxWeek.Valid {
				v := int(maxWeek.Int64)
				r.MaxWeek = &v
			}
			if weekCount.Valid {
				v := int(weekCount.Int64)
				r.WeekCount = &v
			}
			resp.Games = append(resp.Games, r)
		}
	}

	// --- fantasy_rankings breakdown ---
	{
		where := "WHERE 1=1"
		args := []any{}
		if filter.Source != "" {
			args = append(args, filter.Source)
			where += fmt.Sprintf(" AND source = $%d", len(args))
		}
		if filter.Season > 0 {
			args = append(args, filter.Season)
			where += fmt.Sprintf(" AND season = $%d", len(args))
		}
		if filter.RankType != "" {
			args = append(args, filter.RankType)
			where += fmt.Sprintf(" AND rank_type = $%d", len(args))
		}
		rRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(source, 'unknown'), season,
			       rank_type,
			       COUNT(*) AS rows,
			       COUNT(DISTINCT player_id) AS distinct_players,
			       MAX(created_at)::text AS last_updated
			FROM fantasy_rankings %s
			GROUP BY source, season, rank_type
			ORDER BY season DESC, rank_type
		`, where), args...)
		if err != nil {
			return nil, fmt.Errorf("inventory rankings: %w", err)
		}
		defer rRows.Close()
		for rRows.Next() {
			var r InventoryRow
			var rankType sql.NullString
			r.Table = "fantasy_rankings"
			err := rRows.Scan(&r.Source, &r.Season, &rankType, &r.Rows, &r.DistinctPlayers, &r.LastUpdated)
			if err != nil {
				return nil, fmt.Errorf("scan rankings inventory: %w", err)
			}
			if rankType.Valid {
				r.RankType = &rankType.String
			}
			resp.Rankings = append(resp.Rankings, r)
		}
	}

	// --- Grand totals (unfiltered, always show full DB stats) ---
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players`).Scan(&resp.Totals.Players)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_stats`).Scan(&resp.Totals.Stats)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM games`).Scan(&resp.Totals.Games)
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fantasy_rankings`).Scan(&resp.Totals.Rankings)

	return resp, nil
}

// RunAudit runs data quality checks. If season > 0, scopes checks to that season.
func (s *PostgresInventoryStore) RunAudit(ctx context.Context, table string, season int) (*AuditResult, error) {
	result := &AuditResult{
		Duplicates:     make([]AuditDuplicate, 0),
		Completeness:   make([]AuditCompleteness, 0),
		PlayerCoverage: make([]AuditPlayerCoverage, 0),
	}

	// --- 1. Duplicate check ---
	dupQuery := `
		SELECT season, week, stat_type, source, COUNT(*) - COUNT(DISTINCT (player_id, season, week, stat_type, source)) AS duplicates
		FROM player_stats
		WHERE 1=1
	`
	var dupArgs []any
	if season > 0 {
		dupArgs = append(dupArgs, season)
		dupQuery += fmt.Sprintf(" AND season = $%d", len(dupArgs))
	}
	dupQuery += `
		GROUP BY season, week, stat_type, source
		HAVING COUNT(*) > COUNT(DISTINCT (player_id, season, week, stat_type, source))
		ORDER BY season DESC, week
	`

	dupRows, err := s.db.QueryContext(ctx, dupQuery, dupArgs...)
	if err != nil {
		return nil, fmt.Errorf("audit duplicates: %w", err)
	}
	defer dupRows.Close()
	for dupRows.Next() {
		var d AuditDuplicate
		if err := dupRows.Scan(&d.Season, &d.Week, &d.StatType, &d.Source, &d.Duplicates); err != nil {
			return nil, fmt.Errorf("scan duplicate: %w", err)
		}
		result.Duplicates = append(result.Duplicates, d)
	}

	// --- 2. Completeness check (expected weeks per season) ---
	compQuery := `
		SELECT season,
		       CASE
		           WHEN season >= 2021 THEN 18
		           ELSE 17
		       END AS expected_weeks,
		       COUNT(DISTINCT week) AS actual_weeks
		FROM player_stats
		WHERE stat_type = 'actual' AND week > 0
	`
	var compArgs []any
	if season > 0 {
		compArgs = append(compArgs, season)
		compQuery += fmt.Sprintf(" AND season = $%d", len(compArgs))
	}
	compQuery += `
		GROUP BY season
		ORDER BY season DESC
	`

	compRows, err := s.db.QueryContext(ctx, compQuery, compArgs...)
	if err != nil {
		return nil, fmt.Errorf("audit completeness: %w", err)
	}
	defer compRows.Close()
	for compRows.Next() {
		var c AuditCompleteness
		if err := compRows.Scan(&c.Season, &c.ExpectedWeeks, &c.ActualWeeks); err != nil {
			return nil, fmt.Errorf("scan completeness: %w", err)
		}
		c.MissingWeeks = c.ExpectedWeeks - c.ActualWeeks
		if c.MissingWeeks < 0 {
			c.MissingWeeks = 0
		}
		result.Completeness = append(result.Completeness, c)
	}

	// --- 3. Player coverage (rostered players without stats) ---
	covQuery := `
		SELECT ps.season,
		       COUNT(DISTINCT p.player_id) AS rostered,
		       COUNT(DISTINCT s.player_id) AS with_stats,
		       COUNT(DISTINCT p.player_id) - COUNT(DISTINCT s.player_id) AS missing
		FROM players p
		CROSS JOIN (SELECT DISTINCT season FROM player_stats WHERE stat_type = 'actual' AND week > 0) ps
		LEFT JOIN player_stats s ON s.player_id = p.player_id AND s.season = ps.season AND s.stat_type = 'actual'
	`
	var covArgs []any
	if season > 0 {
		covArgs = append(covArgs, season)
		covQuery += fmt.Sprintf(" WHERE ps.season = $%d", len(covArgs))
	}
	covQuery += `
		GROUP BY ps.season
		ORDER BY ps.season DESC
	`

	covRows, err := s.db.QueryContext(ctx, covQuery, covArgs...)
	if err != nil {
		return nil, fmt.Errorf("audit player coverage: %w", err)
	}
	defer covRows.Close()
	for covRows.Next() {
		var c AuditPlayerCoverage
		if err := covRows.Scan(&c.Season, &c.RosteredPlayers, &c.PlayersWithStats, &c.MissingStats); err != nil {
			return nil, fmt.Errorf("scan coverage: %w", err)
		}
		result.PlayerCoverage = append(result.PlayerCoverage, c)
	}

	// --- 4. Ranking resolution ---
	var rc AuditRankingCoverage
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) AS total,
		       COUNT(player_id) AS resolved,
		       COUNT(*) - COUNT(player_id) AS unresolved,
		       CASE WHEN COUNT(*) > 0
		            THEN ROUND(COUNT(player_id)::numeric / COUNT(*)::numeric * 100, 1)
		            ELSE 0 END AS pct
		FROM fantasy_rankings
	`).Scan(&rc.TotalRankings, &rc.ResolvedPlayers, &rc.UnresolvedPlayers, &rc.ResolutionPct)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("audit rankings: %w", err)
	}
	result.RankingCoverage = &rc

	return result, nil
}
