package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type EmbedSuite struct {
	suite.Suite
}

func TestEmbed(t *testing.T) { suite.Run(t, new(EmbedSuite)) }

// CosineSimilarity tests

func (s *EmbedSuite) TestCosineSimilarity_IdenticalVectors() {
	v := []float32{1.0, 2.0, 3.0}
	result := CosineSimilarity(v, v)
	s.InDelta(1.0, result, 1e-6)
}

func (s *EmbedSuite) TestCosineSimilarity_OrthogonalVectors() {
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{0.0, 1.0, 0.0}
	result := CosineSimilarity(a, b)
	s.InDelta(0.0, result, 1e-6)
}

func (s *EmbedSuite) TestCosineSimilarity_OppositeVectors() {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{-1.0, -2.0, -3.0}
	result := CosineSimilarity(a, b)
	s.InDelta(-1.0, result, 1e-6)
}

func (s *EmbedSuite) TestCosineSimilarity_ZeroMagnitudeVector() {
	zero := []float32{0.0, 0.0, 0.0}
	nonZero := []float32{1.0, 2.0, 3.0}

	s.Equal(0.0, CosineSimilarity(zero, nonZero), "zero first arg")
	s.Equal(0.0, CosineSimilarity(nonZero, zero), "zero second arg")
	s.Equal(0.0, CosineSimilarity(zero, zero), "both zero")
}

func (s *EmbedSuite) TestCosineSimilarity_MismatchedLengths() {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0}

	s.Equal(0.0, CosineSimilarity(a, b), "first vector longer")
	s.Equal(0.0, CosineSimilarity(b, a), "second vector longer")
}

// embedText tests

func (s *EmbedSuite) TestEmbedText_Success() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings": [[0.1, 0.2, 0.3]]}`)
	}))
	defer server.Close()

	vec, latency, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Require().NoError(err)
	s.Equal([]float32{0.1, 0.2, 0.3}, vec)
	s.Greater(latency, int64(0))
}

func (s *EmbedSuite) TestEmbedText_HTTPError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	vec, _, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Error(err)
	s.Nil(vec)
}

func (s *EmbedSuite) TestEmbedText_InvalidJSON() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `not json`)
	}))
	defer server.Close()

	vec, _, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Error(err)
	s.Nil(vec)
}

func (s *EmbedSuite) TestEmbedText_EmptyEmbeddings() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings": []}`)
	}))
	defer server.Close()

	vec, _, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Error(err)
	s.Nil(vec)
}

func (s *EmbedSuite) TestEmbedText_SendsCorrectRequest() {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings": [[0.1]]}`)
	}))
	defer server.Close()

	_, _, _ = embedText(context.Background(), "test-model", "test input", server.URL, server.Client())

	s.Require().NotNil(capturedBody, "request body was not captured")
	s.Equal("test-model", capturedBody["model"])
	s.Equal("test input", capturedBody["input"])
}

// SelectExamplesByEmbedding tests

func (s *EmbedSuite) TestSelectExamplesByEmbedding_RanksBySimilarity() {
	scored := []float32{1, 0, 0}
	index := EmbedIndex{
		Pool: []EmbedResult{
			{Entry: CorpusEntry{ID: "far"}, Embedding: []float32{0, 1, 0}},
			{Entry: CorpusEntry{ID: "close"}, Embedding: []float32{0.9, 0.1, 0}},
			{Entry: CorpusEntry{ID: "medium"}, Embedding: []float32{0.5, 0.5, 0}},
		},
		Scored: map[string][]float32{
			"entry1": scored,
		},
	}

	results := SelectExamplesByEmbedding("entry1", index, 3)

	s.Require().Len(results, 3)
	s.Equal("close", results[0].ID)
	s.Equal("medium", results[1].ID)
	s.Equal("far", results[2].ID)
}

