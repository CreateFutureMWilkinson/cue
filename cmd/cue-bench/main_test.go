package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type CLISuite struct {
	suite.Suite
}

func TestCLI(t *testing.T) { suite.Run(t, new(CLISuite)) }

func (s *CLISuite) TestCLI_MissingModelsFlag_ReturnsError() {
	app := NewApp(nil)
	err := app.Run(context.Background(), []string{"cue-bench"})
	s.Error(err, "running with no flags should return an error because --models is required")
}

func (s *CLISuite) TestCLI_WithModelsFlag_NoError() {
	app := NewApp(nil)
	err := app.Run(context.Background(), []string{"cue-bench", "--models", "phi3:mini", "--dry-run"})
	s.NoError(err, "running with --models and --dry-run should succeed without error")
}

func (s *CLISuite) TestCLI_DefaultBaseline() {
	var captured BenchConfig
	called := false
	app := NewApp(func(cfg BenchConfig) {
		captured = cfg
		called = true
	})

	err := app.Run(context.Background(), []string{"cue-bench", "--models", "phi3:mini", "--dry-run"})
	s.Require().NoError(err, "app.Run should not error with valid flags")
	s.Require().True(called, "onRun callback must be invoked")
	s.Equal("neural-chat", captured.Baseline, "default baseline should be 'neural-chat'")
}

func (s *CLISuite) TestCLI_DefaultFormat() {
	var captured BenchConfig
	called := false
	app := NewApp(func(cfg BenchConfig) {
		captured = cfg
		called = true
	})

	err := app.Run(context.Background(), []string{"cue-bench", "--models", "phi3:mini", "--dry-run"})
	s.Require().NoError(err, "app.Run should not error with valid flags")
	s.Require().True(called, "onRun callback must be invoked")
	s.Equal("table", captured.Format, "default format should be 'table'")
}

// ensure BenchConfig fields match expected types at compile time.
var _ = BenchConfig{
	Baseline:   "",
	Models:     nil,
	OllamaHost: "",
	Timeout:    time.Duration(0),
	CorpusPath: "",
	Format:     "",
	Runs:       0,
	DryRun:     false,
	Cooldown:   time.Duration(0),
	NoFewShot:  false,
	Seed:       0,
}
