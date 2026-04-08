package cuebench

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// BenchReport holds all data needed to render benchmark output in either
// table or JSON format.
type BenchReport struct {
	BaselineName  string
	CorpusStats   string // e.g. "58 messages (35 scored, 23 rated examples), 1 run(s)"
	Results       map[string]map[int]AggregateMetrics
	ModelOrder    []string
	ExampleCounts []int
}

// RenderTable writes an ASCII table summary of the benchmark report to w.
func RenderTable(w io.Writer, report BenchReport) {
	// Header
	fmt.Fprintf(w, "Model Benchmark Results\n")
	fmt.Fprintf(w, "=======================\n")
	fmt.Fprintf(w, "Baseline: %s\n", report.BaselineName)
	fmt.Fprintf(w, "Corpus: %s\n", report.CorpusStats)
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
		fmt.Fprintf(w, "%-16s| %6s | %6s | %6s | %5s | %6d | %6d\n",
			model,
			fmtPct(metrics.BandAccuracy),
			fmtPct(metrics.FalsePositiveRate),
			fmtPct(metrics.FalseNegativeRate),
			fmtPct(metrics.JSONCompliance),
			metrics.P50Ms,
			metrics.P95Ms,
		)
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
		fmt.Fprintf(w, "%-16s| %6s | %6s | %6s | %5s | %6d | %6d | %6s\n",
			model,
			fmtPct(metrics.BandAccuracy),
			fmtPct(metrics.FalsePositiveRate),
			fmtPct(metrics.FalseNegativeRate),
			fmtPct(metrics.JSONCompliance),
			metrics.P50Ms,
			metrics.P95Ms,
			liftStr,
		)
	}
	fmt.Fprintf(w, "\n")
}

func renderTableHeader(w io.Writer, headers []string) {
	// Build header line
	parts := make([]string, len(headers))
	sepParts := make([]string, len(headers))
	for i, h := range headers {
		if i == 0 {
			parts[i] = fmt.Sprintf("%-16s", h)
			sepParts[i] = strings.Repeat("-", 16)
		} else {
			parts[i] = fmt.Sprintf(" %6s ", h)
			sepParts[i] = strings.Repeat("-", 8)
		}
	}
	fmt.Fprintf(w, "%s\n", strings.Join(parts, "|"))
	fmt.Fprintf(w, "%s\n", strings.Join(sepParts, "|"))
}

func fmtPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}
