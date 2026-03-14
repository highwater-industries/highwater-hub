package jobs

import "context"

// Store reads job history from the database.
type Store interface {
	List(ctx context.Context, offset, limit int) ([]JobRecord, int, error)
	Summary(ctx context.Context) (JobSummary, error)
	CleanupStuck(ctx context.Context) (int, error)
}
