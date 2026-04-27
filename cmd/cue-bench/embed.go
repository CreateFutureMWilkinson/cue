package main

import (
	"context"
	"errors"
	"net/http"
)

// embedText calls Ollama /api/embed for a single text input.
// Returns the embedding vector and request latency in milliseconds.
func embedText(ctx context.Context, model, text, host string, httpClient *http.Client) ([]float32, int64, error) {
	return nil, 0, errors.New("not implemented")
}
