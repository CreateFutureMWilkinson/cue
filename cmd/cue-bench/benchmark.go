package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
)

// RunBenchmark executes the scoring loop: for each model (baseline + cfg.Models)
// x each example count x each scored entry, it selects few-shot examples from
// pool, builds a prompt, POSTs to Ollama, parses the response, derives the
// routing band, and appends a RunResult. The returned BenchReport contains
// ModelOrder, RunResults, aggregated Results, BaselineName, and ExampleCounts.
func RunBenchmark(ctx context.Context, cfg BenchConfig, scored []CorpusEntry, pool []CorpusEntry, httpClient *http.Client) (BenchReport, error) {
	// 1. Build model list: baseline first, then additional models.
	models := append([]string{cfg.Baseline}, cfg.Models...)

	// 2. Build example counts.
	var exampleCounts []int
	if cfg.NoFewShot {
		exampleCounts = []int{0}
	} else {
		exampleCounts = []int{0, 1, 3, 5}
	}

	// 3. Scoring loop: model x exampleCount x entry.
	var allResults []RunResult

	for _, model := range models {
		for _, exampleCount := range exampleCounts {
			for _, entry := range scored {
				// a. Select few-shot examples from pool.
				exs := SelectExamples(entry, pool, exampleCount, cfg.Seed)

				// b. Convert to decisionengine.FewShotExample.
				fewShot := make([]decisionengine.FewShotExample, len(exs))
				for i, e := range exs {
					fewShot[i] = decisionengine.FewShotExample{
						Content:    e.Content,
						UserRating: *e.UserRating,
					}
				}

				// c. Build a repository.Message from the entry.
				msg := repository.Message{
					Source:     entry.Source,
					Sender:     entry.Sender,
					Channel:    entry.Channel,
					RawContent: entry.Content,
				}

				// d. Build prompt.
				prompt := decisionengine.BuildPromptWithExamples(&msg, fewShot)

				// e. Build Ollama request body.
				reqBody := decisionengine.OllamaRequest{
					Model:  model,
					Prompt: prompt,
					Stream: false,
					Format: "json",
				}

				// f. POST to Ollama.
				result := RunResult{
					ModelName:    model,
					EntryID:      entry.ID,
					ExampleCount: exampleCount,
					ExpectedBand: entry.ExpectedBand,
				}

				bodyBytes, err := json.Marshal(reqBody)
				if err != nil {
					result.JSONValid = false
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = "error: " + err.Error()
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == entry.ExpectedBand)
					allResults = append(allResults, result)
					continue
				}

				url := cfg.OllamaHost + "/api/generate"
				httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
				if err != nil {
					result.JSONValid = false
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = "error: " + err.Error()
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == entry.ExpectedBand)
					allResults = append(allResults, result)
					continue
				}
				httpReq.Header.Set("Content-Type", "application/json")

				start := time.Now()
				resp, err := httpClient.Do(httpReq)
				result.InferenceMs = time.Since(start).Milliseconds()

				if err != nil {
					result.JSONValid = false
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = "error: " + err.Error()
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == entry.ExpectedBand)
					allResults = append(allResults, result)
					continue
				}

				// g. Read response body.
				respBody, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					result.JSONValid = false
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = "error: " + err.Error()
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == entry.ExpectedBand)
					allResults = append(allResults, result)
					continue
				}

				// h. Parse outer JSON into OllamaResponse.
				var ollamaResp decisionengine.OllamaResponse
				if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
					result.JSONValid = false
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = "error: " + err.Error()
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == entry.ExpectedBand)
					allResults = append(allResults, result)
					continue
				}

				// i. Try to parse inner JSON into ScorerResponse.
				var sr decisionengine.ScorerResponse
				if err := json.Unmarshal([]byte(ollamaResp.Response), &sr); err != nil {
					result.JSONValid = false
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = "error: " + err.Error()
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == entry.ExpectedBand)
					allResults = append(allResults, result)
					continue
				}

				result.JSONValid = true
				result.IS = sr.ImportanceScore
				result.CS = sr.ConfidenceScore
				result.Reasoning = sr.Reasoning

				// j. Derive band.
				result.Band = DeriveBand(sr.ImportanceScore, sr.ConfidenceScore)

				// k. Check band correctness.
				result.BandCorrect = (result.Band == entry.ExpectedBand)

				allResults = append(allResults, result)
			}
		}
	}

	// 4. Populate BenchReport.
	report := BenchReport{
		ModelOrder:    models,
		BaselineName:  cfg.Baseline,
		ExampleCounts: exampleCounts,
		RunResults:    allResults,
		Results:       make(map[string]map[int]AggregateMetrics),
	}

	// Compute aggregated results: for each model, for each exampleCount.
	for _, model := range models {
		report.Results[model] = make(map[int]AggregateMetrics)
		for _, ec := range exampleCounts {
			var filtered []RunResult
			for _, r := range allResults {
				if r.ModelName == model && r.ExampleCount == ec {
					filtered = append(filtered, r)
				}
			}
			report.Results[model][ec] = CalcMetrics(filtered)
		}
	}

	// CorpusStats.
	totalMessages := len(scored) + len(pool)
	ratedCount := len(RatedEntries(pool))
	report.CorpusStats = fmt.Sprintf("%d messages (%d scored, %d rated examples), %d run(s)",
		totalMessages, len(scored), ratedCount, cfg.Runs)

	return report, nil
}
