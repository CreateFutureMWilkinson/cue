package client

import (
	"context"
	"errors"
	"net/http"
)

// ErrNotImplemented is returned by stubs not yet implemented.
var ErrNotImplemented = errors.New("not implemented")

// APIClient is the base client for the Cue server API.
// Adapters share this for HTTP transport, auth header injection, and error parsing.
type APIClient struct {
	// fields added in GREEN phase
}

// New constructs a new APIClient targeting the given server base URL.
func New(baseURL string) *APIClient {
	return &APIClient{}
}

// Health calls GET /health on the server and returns nil on a 2xx response.
func (c *APIClient) Health(ctx context.Context) error {
	return ErrNotImplemented
}

// Ensure http is referenced so the import is not unused when tests compile.
var _ = http.StatusOK
