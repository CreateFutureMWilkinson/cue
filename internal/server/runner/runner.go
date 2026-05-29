// Package runner extracts the cue-server boot sequence so it can be
// invoked from both cmd/cue-server (legacy entry point) and the
// `cue server` subcommand on the unified cmd/cue binary.
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

// Run boots the full server composition (repositories, services,
// orchestrator, watchers, HTTP/WebSocket surface), blocks until SIGINT,
// SIGTERM, or a listener error, then performs an ordered shutdown.
//
// The supplied config must already have been validated with
// config.ValidateForServer.
func Run(ctx context.Context, cfg config.Config) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	comp, err := server.NewComposition(runCtx, cfg)
	if err != nil {
		return fmt.Errorf("building composition: %w", err)
	}

	hubCtx, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go func() {
		if err := comp.Hub.Run(hubCtx); err != nil {
			slog.Error("hub stopped with error", "error", err)
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("cue-server listening", "host", cfg.Server.Host, "port", cfg.Server.Port)
		errCh <- comp.HTTP.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig.String())
	case <-runCtx.Done():
		slog.Info("context cancelled, shutting down")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = comp.Shutdown(context.Background())
			return fmt.Errorf("server error: %w", err)
		}
	}

	if err := comp.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	cancelHub()
	return nil
}
