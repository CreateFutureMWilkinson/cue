package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type BenchmarkSuite struct {
	suite.Suite
}

func TestBenchmark(t *testing.T) { suite.Run(t, new(BenchmarkSuite)) }

func (s *BenchmarkSuite) TestRunBenchmark_ProducesRunResults() {
	// Start a mock Ollama server that always returns a valid scored response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"response":"{\"importance_score\":8.0,\"confidence_score\":0.9,\"reasoning\":\"test\"}","done":true}`)
	}))
	defer server.Close()

	// Two scored entries (UserRating nil), zero rated pool entries.
	scored := []CorpusEntry{
		{
			ID:           "bench-01",
			Source:       "slack",
			Sender:       "alice",
			Channel:      "#ops",
			Content:      "Production database is unreachable",
			ExpectedBand: "notified",
			Tags:         []string{"outage"},
			UserRating:   nil,
		},
		{
			ID:           "bench-02",
			Source:       "email",
			Sender:       "bob@example.com",
			Channel:      "inbox",
			Content:      "Weekly status update",
			ExpectedBand: "ignored",
			Tags:         []string{"routine"},
			UserRating:   nil,
		},
	}
	var pool []CorpusEntry // empty pool — no rated examples available

	cfg := BenchConfig{
		Baseline:   "test-model",
		Models:     []string{},
		OllamaHost: server.URL,
		Timeout:    5 * time.Second,
		NoFewShot:  true, // only example_count=0
		Seed:       42,
	}

	ctx := context.Background()
	report, err := RunBenchmark(ctx, cfg, scored, pool, server.Client())

	s.Require().NoError(err, "RunBenchmark should not return an error")

	// With NoFewShot=true, only example_count=0 is used.
	// 1 model (baseline) x 1 example_count (0) x 2 scored entries = 2 results.
	s.Require().Len(report.RunResults, 2,
		"expected 2 RunResults (2 entries x 1 model x 1 example count)")

	s.Equal([]string{"test-model"}, report.ModelOrder,
		"ModelOrder should contain only the baseline model")

	s.Equal("test-model", report.RunResults[0].ModelName,
		"first result should be for the baseline model")
	s.Equal(0, report.RunResults[0].ExampleCount,
		"first result should have ExampleCount=0")
}
