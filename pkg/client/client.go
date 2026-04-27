package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ErrNotImplemented is returned by stubs not yet implemented.
var ErrNotImplemented = errors.New("not implemented")

// APIClient is the base client for the Cue server API.
// Adapters share this for HTTP transport, auth header injection, and error parsing.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a new APIClient targeting the given server base URL.
func New(baseURL string) *APIClient {
	return &APIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Health calls GET /health on the server and returns nil on a 2xx response.
func (c *APIClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health returned status %d", resp.StatusCode)
	}
	return nil
}
