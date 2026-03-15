package fantasy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client talks to the Python fantasy-import endpoint.
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

// StartImport dispatches a fantasy league import to the Python service.
func (c *Client) StartImport(ctx context.Context, req ImportRequest) (ImportAccepted, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return ImportAccepted{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v1/fantasy/import",
		bytes.NewReader(body),
	)
	if err != nil {
		return ImportAccepted{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ImportAccepted{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		var errBody struct {
			Detail string `json:"detail"`
		}
		json.NewDecoder(resp.Body).Decode(&errBody)
		return ImportAccepted{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, errBody.Detail)
	}

	var result ImportAccepted
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ImportAccepted{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}
