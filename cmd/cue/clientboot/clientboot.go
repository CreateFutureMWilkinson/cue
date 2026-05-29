// Package clientboot encapsulates the small amount of bootstrap logic
// that runs in the cue ui action between config-load and the first
// SDK-backed presenter call: building the APIClient and waiting for
// the server's /health endpoint to respond.
//
// Keeping this in its own package lets us test the retry/timeout
// behavior end-to-end against httptest without booting Fyne.
package clientboot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// Defaults chosen to match the doc: the user notices a five-second
// stall, but each individual probe is short enough that a refused
// connection retries quickly.
const (
	defaultTotalTimeout  = 5 * time.Second
	defaultPerAttemptDur = 1 * time.Second
	defaultBackoff       = 250 * time.Millisecond
)

// Options tunes Connect's retry budget. The zero value uses the
// defaults documented on the package constants.
type Options struct {
	// TotalTimeout is the wall-clock budget for the entire health
	// poll. When exceeded the latest probe error is returned.
	TotalTimeout time.Duration

	// PerAttempt is the per-probe timeout. Each attempt's context is
	// derived from the caller's ctx with this deadline.
	PerAttempt time.Duration

	// Backoff is the gap between successive probes after a failure.
	Backoff time.Duration

	// Now is overridable for deterministic tests; defaults to time.Now.
	Now func() time.Time

	// Sleep is overridable for deterministic tests; defaults to a
	// context-aware sleep.
	Sleep func(ctx context.Context, d time.Duration)
}

// Connect builds an *client.APIClient targeting the given server
// config and polls Health() until it succeeds or the total timeout
// elapses. Returns the ready-to-use client on success.
//
// The returned APIClient has no token set; callers wire the token
// via auth.Bootstrap.
func Connect(ctx context.Context, srv config.ServerConfig, opts Options) (*client.APIClient, error) {
	host := srv.Host
	if host == "" || host == "0.0.0.0" {
		// 0.0.0.0 is the listener bind address; clients dial loopback.
		host = "127.0.0.1"
	}
	if srv.Port == 0 {
		return nil, errors.New("clientboot: server.port must be set")
	}
	baseURL := "http://" + net.JoinHostPort(host, strconv.Itoa(srv.Port))

	api := client.New(baseURL)

	if opts.TotalTimeout <= 0 {
		opts.TotalTimeout = defaultTotalTimeout
	}
	if opts.PerAttempt <= 0 {
		opts.PerAttempt = defaultPerAttemptDur
	}
	if opts.Backoff <= 0 {
		opts.Backoff = defaultBackoff
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Sleep == nil {
		opts.Sleep = sleepCtx
	}

	deadline := opts.Now().Add(opts.TotalTimeout)
	var lastErr error
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, opts.PerAttempt)
		err := api.Health(attemptCtx)
		cancel()
		if err == nil {
			return api, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return nil, fmt.Errorf("clientboot: aborted before server became healthy: %w", ctx.Err())
		}
		if !opts.Now().Before(deadline) {
			return nil, fmt.Errorf("clientboot: server not healthy at %s after %s: %w", baseURL, opts.TotalTimeout, lastErr)
		}
		opts.Sleep(ctx, opts.Backoff)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
