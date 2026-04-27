package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ClientSuite covers the APIClient base type and Health() method.
type ClientSuite struct {
	suite.Suite
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}

// TestNewAPIClient verifies that New() returns a non-nil APIClient.
func (s *ClientSuite) TestNewAPIClient() {
	c := client.New("http://example.com")
	s.NotNil(c, "New should return a non-nil *APIClient")
}

// TestHealthReturnsNilOn200 verifies Health() returns nil when the server
// responds to GET /health with a 200 status code.
func (s *ClientSuite) TestHealthReturnsNilOn200() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method, "Health should issue a GET request")
		s.Equal("/health", r.URL.Path, "Health should target /health")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	c := client.New(server.URL)
	err := c.Health(context.Background())
	s.NoError(err, "Health should return nil on 200 response")
}

// TestHealthReturnsErrorOn503 verifies Health() returns a non-nil error when
// the server responds to GET /health with a 503 status code.
func (s *ClientSuite) TestHealthReturnsErrorOn503() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method, "Health should issue a GET request")
		s.Equal("/health", r.URL.Path, "Health should target /health")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := client.New(server.URL)
	err := c.Health(context.Background())
	s.Error(err, "Health should return a non-nil error on 503 response")
}
