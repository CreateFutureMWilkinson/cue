package planner

import (
	"context"
	"fmt"
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
	return &OllamaTaskEstimator{client: client}
}

type estimateResponse struct {
	Minutes int `json:"minutes"`
}

// EstimateMinutes asks Ollama to estimate the number of minutes for a task.
// On any failure (network, invalid JSON, zero/negative), falls back to 30 minutes.
func (e *OllamaTaskEstimator) EstimateMinutes(ctx context.Context, title string, description string) (int, error) {
	_ = fmt.Sprintf("placeholder %s %s", title, description)
	return 0, fmt.Errorf("not implemented")
}