func (s *EmbedSuite) TestSelectExamplesByEmbedding_CapsAtN() {
	scored := []float32{1, 0, 0}
	index := EmbedIndex{
		Pool: []EmbedResult{
			{Entry: CorpusEntry{ID: "a"}, Embedding: []float32{0.9, 0.1, 0}},
			{Entry: CorpusEntry{ID: "b"}, Embedding: []float32{0.5, 0.5, 0}},
			{Entry: CorpusEntry{ID: "c"}, Embedding: []float32{0, 1, 0}},
		},
		Scored: map[string][]float32{
			"entry1": scored,
		},
	}

	results := SelectExamplesByEmbedding("entry1", index, 2)

	s.Require().Len(results, 2)
	s.Equal("a", results[0].ID)
	s.Equal("b", results[1].ID)
}

func (s *EmbedSuite) TestSelectExamplesByEmbedding_MissingEntryID() {
	index := EmbedIndex{
		Pool: []EmbedResult{
			{Entry: CorpusEntry{ID: "a"}, Embedding: []float32{1, 0, 0}},
		},
		Scored: map[string][]float32{
			"other": {1, 0, 0},
		},
	}

	results := SelectExamplesByEmbedding("nonexistent", index, 3)

	s.Nil(results)
}

func (s *EmbedSuite) TestSelectExamplesByEmbedding_EmptyPool() {
	index := EmbedIndex{
		Pool:   []EmbedResult{},
		Scored: map[string][]float32{"entry1": {1, 0, 0}},
	}

	results := SelectExamplesByEmbedding("entry1", index, 3)

	s.Nil(results)
}

func (s *EmbedSuite) TestSelectExamplesByEmbedding_NZero() {
	index := EmbedIndex{
		Pool: []EmbedResult{
			{Entry: CorpusEntry{ID: "a"}, Embedding: []float32{1, 0, 0}},
		},
		Scored: map[string][]float32{"entry1": {1, 0, 0}},
	}

	results := SelectExamplesByEmbedding("entry1", index, 0)

	s.Nil(results)
}

// BuildEmbedIndex tests

func makeEntry(id, content string, rating *int) CorpusEntry {
	return CorpusEntry{ID: id, Content: content, UserRating: rating}
}

