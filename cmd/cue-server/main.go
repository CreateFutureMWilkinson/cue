// Command cue-server runs the headless HTTP/WebSocket entry point for
// Cue. It loads configuration from ~/.cue/config.toml, validates that
// the [server] section is present, and boots a full server.Composition
// (repositories, services, orchestrator, watchers, HTTP/WebSocket
// surface). SIGINT or SIGTERM triggers an ordered shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

const configRelPath = ".cue/config.toml"

func main() {
	if err := run(); err != nil {
		slog.Error("cue-server: fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}
	cfgPath := filepath.Join(home, configRelPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.ValidateForServer(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comp, err := server.NewComposition(ctx, *cfg)
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
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Attempt an orderly shutdown even on listener failure so repos/orchestrator close cleanly.
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
