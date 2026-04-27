package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
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

// EmbedResult pairs a corpus entry with its embedding vector.
type EmbedResult struct {
	Entry     CorpusEntry
	Embedding []float32
}

// EmbedIndex holds pre-computed embeddings for vector-based selection.
type EmbedIndex struct {
	Pool   []EmbedResult
	Scored map[string][]float32 // keyed by entry ID
}

// SelectExamplesByEmbedding selects up to n pool entries most similar
// to the scored entry's embedding.
func SelectExamplesByEmbedding(entryID string, index EmbedIndex, n int) []CorpusEntry {
	scoredVec, ok := index.Scored[entryID]
	if !ok {
		return nil
	}
	if n <= 0 || len(index.Pool) == 0 {
		return nil
	}

	type ranked struct {
		entry      CorpusEntry
		similarity float64
	}

	items := make([]ranked, len(index.Pool))
	for i, er := range index.Pool {
		items[i] = ranked{
			entry:      er.Entry,
			similarity: CosineSimilarity(scoredVec, er.Embedding),
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].similarity > items[j].similarity
	})

	if n > len(items) {
		n = len(items)
	}

	result := make([]CorpusEntry, n)
	for i := 0; i < n; i++ {
		result[i] = items[i].entry
	}
	return result
}

// BuildEmbedIndex embeds all pool and scored entries, returning the
// index and per-request latencies for metrics reporting.
func BuildEmbedIndex(ctx context.Context, model, host string, pool, scored []CorpusEntry, httpClient *http.Client, progressWriter io.Writer) (EmbedIndex, []int64, error) {
	index := EmbedIndex{
		Pool:   make([]EmbedResult, 0, len(pool)),
		Scored: make(map[string][]float32, len(scored)),
	}

	total := len(pool) + len(scored)
	latencies := make([]int64, 0, total)
	current := 0

	for _, entry := range pool {
		vec, latency, err := embedText(ctx, model, entry.Content, host, httpClient)
		if err != nil {
			return EmbedIndex{}, nil, err
		}
		index.Pool = append(index.Pool, EmbedResult{Entry: entry, Embedding: vec})
		latencies = append(latencies, latency)
		current++
		fmt.Fprintf(progressWriter, "Embedding %d/%d...\n", current, total)
	}

	for _, entry := range scored {
		vec, latency, err := embedText(ctx, model, entry.Content, host, httpClient)
		if err != nil {
			return EmbedIndex{}, nil, err
		}
		index.Scored[entry.ID] = vec
		latencies = append(latencies, latency)
		current++
		fmt.Fprintf(progressWriter, "Embedding %d/%d...\n", current, total)
	}

	return index, latencies, nil
}
