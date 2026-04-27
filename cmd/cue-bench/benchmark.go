package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
)

// scoreEntry executes a single HTTP request to Ollama and parses the response.
// Returns the parsed ScorerResponse, inference duration in milliseconds,
// JSON validity flag, and any error encountered.
func scoreEntry(ctx context.Context, model, prompt, host string, httpClient *http.Client) (decisionengine.ScorerResponse, int64, bool, error) {
	// Build Ollama request body.
	reqBody := decisionengine.OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return decisionengine.ScorerResponse{}, 0, false, fmt.Errorf("marshal request: %w", err)
	}

	url := host + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return decisionengine.ScorerResponse{}, 0, false, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := httpClient.Do(httpReq)
	inferenceMs := time.Since(start).Milliseconds()

	if err != nil {
		return decisionengine.ScorerResponse{}, inferenceMs, false, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Read response body.
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return decisionengine.ScorerResponse{}, inferenceMs, false, fmt.Errorf("read response: %w", err)
	}

	// Parse outer JSON into OllamaResponse.
	var ollamaResp decisionengine.OllamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return decisionengine.ScorerResponse{}, inferenceMs, false, fmt.Errorf("parse outer JSON: %w", err)
	}

	// Try to parse inner JSON into ScorerResponse.
	var sr decisionengine.ScorerResponse
	if err := json.Unmarshal([]byte(ollamaResp.Response), &sr); err != nil {
		return decisionengine.ScorerResponse{}, inferenceMs, false, fmt.Errorf("parse inner JSON: %w", err)
	}

	return sr, inferenceMs, true, nil
}

// exampleCounts returns the list of example counts to test based on configuration.
func exampleCounts(cfg BenchConfig) []int {
	if cfg.NoFewShot {
		return []int{0}
	}
	return []int{0, 1, 3, 5}
}

// formatCorpusStats formats the corpus statistics string for BenchReport.
func formatCorpusStats(scoredCount, poolCount, ratedCount, runs int) string {
	totalMessages := scoredCount + poolCount
	return fmt.Sprintf("%d messages (%d scored, %d rated examples), %d run(s)",
		totalMessages, scoredCount, ratedCount, runs)
}

// writeProgress writes a progress line to the given writer.
func writeProgress(w io.Writer, modelName string, exampleCount, current, total int) {
	if w == nil {
		return
	}
	const barWidth = 20
	pct := current * 100 / total
	filled := barWidth * current / total
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Fprintf(w, "\r%s [%d-shot]  %s  %d/%d  (%d%%)", modelName, exampleCount, bar, current, total, pct)
}

// RunBenchmark executes the scoring loop: for each model (baseline + cfg.Models)
// x each example count x each scored entry, it selects few-shot examples from
// pool, builds a prompt, POSTs to Ollama, parses the response, derives the
// routing band, and appends a RunResult. The returned BenchReport contains
// ModelOrder, RunResults, aggregated Results, BaselineName, and ExampleCounts.
func RunBenchmark(ctx context.Context, cfg BenchConfig, scored []CorpusEntry, pool []CorpusEntry, httpClient *http.Client, progressWriter io.Writer) (BenchReport, error) {
	// 1. Build model list: baseline first, then additional models.
	models := append([]string{cfg.Baseline}, cfg.Models...)

	// 2. Build example counts.
	counts := exampleCounts(cfg)

	// 3. Scoring loop: model x exampleCount x entry.
	var allResults []RunResult

	for _, modelName := range models {
		for _, exampleCount := range counts {
			total := len(scored)
			current := 0
			for _, scoredEntry := range scored {
				// a. Select few-shot examples from pool.
				selectedExamples := SelectExamples(scoredEntry, pool, exampleCount, cfg.Seed)

				// b. Convert to decisionengine.FewShotExample.
				fewShotExamples := make([]decisionengine.FewShotExample, len(selectedExamples))
				for i, example := range selectedExamples {
					fewShotExamples[i] = decisionengine.FewShotExample{
						Content:    example.Content,
						UserRating: *example.UserRating,
					}
				}

				// c. Build a repository.Message from the entry.
				msg := repository.Message{
					Source:     scoredEntry.Source,
					Sender:     scoredEntry.Sender,
					Channel:    scoredEntry.Channel,
					RawContent: scoredEntry.Content,
				}

				// d. Build prompt.
				prompt := decisionengine.BuildPromptWithExamples(&msg, fewShotExamples)

				// e. Initialize result with known values.
				result := RunResult{
					ModelName:    modelName,
					EntryID:      scoredEntry.ID,
					ExampleCount: exampleCount,
					ExpectedBand: scoredEntry.ExpectedBand,
				}

				// f. Score the entry via Ollama.
				scorerResp, inferenceMs, jsonValid, err := scoreEntry(ctx, modelName, prompt, cfg.OllamaHost, httpClient)

				result.InferenceMs = inferenceMs
				result.JSONValid = jsonValid

				if err != nil {
					// Fallback scoring on any error.
					result.IS = 7
					result.CS = 0.0
					result.Reasoning = fmt.Sprintf("error: %v", err)
					result.Band = DeriveBand(7, 0)
					result.BandCorrect = (result.Band == scoredEntry.ExpectedBand)
				} else {
					// g. Populate successful scoring result.
					result.IS = scorerResp.ImportanceScore
					result.CS = scorerResp.ConfidenceScore
					result.Reasoning = scorerResp.Reasoning

					// h. Derive band.
					result.Band = DeriveBand(scorerResp.ImportanceScore, scorerResp.ConfidenceScore)

					// i. Check band correctness.
					result.BandCorrect = (result.Band == scoredEntry.ExpectedBand)
				}

				allResults = append(allResults, result)
				current++
				writeProgress(progressWriter, modelName, exampleCount, current, total)
			}
			if progressWriter != nil {
				fmt.Fprint(progressWriter, "\n")
			}
		}
	}

	// 4. Populate BenchReport.
	report := BenchReport{
		ModelOrder:    models,
		BaselineName:  cfg.Baseline,
		ExampleCounts: counts,
		RunResults:    allResults,
		Results:       make(map[string]map[int]AggregateMetrics),
	}

	// Compute aggregated results: for each model, for each exampleCount.
	for _, modelName := range models {
		report.Results[modelName] = make(map[int]AggregateMetrics)
		for _, exampleCount := range counts {
			var filteredResults []RunResult
			for _, result := range allResults {
				if result.ModelName == modelName && result.ExampleCount == exampleCount {
					filteredResults = append(filteredResults, result)
				}
			}
			report.Results[modelName][exampleCount] = CalcMetrics(filteredResults)
		}
	}

	// CorpusStats.
	ratedCount := len(RatedEntries(pool))
	report.CorpusStats = formatCorpusStats(len(scored), len(pool), ratedCount, cfg.Runs)

	return report, nil
}