func embedMockServer(counter *atomic.Int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := float32(counter.Add(1))
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"embeddings": [][]float32{{n, 0, 0}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func (s *EmbedSuite) TestBuildEmbedIndex_IndexesPoolAndScored() {
	var counter atomic.Int64
	server := embedMockServer(&counter)
	defer server.Close()

	pool := []CorpusEntry{
		makeEntry("pool-1", "first pool entry", intPtr(5)),
		makeEntry("pool-2", "second pool entry", intPtr(8)),
	}
	scored := []CorpusEntry{
		makeEntry("scored-1", "first scored entry", nil),
		makeEntry("scored-2", "second scored entry", nil),
	}

	index, _, err := BuildEmbedIndex(context.Background(), "test-model", server.URL, pool, scored, server.Client(), io.Discard)

	s.Require().NoError(err)
	s.Require().Len(index.Pool, 2, "pool should have 2 entries")
	s.Require().Len(index.Scored, 2, "scored should have 2 entries")

	for i, p := range index.Pool {
		s.NotNil(p.Embedding, "pool entry %d embedding should not be nil", i)
		s.NotEmpty(p.Embedding, "pool entry %d embedding should not be empty", i)
	}

	s.NotNil(index.Scored["scored-1"], "scored-1 embedding should exist")
	s.NotNil(index.Scored["scored-2"], "scored-2 embedding should exist")
}

func (s *EmbedSuite) TestBuildEmbedIndex_CollectsLatencies() {
	var counter atomic.Int64
	server := embedMockServer(&counter)
	defer server.Close()

	pool := []CorpusEntry{
		makeEntry("pool-1", "entry one", intPtr(3)),
		makeEntry("pool-2", "entry two", intPtr(7)),
	}
	scored := []CorpusEntry{
		makeEntry("scored-1", "entry three", nil),
	}

	_, latencies, err := BuildEmbedIndex(context.Background(), "test-model", server.URL, pool, scored, server.Client(), io.Discard)

	s.Require().NoError(err)
	s.Require().Len(latencies, 3, "should have 3 latencies (2 pool + 1 scored)")
	for i, lat := range latencies {
		s.Greater(lat, int64(0), "latency %d should be > 0", i)
	}
}

func (s *EmbedSuite) TestBuildEmbedIndex_PrintsProgress() {
	var counter atomic.Int64
	server := embedMockServer(&counter)
	defer server.Close()

	pool := []CorpusEntry{
		makeEntry("pool-1", "entry one", intPtr(5)),
	}
	scored := []CorpusEntry{
		makeEntry("scored-1", "entry two", nil),
	}

	var buf bytes.Buffer
	_, _, err := BuildEmbedIndex(context.Background(), "test-model", server.URL, pool, scored, server.Client(), &buf)

	s.Require().NoError(err)
	s.NotEmpty(buf.String(), "progress writer should have received output")
}

func (s *EmbedSuite) TestRunBenchmark_UsesEmbedIndex() {
	// Mock Ollama server for inference (scoring).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"response":"{\"importance_score\":8.0,\"confidence_score\":0.9,\"reasoning\":\"test\"}","done":true}`)
	}))
	defer server.Close()

	rating5 := 5
	rating8 := 8

	scored := []CorpusEntry{
		{
			ID:           "embed-scored-01",
			Source:       "slack",
			Sender:       "alice",
			Channel:      "#ops",
			Content:      "Production alert fired",
			ExpectedBand: "notified",
			Tags:         []string{"outage"},
		},
	}

	pool := []CorpusEntry{
		{
			ID:         "embed-pool-01",
			Source:     "slack",
			Sender:     "bob",
			Channel:    "#general",
			Content:    "Similar production issue",
			Tags:       []string{"outage"},
			UserRating: &rating8,
		},
		{
			ID:         "embed-pool-02",
			Source:     "email",
			Sender:     "carol@example.com",
			Channel:    "inbox",
			Content:    "Weekly status report",
			Tags:       []string{"routine"},
			UserRating: &rating5,
		},
	}

	// Build an EmbedIndex with pre-computed embeddings.
	embedIndex := &EmbedIndex{
		Pool: []EmbedResult{
			{Entry: pool[0], Embedding: []float32{0.9, 0.1, 0}},
			{Entry: pool[1], Embedding: []float32{0.1, 0.9, 0}},
		},
		Scored: map[string][]float32{
			"embed-scored-01": {1, 0, 0},
		},
	}

	cfg := BenchConfig{
		Baseline:   "test-model",
		Models:     []string{},
		OllamaHost: server.URL,
		Timeout:    5 * time.Second,
		NoFewShot:  false,
		Seed:       42,
	}

	ctx := context.Background()
	report, err := RunBenchmark(ctx, cfg, scored, pool, embedIndex, server.Client(), io.Discard)

	s.Require().NoError(err, "RunBenchmark with embedIndex should not error")
	s.NotEmpty(report.RunResults, "should produce RunResults when using embed index")
}

func (s *EmbedSuite) TestRunBenchmark_NilEmbedIndex_UsesTagSelection() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"response":"{\"importance_score\":6.0,\"confidence_score\":0.5,\"reasoning\":\"tag test\"}","done":true}`)
	}))
	defer server.Close()

	scored := []CorpusEntry{
		{
			ID:           "tag-scored-01",
			Source:       "slack",
			Sender:       "alice",
			Channel:      "#ops",
			Content:      "Server restarted",
			ExpectedBand: "ignored",
			Tags:         []string{"ops"},
		},
	}
	var pool []CorpusEntry

	cfg := BenchConfig{
		Baseline:   "test-model",
		Models:     []string{},
		OllamaHost: server.URL,
		Timeout:    5 * time.Second,
		NoFewShot:  true,
		Seed:       42,
	}

	ctx := context.Background()
	report, err := RunBenchmark(ctx, cfg, scored, pool, nil, server.Client(), io.Discard)

	s.Require().NoError(err, "RunBenchmark with nil embedIndex should not error")
	s.Len(report.RunResults, 1, "should produce 1 RunResult")
}

func (s *EmbedSuite) TestBuildEmbedIndex_ErrorOnEmbedFailure() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pool := []CorpusEntry{
		makeEntry("pool-1", "entry", intPtr(5)),
	}
	scored := []CorpusEntry{
		makeEntry("scored-1", "entry", nil),
	}

	_, _, err := BuildEmbedIndex(context.Background(), "test-model", server.URL, pool, scored, server.Client(), io.Discard)

	s.Error(err, "should return error when embed requests fail")
}
