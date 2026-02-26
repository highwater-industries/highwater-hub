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
