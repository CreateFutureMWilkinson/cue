package main

import (
	"context"
	"errors"
	"net/http"
)

// CosineSimilarity computes cosine similarity between two vectors.
// Returns 0 for zero-magnitude vectors.
func CosineSimilarity(a, b []float32) float64 {
	return 0
}

// embedText calls Ollama /api/embed for a single text input.
// Returns the embedding vector and request latency in milliseconds.
func embedText(ctx context.Context, model, text, host string, httpClient *http.Client) ([]float32, int64, error) {
	return nil, 0, errors.New("not implemented")
}
