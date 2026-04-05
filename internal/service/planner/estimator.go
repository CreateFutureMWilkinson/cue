package planner

import (
	"context"
)

// OllamaGenerator abstracts the Ollama client for task estimation.
type OllamaGenerator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// OllamaTaskEstimator implements TaskEstimator using Ollama inference.
type OllamaTaskEstimator struct {
	client OllamaGenerator
}

// NewOllamaTaskEstimator creates a new OllamaTaskEstimator.
func NewOllamaTaskEstimator(client OllamaGenerator) *OllamaTaskEstimator {
	return nil
}

// EstimatePomodoros asks Ollama to estimate the number of pomodoros for a task.
func (e *OllamaTaskEstimator) EstimatePomodoros(ctx context.Context, title string, description string) (int, error) {
	return 0, nil
}
