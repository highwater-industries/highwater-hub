package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client talks to the Python NFL stats service over HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a client pointing at the Python service.
// baseURL is something like "http://python-service:3142".
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// StartImport dispatches an import job to the Python service.
// Returns the accepted response (containing the job ID) or an error.
func (c *Client) StartImport(ctx context.Context, req ImportRequest) (ImportAccepted, error) {
	// 1. Marshal the request struct into JSON bytes
	body, err := json.Marshal(req)
	if err != nil {
		return ImportAccepted{}, fmt.Errorf("marshal request: %w", err)
	}

	// 2. Build an HTTP request with context
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v1/nflstats/import",
		bytes.NewReader(body),
	)
	if err != nil {
		return ImportAccepted{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 3. Execute the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ImportAccepted{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 4. Check for non-success status
	if resp.StatusCode != http.StatusAccepted {
		return ImportAccepted{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// 5. Decode the JSON response into our struct
	var result ImportAccepted
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ImportAccepted{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

// GetJobStatus polls the Python service for a job's current status.
func (c *Client) GetJobStatus(ctx context.Context, jobID string) (JobStatus, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v1/nflstats/jobs/"+jobID,
		nil, // GET requests have no body
	)
	if err != nil {
		return JobStatus{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return JobStatus{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return JobStatus{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result JobStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return JobStatus{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

// RevokeTask asks the Python service to revoke (terminate) a Celery task.
// This is best-effort: we don't fail the overall abort if revoke fails.
func (c *Client) RevokeTask(ctx context.Context, celeryTaskID string) error {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v1/nflstats/jobs/"+celeryTaskID+"/revoke",
		nil,
	)
	if err != nil {
		return fmt.Errorf("build revoke request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do revoke request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke unexpected status: %d", resp.StatusCode)
	}
	return nil
}
