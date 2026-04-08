package main

import (
	"context"
	"net/http"
)

// RunBenchmark executes the scoring loop: for each model (baseline + cfg.Models)
// x each example count x each scored entry, it selects few-shot examples from
// pool, builds a prompt, POSTs to Ollama, parses the response, derives the
// routing band, and appends a RunResult. The returned BenchReport contains
// ModelOrder, RunResults, aggregated Results, BaselineName, and ExampleCounts.
func RunBenchmark(ctx context.Context, cfg BenchConfig, scored []CorpusEntry, pool []CorpusEntry, httpClient *http.Client) (BenchReport, error) {
	return BenchReport{}, ErrNotImplemented
}
