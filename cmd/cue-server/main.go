// Command cue-server runs the headless HTTP/WebSocket entry point for
// Cue. It is retained as a thin compatibility wrapper around
// internal/server/runner so existing scripts keep working; the canonical
// entry point is `cue server` on the unified cmd/cue binary.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/server/runner"
)

const configRelPath = ".cue/config.toml"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--reset-auth" {
		if err := resetAuth(); err != nil {
			slog.Error("reset-auth failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("All auth tokens deleted. The next client to connect will be auto-issued a token.")
		os.Exit(0)
	}

	if err := run(); err != nil {
		slog.Error("cue-server: fatal", "error", err)
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	cfg, err := config.Load(filepath.Join(home, configRelPath))
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

func resetAuth() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	return server.ResetAuth(cfg.Database.Path)
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := cfg.ValidateForServer(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}
	return runner.Run(context.Background(), *cfg)
}
