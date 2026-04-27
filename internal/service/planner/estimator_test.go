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
// 1. EstimateMinutes success — returns parsed minute count
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateMinutesSuccess() {
	client := &mockOllamaClient{response: `{"minutes": 45}`, err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	minutes, err := estimator.EstimateMinutes(context.Background(), "Write unit tests", "Cover all edge cases")
	s.Require().NoError(err)
	s.Equal(45, minutes)
}

// ---------------------------------------------------------------------------
// 2. Ollama failure fallback — returns 30 minutes on error
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateMinutesFallbackOnError() {
	client := &mockOllamaClient{response: "", err: fmt.Errorf("connection refused")}
	estimator := planner.NewOllamaTaskEstimator(client)

	minutes, err := estimator.EstimateMinutes(context.Background(), "Deploy service", "Push to production")
	s.NoError(err, "should not propagate Ollama errors — fallback instead")
	s.Equal(30, minutes, "fallback should be 30 minutes")
}

// ---------------------------------------------------------------------------
// 3. Ollama returns invalid JSON — falls back to 30 minutes
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateMinutesFallbackOnInvalidJSON() {
	client := &mockOllamaClient{response: "not json at all", err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	minutes, err := estimator.EstimateMinutes(context.Background(), "Review PR", "Check for bugs")
	s.NoError(err)
	s.Equal(30, minutes, "invalid JSON should fallback to 30 minutes")
}

// ---------------------------------------------------------------------------
// 4. Ollama returns zero or negative — falls back to 30 minutes
// ---------------------------------------------------------------------------

func (s *TaskEstimationSuite) TestEstimateMinutesFallbackOnZero() {
	client := &mockOllamaClient{response: `{"minutes": 0}`, err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	minutes, err := estimator.EstimateMinutes(context.Background(), "Quick fix", "One-liner")
	s.NoError(err)
	s.Equal(30, minutes, "zero estimate should fallback to 30 minutes")
}

func (s *TaskEstimationSuite) TestEstimateMinutesFallbackOnNegative() {
	client := &mockOllamaClient{response: `{"minutes": -5}`, err: nil}
	estimator := planner.NewOllamaTaskEstimator(client)

	minutes, err := estimator.EstimateMinutes(context.Background(), "Quick fix", "One-liner")
	s.NoError(err)
	s.Equal(30, minutes, "negative estimate should fallback to 30 minutes")
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
