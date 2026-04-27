package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	report, err := RunBenchmark(ctx, cfg, scored, pool, server.Client(), io.Discard)

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

func (s *BenchmarkSuite) TestRunBenchmark_ProgressOutput() {
	// Mock Ollama server returning valid scored responses.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"response":"{\"importance_score\":6.0,\"confidence_score\":0.5,\"reasoning\":\"bench progress test\"}","done":true}`)
	}))
	defer server.Close()

	scored := []CorpusEntry{
		{
			ID:           "prog-01",
			Source:       "slack",
			Sender:       "alice",
			Channel:      "#ops",
			Content:      "Server alert triggered",
			ExpectedBand: "buffered",
			Tags:         []string{"alert"},
		},
		{
			ID:           "prog-02",
			Source:       "email",
			Sender:       "bob@example.com",
			Channel:      "inbox",
			Content:      "Meeting notes from standup",
			ExpectedBand: "ignored",
			Tags:         []string{"routine"},
		},
		{
			ID:           "prog-03",
			Source:       "slack",
			Sender:       "carol",
			Channel:      "#general",
			Content:      "Reminder about Friday deadline",
			ExpectedBand: "ignored",
			Tags:         []string{"reminder"},
		},
	}
	var pool []CorpusEntry

	cfg := BenchConfig{
		Baseline:   "progress-model",
		Models:     []string{},
		OllamaHost: server.URL,
		Timeout:    5 * time.Second,
		NoFewShot:  true,
		Seed:       42,
	}

	var progressBuf bytes.Buffer
	ctx := context.Background()
	_, err := RunBenchmark(ctx, cfg, scored, pool, server.Client(), &progressBuf)
	s.Require().NoError(err, "RunBenchmark should not return an error")

	output := progressBuf.String()
	s.NotEmpty(output, "progress output should not be empty")

	// Should contain model name.
	s.Contains(output, "progress-model",
		"progress output should contain the model name")

	// Should contain the shot label for 0-shot.
	s.Contains(output, "0-shot",
		"progress output should contain the example count label")

	// Should contain the total entry count.
	s.Contains(output, "3/3",
		"progress output should contain entry progress (current/total)")

	// Should contain 100% at completion.
	s.Contains(output, "100%",
		"progress output should show 100%% when complete")
}
