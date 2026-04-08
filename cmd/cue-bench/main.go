package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "baseline", Value: "neural-chat", Usage: "Baseline model to compare against"},
			&cli.StringFlag{Name: "models", Required: true, Usage: "Comma-separated list of models to benchmark"},
			&cli.StringFlag{Name: "ollama-host", Value: "http://localhost:11434", Usage: "Ollama API host URL"},
			&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "Per-request timeout for Ollama calls"},
			&cli.StringFlag{Name: "corpus", Value: "", Usage: "Path to test corpus file"},
			&cli.StringFlag{Name: "format", Value: "table", Usage: "Output format (table, csv, json)"},
			&cli.IntFlag{Name: "runs", Value: 1, Usage: "Number of benchmark runs per model"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Parse flags and exit without running benchmarks"},
			&cli.DurationFlag{Name: "cooldown", Value: 2 * time.Second, Usage: "Cooldown between benchmark runs"},
			&cli.BoolFlag{Name: "no-fewshot", Usage: "Disable few-shot prompt injection"},
			&cli.IntFlag{Name: "seed", Value: 42, Usage: "Random seed for reproducibility"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg := BenchConfig{
				Baseline:   cmd.String("baseline"),
				Models:     strings.Split(cmd.String("models"), ","),
				OllamaHost: cmd.String("ollama-host"),
				Timeout:    cmd.Duration("timeout"),
				CorpusPath: cmd.String("corpus"),
				Format:     cmd.String("format"),
				Runs:       int(cmd.Int("runs")),
				DryRun:     cmd.Bool("dry-run"),
				Cooldown:   cmd.Duration("cooldown"),
				NoFewShot:  cmd.Bool("no-fewshot"),
				Seed:       int64(cmd.Int("seed")),
			}

			if onRun != nil {
				onRun(cfg)
				return nil
			}

			if cfg.DryRun {
				return nil
			}

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
