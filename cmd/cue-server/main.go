// Command cue-server runs the headless HTTP/WebSocket entry point for
// Cue. It loads the same configuration as the GUI binary but replaces
// the Fyne UI with an HTTP server. The current binary wires the HTTP
// surface (health, middleware, WebSocket hub); subsequent features
// will attach repositories, services, and watchers.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	sqlite "github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

const (
	configRelPath   = ".cue/config.toml"
	shutdownTimeout = 15 * time.Second
)

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
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	msgRepo, err := sqlite.NewSQLiteMessageRepository(cfg.Database.Path, cfg.Orchestrator.Router.BufferSizePerSource)
	if err != nil {
		return fmt.Errorf("opening message database: %w", err)
	}

	srv, err := server.New(cfg.Server, server.Deps{Messages: msgRepo})
	if err != nil {
		return fmt.Errorf("constructing server: %w", err)
	}

	hubCtx, cancelHub := context.WithCancel(context.Background())
	defer cancelHub()
	go func() {
		if err := srv.Hub().Run(hubCtx); err != nil {
			slog.Error("hub stopped with error", "error", err)
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("cue-server listening", "host", cfg.Server.Host, "port", cfg.Server.Port)
		errCh <- srv.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("shutdown: %w", err)
	}
	cancelHub()
	return nil
}
