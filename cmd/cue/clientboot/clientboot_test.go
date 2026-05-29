package clientboot_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/clientboot"
	"github.com/CreateFutureMWilkinson/cue/internal/config"
)

type ClientBootSuite struct {
	suite.Suite
}

func TestClientBoot(t *testing.T) {
	suite.Run(t, new(ClientBootSuite))
}

// hostPort splits an httptest.Server URL into a host string and an int
// port suitable for stuffing into config.ServerConfig.
func hostPort(s *suite.Suite, raw string) (string, int) {
	u, err := url.Parse(raw)
	s.Require().NoError(err)
	port, err := strconv.Atoi(u.Port())
	s.Require().NoError(err)
	return u.Hostname(), port
}

func (s *ClientBootSuite) TestConnectReturnsClientOnHealthyServer() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port := hostPort(&s.Suite, srv.URL)
	api, err := clientboot.Connect(
		context.Background(),
		config.ServerConfig{Host: host, Port: port},
		clientboot.Options{TotalTimeout: time.Second, PerAttempt: time.Second, Backoff: time.Millisecond},
	)
	s.Require().NoError(err)
	s.Require().NotNil(api)
	s.Empty(api.Token(), "Connect must not stamp a token; that's auth.Bootstrap's job")
}

func (s *ClientBootSuite) TestConnectRetriesUntilHealthy() {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port := hostPort(&s.Suite, srv.URL)
	api, err := clientboot.Connect(
		context.Background(),
		config.ServerConfig{Host: host, Port: port},
		clientboot.Options{TotalTimeout: 2 * time.Second, PerAttempt: 500 * time.Millisecond, Backoff: 5 * time.Millisecond},
	)
	s.Require().NoError(err)
	s.Require().NotNil(api)
	s.GreaterOrEqual(atomic.LoadInt32(&attempts), int32(3))
}

func (s *ClientBootSuite) TestConnectFailsWhenServerStaysDown() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	host, port := hostPort(&s.Suite, srv.URL)
	_, err := clientboot.Connect(
		context.Background(),
		config.ServerConfig{Host: host, Port: port},
		clientboot.Options{TotalTimeout: 50 * time.Millisecond, PerAttempt: 20 * time.Millisecond, Backoff: 5 * time.Millisecond},
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "clientboot: server not healthy")
}

func (s *ClientBootSuite) TestConnectRejectsZeroPort() {
	_, err := clientboot.Connect(
		context.Background(),
		config.ServerConfig{Host: "127.0.0.1", Port: 0},
		clientboot.Options{},
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "server.port must be set")
}

func (s *ClientBootSuite) TestConnectAbortsOnContextCancel() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	host, port := hostPort(&s.Suite, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := clientboot.Connect(
		ctx,
		config.ServerConfig{Host: host, Port: port},
		clientboot.Options{TotalTimeout: 5 * time.Second, PerAttempt: time.Second, Backoff: 5 * time.Millisecond},
	)
	s.Require().Error(err)
	s.True(
		strings.Contains(err.Error(), "aborted") || strings.Contains(err.Error(), "context"),
		"expected cancellation error, got %v", err,
	)
}

func (s *ClientBootSuite) TestConnectMapsZeroHostToLoopback() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, port := hostPort(&s.Suite, srv.URL)
	// Host = "0.0.0.0" should be rewritten to 127.0.0.1 for the dial.
	api, err := clientboot.Connect(
		context.Background(),
		config.ServerConfig{Host: "0.0.0.0", Port: port},
		clientboot.Options{TotalTimeout: time.Second, PerAttempt: time.Second, Backoff: time.Millisecond},
	)
	s.Require().NoError(err)
	s.NotNil(api)
}
