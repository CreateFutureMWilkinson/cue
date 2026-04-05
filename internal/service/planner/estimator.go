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
	Pomodoros int `json:"pomodoros"`
}

// EstimatePomodoros asks Ollama to estimate the number of pomodoros for a task.
// On any failure (network, invalid JSON, zero/negative), falls back to 1 pomodoro.
func (e *OllamaTaskEstimator) EstimatePomodoros(ctx context.Context, title string, description string) (int, error) {
	prompt := fmt.Sprintf(
		`Estimate how many 25-minute Pomodoro sessions this task will take. Reply with JSON only: {"pomodoros": N}

Task: %s
Description: %s`, title, description)

	resp, err := e.client.Generate(ctx, prompt)
	if err != nil {
		return 1, nil
	}

	var result estimateResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return 1, nil
	}

	if result.Pomodoros <= 0 {
		return 1, nil
	}

	return result.Pomodoros, nil
}
