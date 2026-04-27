package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// jsonReport is the top-level JSON serialization wrapper for benchmark output.
type jsonReport struct {
	Results    []RunResult `json:"results"`
	EmbedModel string      `json:"embed_model,omitempty"`
	EmbedP50Ms int64       `json:"embed_p50_ms,omitempty"`
	EmbedP95Ms int64       `json:"embed_p95_ms,omitempty"`
}

// RenderJSON marshals the report into a JSON object with a "results" key
// and optional embed model stats, then writes it to w.
func RenderJSON(w io.Writer, report BenchReport) error {
	jr := jsonReport{
		Results:    report.RunResults,
		EmbedModel: report.EmbedModel,
		EmbedP50Ms: report.EmbedP50Ms,
		EmbedP95Ms: report.EmbedP95Ms,
	}
	return json.NewEncoder(w).Encode(jr)
}

// Table column widths
const (
	modelColWidth   = 16
	metricColWidth  = 6
	jsonColWidth    = 5
	latencyColWidth = 6
	calibColWidth   = 6
	separatorWidth  = 8 // for header separators
)

// BenchReport holds all data needed to render benchmark output in either
// table or JSON format.
type BenchReport struct {
	BaselineName  string
	CorpusStats   string // e.g. "58 messages (35 scored, 23 rated examples), 1 run(s)"
	Results       map[string]map[int]AggregateMetrics
	ModelOrder    []string
	ExampleCounts []int
	RunResults    []RunResult // per-message results for JSON export
	EmbedModel    string      // empty when tag-based selection used
	EmbedP50Ms    int64       // 0 when no embed model
	EmbedP95Ms    int64       // 0 when no embed model
}

// RenderTable writes an ASCII table summary of the benchmark report to w.
func RenderTable(w io.Writer, report BenchReport) {
	// Header
	fmt.Fprintf(w, "Model Benchmark Results\n")
	fmt.Fprintf(w, "=======================\n")
	fmt.Fprintf(w, "Baseline: %s\n", report.BaselineName)
	fmt.Fprintf(w, "Corpus: %s\n", report.CorpusStats)
	if report.EmbedModel != "" {
		fmt.Fprintf(w, "Embed Model: %s (p50: %dms, p95: %dms)\n", report.EmbedModel, report.EmbedP50Ms, report.EmbedP95Ms)
	}
	fmt.Fprintf(w, "\n")

	// Sort example counts so 0 comes first
	counts := make([]int, len(report.ExampleCounts))
	copy(counts, report.ExampleCounts)
	sort.Ints(counts)

	for _, n := range counts {
		if n == 0 {
			renderBaseSection(w, report)
		} else {
			renderFewShotSection(w, n, report)
		}
	}
}

// baseHeaders are the column headers shared by both base and few-shot tables.
var baseHeaders = []string{"Model", "Band Acc", "FP Rate", "FN Rate", "JSON %", "p50 ms", "p95 ms"}

func renderBaseSection(w io.Writer, report BenchReport) {
	fmt.Fprintf(w, "Base Scoring (0 examples):\n\n")
	renderTableHeader(w, baseHeaders)
	for _, model := range report.ModelOrder {
		metrics, ok := report.Results[model][0]
		if !ok {
			continue
		}
		renderMetricRow(w, model, metrics, "")
	}
	fmt.Fprintf(w, "\n")
}

func renderFewShotSection(w io.Writer, n int, report BenchReport) {
	fmt.Fprintf(w, "Few-Shot Calibration (%d examples):\n\n", n)
	headers := append(baseHeaders, "Cal Lift")
	renderTableHeader(w, headers)
	for _, model := range report.ModelOrder {
		metrics, ok := report.Results[model][n]
		if !ok {
			continue
		}
		baseMetrics := report.Results[model][0]
		lift := CalibrationLift(baseMetrics, metrics)
		liftStr := fmt.Sprintf("%+.1f%%", lift)
		renderMetricRow(w, model, metrics, liftStr)
	}
	fmt.Fprintf(w, "\n")
}

func renderTableHeader(w io.Writer, headers []string) {
	// Build header line
	parts := make([]string, len(headers))
	sepParts := make([]string, len(headers))
	for i, h := range headers {
		if i == 0 {
			parts[i] = fmt.Sprintf("%-*s", modelColWidth, h)
			sepParts[i] = strings.Repeat("-", modelColWidth)
		} else {
			parts[i] = fmt.Sprintf(" %*s ", metricColWidth, h)
			sepParts[i] = strings.Repeat("-", separatorWidth)
		}
	}
	fmt.Fprintf(w, "%s\n", strings.Join(parts, "|"))
	fmt.Fprintf(w, "%s\n", strings.Join(sepParts, "|"))
}

// renderMetricRow outputs a single row of metrics. If calibrationLift is empty,
// no calibration column is included.
func renderMetricRow(w io.Writer, model string, metrics AggregateMetrics, calibrationLift string) {
	if calibrationLift == "" {
		fmt.Fprintf(w, "%-*s| %*s | %*s | %*s | %*s | %*d | %*d\n",
			modelColWidth, model,
			metricColWidth, fmtPct(metrics.BandAccuracy),
			metricColWidth, fmtPct(metrics.FalsePositiveRate),
			metricColWidth, fmtPct(metrics.FalseNegativeRate),
			jsonColWidth, fmtPct(metrics.JSONCompliance),
			latencyColWidth, metrics.P50Ms,
			latencyColWidth, metrics.P95Ms,
		)
	} else {
		fmt.Fprintf(w, "%-*s| %*s | %*s | %*s | %*s | %*d | %*d | %*s\n",
			modelColWidth, model,
			metricColWidth, fmtPct(metrics.BandAccuracy),
			metricColWidth, fmtPct(metrics.FalsePositiveRate),
			metricColWidth, fmtPct(metrics.FalseNegativeRate),
			jsonColWidth, fmtPct(metrics.JSONCompliance),
			latencyColWidth, metrics.P50Ms,
			latencyColWidth, metrics.P95Ms,
			calibColWidth, calibrationLift,
		)
	}
}

func fmtPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}
