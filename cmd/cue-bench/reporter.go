package cuebench

import "io"

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
	// noop stub — implementation pending
}
