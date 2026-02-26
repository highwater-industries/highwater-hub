package jobs

// ImportRequest is the body sent to start a new import job.
type ImportRequest struct {
	CollectorType string `json:"collector_type"`
	Seasons       []int  `json:"seasons"`
	Strategy      string `json:"strategy"`
}

// ImportAccepted is the response from the Python service when a job is dispatched.
type ImportAccepted struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	CollectorType string `json:"collector_type"`
	Seasons       []int  `json:"seasons"`
	Strategy      string `json:"strategy"`
}

// JobStatus is the response when polling a running or completed job.
type JobStatus struct {
	JobID    string         `json:"job_id"`
	Status   string         `json:"status"`
	Progress *float64       `json:"progress,omitempty"`
	Meta     map[string]any `json:"meta,omitempty"`
	Result   map[string]any `json:"result,omitempty"`
	Error    *string        `json:"error,omitempty"`
}

// JobRecord is a historical import run from the collection_history table.
type JobRecord struct {
	ID              int            `json:"id"`
	CollectorType   string         `json:"collector_type"`
	Status          string         `json:"status"`
	RecordsFetched  int            `json:"records_fetched"`
	RecordsInserted int            `json:"records_inserted"`
	RecordsUpdated  int            `json:"records_updated"`
	RecordsSkipped  int            `json:"records_skipped"`
	ErrorMessage    *string        `json:"error_message,omitempty"`
	StartedAt       string         `json:"started_at"`
	FinishedAt      *string        `json:"finished_at,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
}
