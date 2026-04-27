package planner

import (
	"context"
	"encoding/json"
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
	const fallback = 30

	prompt := fmt.Sprintf("Estimate how many minutes this task will take. Reply with JSON only: {\"minutes\": N}\n\nTask: %s\nDescription: %s", title, description)

	raw, err := e.client.Generate(ctx, prompt)
	if err != nil {
		return fallback, nil
	}

	var resp estimateResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return fallback, nil
	}

	if resp.Minutes <= 0 {
		return fallback, nil
	}

	return resp.Minutes, nil
}
