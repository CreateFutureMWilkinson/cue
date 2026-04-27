package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// CosineSimilarity computes cosine similarity between two vectors.
// Returns 0 for zero-magnitude vectors or mismatched lengths.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, magA, magB float64
	for i := range a {
		aVal, bVal := float64(a[i]), float64(b[i])
		dot += aVal * bVal
		magA += aVal * aVal
		magB += bVal * bVal
	}

	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (magA * magB)
}

// embedText calls Ollama /api/embed for a single text input.
// Returns the embedding vector and request latency in milliseconds.
func embedText(ctx context.Context, model, text, host string, httpClient *http.Client) ([]float32, int64, error) {
	reqBody, err := json.Marshal(map[string]string{
		"model": model,
		"input": text,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/api/embed", bytes.NewReader(reqBody))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := httpClient.Do(req)
	elapsed := time.Since(start)
	latencyMs := max(elapsed.Milliseconds(), 1)
	if err != nil {
		return nil, 0, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("embed request returned status %d", resp.StatusCode)
	}

	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, 0, fmt.Errorf("empty embeddings in response")
	}

	return result.Embeddings[0], latencyMs, nil
}
