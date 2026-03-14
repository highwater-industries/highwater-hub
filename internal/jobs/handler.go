package jobs

import (
	"net/http"
	"strconv"

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
//	GET /api/jobs?collector_type=nflreadpy&status=completed&season=2024
func HandleListJobs(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := httputil.ParsePagination(
			r.URL.Query().Get("offset"),
			r.URL.Query().Get("limit"),
		)

		var filter JobFilter
		filter.CollectorType = r.URL.Query().Get("collector_type")
		filter.Status = r.URL.Query().Get("status")
		if v := r.URL.Query().Get("season"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				filter.Season = n
			}
		}

		records, total, err := store.List(r.Context(), p.Offset, p.Limit, filter)
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

// HandleCleanupStuck marks stale running/pending jobs as failed.
//
//	POST /api/jobs/cleanup
func HandleCleanupStuck(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := store.CleanupStuck(r.Context())
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to cleanup stuck jobs",
			})
			return
		}

		httputil.Encode(w, http.StatusOK, map[string]any{
			"cleaned": count,
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

// HandleGetInventory returns an overview of all data in the database.
//
//	GET /api/data/inventory?source=nflreadpy&season=2024&stat_type=actual&season_type=REG&rank_type=draft
func HandleGetInventory(store InventoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var filter InventoryFilter
		filter.Source = r.URL.Query().Get("source")
		filter.StatType = r.URL.Query().Get("stat_type")
		filter.SeasonType = r.URL.Query().Get("season_type")
		filter.RankType = r.URL.Query().Get("rank_type")
		if v := r.URL.Query().Get("season"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				filter.Season = n
			}
		}

		inv, err := store.GetInventory(r.Context(), filter)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to get inventory: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, inv)
	}
}

// HandleAbortJob marks a single job as failed and revokes its Celery task.
//
//	POST /api/jobs/{id}/abort
func HandleAbortJob(store Store, client *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			httputil.Encode(w, http.StatusBadRequest, httputil.ErrorResponse{
				Error: "invalid job id",
			})
			return
		}

		celeryID, err := store.AbortJob(r.Context(), id)
		if err != nil {
			httputil.Encode(w, http.StatusNotFound, httputil.ErrorResponse{
				Error: err.Error(),
			})
			return
		}

		// Best-effort revoke of the Celery task
		revokeErr := ""
		if celeryID != "" {
			if err := client.RevokeTask(r.Context(), celeryID); err != nil {
				revokeErr = err.Error()
			}
		}

		httputil.Encode(w, http.StatusOK, map[string]any{
			"aborted":      id,
			"celery_task":  celeryID,
			"revoke_error": revokeErr,
		})
	}
}

// HandleAbortAll marks all active jobs as failed and revokes Celery tasks.
//
//	POST /api/jobs/abort-all
func HandleAbortAll(store Store, client *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, celeryIDs, err := store.AbortAllActive(r.Context())
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to abort jobs: " + err.Error(),
			})
			return
		}

		// Best-effort revoke each Celery task
		revoked := 0
		for _, cid := range celeryIDs {
			if err := client.RevokeTask(r.Context(), cid); err == nil {
				revoked++
			}
		}

		httputil.Encode(w, http.StatusOK, map[string]any{
			"aborted":        count,
			"celery_revoked": revoked,
		})
	}
}

// HandleRunAudit runs data quality checks on the specified table/season.
//
//	GET /api/data/audit?table=player_stats&season=2024
func HandleRunAudit(store InventoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		table := r.URL.Query().Get("table")
		if table == "" {
			table = "player_stats"
		}

		season := 0
		if v := r.URL.Query().Get("season"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				season = n
			}
		}

		result, err := store.RunAudit(r.Context(), table, season)
		if err != nil {
			httputil.Encode(w, http.StatusInternalServerError, httputil.ErrorResponse{
				Error: "failed to run audit: " + err.Error(),
			})
			return
		}

		httputil.Encode(w, http.StatusOK, result)
	}
}
