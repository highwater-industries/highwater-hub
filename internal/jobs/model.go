package jobs

// ImportRequest is the body sent to start a new import job.
type ImportRequest struct {
	CollectorType string `json:"collector_type"`
	Seasons       []int  `json:"seasons"`
	Strategy      string `json:"strategy"`
	SummaryLevel  string `json:"summary_level,omitempty"`
	RankType      string `json:"rank_type,omitempty"`
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

// BatchImportRequest dispatches multiple imports at once.
type BatchImportRequest struct {
	Imports []ImportRequest `json:"imports"`
}

// BatchImportResult is one entry in the batch response.
type BatchImportResult struct {
	CollectorType string `json:"collector_type"`
	JobID         string `json:"job_id,omitempty"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

// BatchImportResponse is returned from POST /api/jobs/import/batch.
type BatchImportResponse struct {
	Results    []BatchImportResult `json:"results"`
	Dispatched int                 `json:"dispatched"`
	Failed     int                 `json:"failed"`
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
	Progress        *float64       `json:"progress"`
}

// JobFilter holds optional filter criteria for listing jobs.
type JobFilter struct {
	CollectorType string
	Status        string
	Season        int // 0 = all
}

// InventoryFilter holds optional filter criteria for the inventory endpoint.
type InventoryFilter struct {
	Source     string
	Season     int // 0 = all
	StatType   string
	SeasonType string
	RankType   string
}

// JobSummary holds aggregate status counts.
type JobSummary struct {
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}
