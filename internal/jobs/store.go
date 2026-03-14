package jobs

import "context"

// Store reads job history from the database.
type Store interface {
	List(ctx context.Context, offset, limit int, filter JobFilter) ([]JobRecord, int, error)
	Summary(ctx context.Context) (JobSummary, error)
	CleanupStuck(ctx context.Context) (int, error)
	AbortJob(ctx context.Context, id int) (celeryTaskID string, err error)
	AbortAllActive(ctx context.Context) (aborted int, celeryTaskIDs []string, err error)
}
