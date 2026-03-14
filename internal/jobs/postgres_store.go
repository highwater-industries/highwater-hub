package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// PostgresStore implements Store by querying the collection_history table.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new store backed by the given database connection.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// List returns paginated job records, newest first, with optional filters.
func (s *PostgresStore) List(ctx context.Context, offset, limit int, filter JobFilter) ([]JobRecord, int, error) {
	// Build WHERE clause from filters
	where := "WHERE 1=1"
	args := []any{}
	if filter.CollectorType != "" {
		args = append(args, filter.CollectorType)
		where += fmt.Sprintf(" AND collector_type = $%d", len(args))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if filter.Season > 0 {
		args = append(args, fmt.Sprintf("[%d]", filter.Season))
		where += fmt.Sprintf(" AND params::jsonb->'seasons' @> $%d::jsonb", len(args))
	}

	// 1. Get total count
	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM collection_history "+where,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// 2. Fetch the page of rows
	pageArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, collector_type, status,
                records_fetched, records_inserted, records_updated, records_skipped,
                error_message, started_at, finished_at, params, progress
         FROM collection_history %s
         ORDER BY started_at DESC
         LIMIT $%d OFFSET $%d`, where, len(args)+1, len(args)+2),
		pageArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	// 3. Scan each row into a JobRecord
	records := make([]JobRecord, 0)
	for rows.Next() {
		var rec JobRecord
		var paramsJSON sql.NullString
		var errMsg sql.NullString
		var finishedAt sql.NullString
		var progress sql.NullFloat64

		err := rows.Scan(
			&rec.ID,
			&rec.CollectorType,
			&rec.Status,
			&rec.RecordsFetched,
			&rec.RecordsInserted,
			&rec.RecordsUpdated,
			&rec.RecordsSkipped,
			&errMsg,
			&rec.StartedAt,
			&finishedAt,
			&paramsJSON,
			&progress,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan job row: %w", err)
		}

		// Convert nullable columns to pointer fields
		if errMsg.Valid {
			rec.ErrorMessage = &errMsg.String
		}
		if finishedAt.Valid {
			rec.FinishedAt = &finishedAt.String
		}
		if paramsJSON.Valid {
			json.Unmarshal([]byte(paramsJSON.String), &rec.Params)
		}
		if progress.Valid {
			rec.Progress = &progress.Float64
		}

		records = append(records, rec)
	}

	// 4. Check for errors that occurred during iteration
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate jobs: %w", err)
	}

	return records, total, nil
}

// Summary returns aggregate status counts across all collection_history rows.
func (s *PostgresStore) Summary(ctx context.Context) (JobSummary, error) {
	var summary JobSummary

	rows, err := s.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM collection_history GROUP BY status`,
	)
	if err != nil {
		return summary, fmt.Errorf("query job summary: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return summary, fmt.Errorf("scan summary row: %w", err)
		}
		switch status {
		case "pending":
			summary.Pending = count
		case "running":
			summary.Running = count
		case "completed":
			summary.Completed = count
		case "failed":
			summary.Failed = count
		}
		summary.Total += count
	}

	if err := rows.Err(); err != nil {
		return summary, fmt.Errorf("iterate summary: %w", err)
	}

	return summary, nil
}

// CleanupStuck marks any jobs stuck in "running" or "pending" for over 1 hour
// as "failed" and returns the count of affected rows.
func (s *PostgresStore) CleanupStuck(ctx context.Context) (int, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE collection_history
		 SET status = 'failed',
		     error_message = 'Marked as failed: job was stuck',
		     finished_at = NOW()
		 WHERE status IN ('running', 'pending', 'started', 'STARTED')
		   AND started_at < NOW() - INTERVAL '1 hour'`,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup stuck jobs: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// AbortJob marks a single job as failed by ID and returns its Celery task ID.
func (s *PostgresStore) AbortJob(ctx context.Context, id int) (string, error) {
	var celeryID sql.NullString
	err := s.db.QueryRowContext(ctx,
		`UPDATE collection_history
		 SET status = 'failed',
		     error_message = 'Manually aborted',
		     finished_at = NOW()
		 WHERE id = $1 AND status IN ('running', 'pending', 'started', 'STARTED', 'PENDING')
		 RETURNING params::jsonb->>'celery_task_id'`,
		id,
	).Scan(&celeryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("job %d not found or not active", id)
		}
		return "", fmt.Errorf("abort job: %w", err)
	}
	return celeryID.String, nil
}

// AbortAllActive marks all running/pending jobs as failed and returns their Celery task IDs.
func (s *PostgresStore) AbortAllActive(ctx context.Context) (int, []string, error) {
	rows, err := s.db.QueryContext(ctx,
		`UPDATE collection_history
		 SET status = 'failed',
		     error_message = 'Manually aborted (abort all)',
		     finished_at = NOW()
		 WHERE status IN ('running', 'pending', 'started', 'STARTED', 'PENDING')
		 RETURNING params::jsonb->>'celery_task_id'`,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("abort all: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var celeryID sql.NullString
		if err := rows.Scan(&celeryID); err != nil {
			return 0, nil, fmt.Errorf("scan abort result: %w", err)
		}
		if celeryID.Valid && celeryID.String != "" {
			ids = append(ids, celeryID.String)
		}
	}
	return len(ids), ids, rows.Err()
}
