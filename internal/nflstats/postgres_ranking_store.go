package nflstats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PostgresRankingStore implements RankingStore.
type PostgresRankingStore struct {
	db *sql.DB
}

// NewPostgresRankingStore creates a ranking store backed by the given DB.
func NewPostgresRankingStore(db *sql.DB) *PostgresRankingStore {
	return &PostgresRankingStore{db: db}
}

// --------------------------------------------------------------------------
// ListRankings — filtered, paginated
// --------------------------------------------------------------------------

func (s *PostgresRankingStore) ListRankings(ctx context.Context, f RankingFilter, offset, limit int) ([]FantasyRanking, int, error) {
	where, args := buildRankingWhere(f)

	var total int
	countSQL := "SELECT COUNT(*) FROM fantasy_rankings fr" + where
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fantasy_rankings: %w", err)
	}

	orderBy := buildRankingOrderBy(f.Sort, f.Order)

	querySQL := fmt.Sprintf(
		`SELECT fr.id, p.id AS player_db_id, fr.player_id, fr.player_name, fr.pos, fr.team, fr.rank, fr.ecr,
		        fr.sd, fr.best, fr.worst, fr.avg, fr.rank_type, fr.page_type,
		        fr.season, fr.week, fr.source, fr.created_at
		 FROM fantasy_rankings fr
		 LEFT JOIN players p ON fr.player_id = p.player_id%s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderBy, len(args)+1, len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query fantasy_rankings: %w", err)
	}
	defer rows.Close()

	rankings := make([]FantasyRanking, 0)
	for rows.Next() {
		r, err := scanRankingRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan ranking: %w", err)
		}
		rankings = append(rankings, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate rankings: %w", err)
	}

	return rankings, total, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func buildRankingWhere(f RankingFilter) (string, []any) {
	var conditions []string
	var args []any

	if f.RankType != nil {
		args = append(args, *f.RankType)
		conditions = append(conditions, fmt.Sprintf("fr.rank_type = $%d", len(args)))
	}
	if f.Pos != nil {
		args = append(args, *f.Pos)
		conditions = append(conditions, fmt.Sprintf("fr.pos = $%d", len(args)))
	}
	if f.Team != nil {
		args = append(args, *f.Team)
		conditions = append(conditions, fmt.Sprintf("fr.team = $%d", len(args)))
	}
	if f.Search != nil {
		args = append(args, "%"+*f.Search+"%")
		conditions = append(conditions, fmt.Sprintf("fr.player_name ILIKE $%d", len(args)))
	}
	if f.Season != nil {
		args = append(args, *f.Season)
		conditions = append(conditions, fmt.Sprintf("fr.season = $%d", len(args)))
	}
	if f.Week != nil {
		args = append(args, *f.Week)
		conditions = append(conditions, fmt.Sprintf("fr.week = $%d", len(args)))
	}
	if f.Source != nil {
		args = append(args, *f.Source)
		conditions = append(conditions, fmt.Sprintf("fr.source = $%d", len(args)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanRankingRow(rows *sql.Rows) (FantasyRanking, error) {
	var r FantasyRanking
	var playerDbID sql.NullInt64
	var playerID, pos, team, rankType, pageType, source sql.NullString
	var rank, best, worst sql.NullInt64
	var season, week sql.NullInt64
	var ecr, sd, avg sql.NullFloat64

	err := rows.Scan(
		&r.ID, &playerDbID, &playerID, &r.PlayerName, &pos, &team, &rank, &ecr,
		&sd, &best, &worst, &avg, &rankType, &pageType,
		&season, &week, &source, &r.CreatedAt,
	)
	if err != nil {
		return FantasyRanking{}, err
	}

	if playerDbID.Valid {
		v := int(playerDbID.Int64)
		r.PlayerDbID = &v
	}
	if playerID.Valid {
		r.PlayerID = &playerID.String
	}
	if pos.Valid {
		r.Pos = &pos.String
	}
	if team.Valid {
		r.Team = &team.String
	}
	if rankType.Valid {
		r.RankType = &rankType.String
	}
	if pageType.Valid {
		r.PageType = &pageType.String
	}
	if source.Valid {
		r.Source = &source.String
	}
	if season.Valid {
		v := int(season.Int64)
		r.Season = &v
	}
	if week.Valid {
		v := int(week.Int64)
		r.Week = &v
	}
	if rank.Valid {
		v := int(rank.Int64)
		r.Rank = &v
	}
	if best.Valid {
		v := int(best.Int64)
		r.Best = &v
	}
	if worst.Valid {
		v := int(worst.Int64)
		r.Worst = &v
	}
	if ecr.Valid {
		r.ECR = &ecr.Float64
	}
	if sd.Valid {
		r.SD = &sd.Float64
	}
	if avg.Valid {
		r.Avg = &avg.Float64
	}

	return r, nil
}

var validRankingSortColumns = map[string]bool{
	"rank":        true,
	"player_name": true,
	"pos":         true,
	"team":        true,
	"ecr":         true,
	"sd":          true,
	"best":        true,
	"worst":       true,
	"avg":         true,
}

func buildRankingOrderBy(sort, order string) string {
	if sort != "" && validRankingSortColumns[sort] {
		dir := "ASC"
		if order == "desc" {
			dir = "DESC"
		}
		return " ORDER BY fr." + sort + " " + dir + " NULLS LAST"
	}
	return " ORDER BY fr.rank ASC NULLS LAST"
}
