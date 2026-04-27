package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrNotImplemented is returned by stubs not yet implemented.
var ErrNotImplemented = errors.New("not implemented")

// APIClient is the base client for the Cue server API.
// Adapters share this for HTTP transport, auth header injection, and error parsing.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// New constructs a new APIClient targeting the given server base URL.
func New(baseURL string) *APIClient {
	return &APIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// SetToken configures the bearer token sent on subsequent requests.
func (c *APIClient) SetToken(token string) {
	c.token = token
}

// Token returns the bearer token currently stored on the client. Exposed so
// callers can persist a token that was auto-issued by the server on the
// first-client TOKEN_ISSUED flow (Feature 106 Loop 4).
func (c *APIClient) Token() string {
	return c.token
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

// doJSON performs an HTTP request with an optional JSON body and decodes the
// JSON response into out (if out is non-nil). Returns *APIError on non-2xx.
// If reqBody is non-nil, it is marshaled as JSON and sent with
// Content-Type: application/json. The Authorization: Bearer header is set if
// a token is stored on the client.
func (c *APIClient) doJSON(ctx context.Context, method, path string, reqBody, out any) error {
	var body io.Reader
	if reqBody != nil {
		buf, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return ParseErrorResponse(resp.StatusCode, respBody)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
