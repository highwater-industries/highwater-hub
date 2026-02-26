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

// List returns paginated job records, newest first.
func (s *PostgresStore) List(ctx context.Context, offset, limit int) ([]JobRecord, int, error) {
	// 1. Get total count
	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM collection_history",
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// 2. Fetch the page of rows
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, collector_type, status,
                records_fetched, records_inserted, records_updated, records_skipped,
                error_message, started_at, finished_at, params
         FROM collection_history
         ORDER BY started_at DESC
         LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	// 3. Scan each row into a JobRecord
	var records []JobRecord
	for rows.Next() {
		var rec JobRecord
		var paramsJSON sql.NullString
		var errMsg sql.NullString
		var finishedAt sql.NullString

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

		records = append(records, rec)
	}

	// 4. Check for errors that occurred during iteration
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate jobs: %w", err)
	}

	return records, total, nil
}
