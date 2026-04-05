package planner_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/stretchr/testify/suite"
)

// mockOllamaClient simulates Ollama inference for task estimation.
type mockOllamaClient struct {
	response string
	err      error
}

func (m *mockOllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	return m.response, m.err
}

type TaskEstimationSuite struct {
	suite.Suite
}

func TestTaskEstimation(t *testing.T) {
	suite.Run(t, new(TaskEstimationSuite))
}

// ---------------------------------------------------------------------------
// 1. Ollama estimate success — returns parsed pomodoro count
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateSuccess() {
	client := &mockOllamaClient{response: `{"pomodoros": 3}`, err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	pomos, err := estimator.EstimatePomodoros(context.Background(), "Write unit tests", "Cover all edge cases")
	s.Require().NoError(err)
	s.Equal(3, pomos)
}

// ---------------------------------------------------------------------------
// 2. Ollama failure fallback — returns 1 pomodoro on error
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateFallbackOnError() {
	client := &mockOllamaClient{response: "", err: fmt.Errorf("connection refused")}
	estimator := planner.NewOllamaTaskEstimator(client)

	pomos, err := estimator.EstimatePomodoros(context.Background(), "Deploy service", "Push to production")
	s.NoError(err, "should not propagate Ollama errors — fallback instead")
	s.Equal(1, pomos, "fallback should be 1 pomodoro")
}

// ---------------------------------------------------------------------------
// 3. Ollama returns invalid JSON — falls back to 1
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateFallbackOnInvalidJSON() {
	client := &mockOllamaClient{response: "not json at all", err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	pomos, err := estimator.EstimatePomodoros(context.Background(), "Review PR", "Check for bugs")
	s.NoError(err)
	s.Equal(1, pomos, "invalid JSON should fallback to 1 pomodoro")
}

// ---------------------------------------------------------------------------
// 4. Ollama returns zero or negative — falls back to 1
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateFallbackOnZero() {
	client := &mockOllamaClient{response: `{"pomodoros": 0}`, err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	pomos, err := estimator.EstimatePomodoros(context.Background(), "Quick fix", "One-liner")
	s.NoError(err)
	s.Equal(1, pomos, "zero estimate should fallback to 1 pomodoro")
}

// ---------------------------------------------------------------------------
// 5. EffectivePomos with user override
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEffectivePomosWithOverride() {
	override := 2
	te := planner.TaskEstimate{
		Title:          "Task",
		EstimatedPomos: 5,
		UserOverride:   &override,
	}
	s.Equal(2, te.EffectivePomos())
}

// ---------------------------------------------------------------------------
// 6. EffectivePomos without override
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEffectivePomosWithoutOverride() {
	te := planner.TaskEstimate{
		Title:          "Task",
		EstimatedPomos: 5,
	}
	s.Equal(5, te.EffectivePomos())
}
