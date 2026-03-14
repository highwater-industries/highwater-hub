package jobs

import (
	"net/http"

	"myproject/internal/httputil"
)

// HandleStartImport dispatches an import job to the Python service.
//
//	POST /api/jobs/import
func HandleStartImport(client *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httputil.Decode[ImportRequest](r)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid JSON",
			})
			return
		}

		accepted, err := client.StartImport(r.Context(), req)
		if err != nil {
			httputil.Encode(w, http.StatusBadGateway, httputil.ErrorResponse{
				Error: "failed to start import: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusAccepted, accepted)
	}
}

// HandleBatchImport dispatches multiple import jobs in one request.
//
//	POST /api/jobs/import/batch
func HandleBatchImport(client *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httputil.Decode[BatchImportRequest](r)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid JSON",
			})
			return
		}

		if len(req.Imports) == 0 {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "imports array must not be empty",
			})
			return
		}

		results := make([]BatchImportResult, 0)
		dispatched, failed := 0, 0

		for _, imp := range req.Imports {
			accepted, err := client.StartImport(r.Context(), imp)
			if err != nil {
				results = append(results, BatchImportResult{
					CollectorType: imp.CollectorType,
					Status:        "failed",
					Error:         err.Error(),
				})
				failed++
			} else {
				results = append(results, BatchImportResult{
					CollectorType: accepted.CollectorType,
					JobID:         accepted.JobID,
					Status:        "accepted",
				})
				dispatched++
			}
		}

		httputil.Encode(w, http.StatusAccepted, BatchImportResponse{
			Results:    results,
			Dispatched: dispatched,
			Failed:     failed,
		})
	}
}

// HandleGetJobStatus polls the Python service for a job's status.
//
//	GET /api/jobs/{job_id}
func HandleGetJobStatus(client *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("job_id")
		if jobID == "" {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "missing job_id",
			})
			return
		}

		status, err := client.GetJobStatus(r.Context(), jobID)
		if err != nil {
			httputil.Encode(w, http.StatusBadGateway, httputil.ErrorResponse{
				Error: "failed to get job status: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, status)
	}
}

// HandleListJobs returns historical import runs from the database.
//
//	GET /api/jobs
func HandleListJobs(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		records, total, err := store.List(r.Context(), p.Offset, p.Limit)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to list jobs",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, httputil.ListResponse[JobRecord]{
			Items:  records,
			Total:  total,
			Offset: p.Offset,
			Limit:  p.Limit,
		})
	}
}

// HandleGetJobSummary returns aggregate status counts from collection_history.
//
//	GET /api/jobs/summary
func HandleGetJobSummary(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := store.Summary(r.Context())
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get job summary",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, summary)
	}
}
