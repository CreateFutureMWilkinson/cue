package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"
)

// BenchConfig holds the parsed CLI configuration for a benchmark run.
type BenchConfig struct {
	Baseline   string
	Models     []string
	OllamaHost string
	Timeout    time.Duration
	CorpusPath string
	Format     string
	Runs       int
	DryRun     bool
	Cooldown   time.Duration
	NoFewShot  bool
	Seed       int64
}

// NewApp creates a configured *cli.Command for the cue-bench CLI. The onRun
// callback, if non-nil, is invoked with the parsed BenchConfig instead of
// running the actual benchmark. This allows tests to inspect parsed values.
func NewApp(onRun func(cfg BenchConfig)) *cli.Command {
	return &cli.Command{
		Name:  "cue-bench",
		Usage: "Benchmark Ollama models for cue routing accuracy",
		Action: func(_ context.Context, _ *cli.Command) error {
			return ErrNotImplemented
		},
	}
}

func main() {
	app := NewApp(nil)
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
