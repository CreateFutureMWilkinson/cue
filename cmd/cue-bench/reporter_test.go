package cuebench_test

import (
	"bytes"
	"testing"

	cuebench "github.com/CreateFutureMWilkinson/cue/cmd/cue-bench"
	"github.com/stretchr/testify/suite"
)

type ReporterSuite struct {
	suite.Suite
}

func TestReporter(t *testing.T) { suite.Run(t, new(ReporterSuite)) }

// helperReport builds a BenchReport with one model and both 0-example (base)
// and 5-example (few-shot) results for use across test methods.
func (s *ReporterSuite) helperReport() cuebench.BenchReport {
	return cuebench.BenchReport{
		BaselineName: "neural-chat",
		CorpusStats:  "58 messages (35 scored, 23 rated examples), 1 run(s)",
		Results: map[string]map[int]cuebench.AggregateMetrics{
			"neural-chat": {
				0: cuebench.AggregateMetrics{
					BandAccuracy:      72.5,
					FalsePositiveRate: 18.3,
					FalseNegativeRate: 9.1,
					JSONCompliance:    95.0,
					P50Ms:             120,
					P95Ms:             340,
				},
				5: cuebench.AggregateMetrics{
					BandAccuracy:      80.2,
					FalsePositiveRate: 12.1,
					FalseNegativeRate: 7.8,
					JSONCompliance:    97.5,
					P50Ms:             150,
					P95Ms:             410,
				},
			},
		},
		ModelOrder:    []string{"neural-chat"},
		ExampleCounts: []int{0, 5},
	}
}

func (s *ReporterSuite) TestRenderTable_ContainsHeader() {
	var buf bytes.Buffer
	report := s.helperReport()

	cuebench.RenderTable(&buf, report)
	output := buf.String()

	s.Contains(output, "Model Benchmark Results",
		"output must contain the main header 'Model Benchmark Results'")
	s.Contains(output, "neural-chat",
		"output must contain the baseline model name")
}

func (s *ReporterSuite) TestRenderTable_ContainsBaseSection() {
	var buf bytes.Buffer
	report := s.helperReport()

	cuebench.RenderTable(&buf, report)
	output := buf.String()

	s.Contains(output, "Base Scoring",
		"output must contain the 'Base Scoring' section header")
	s.Contains(output, "Model",
		"output must contain the 'Model' column header")
	s.Contains(output, "Band Acc",
		"output must contain the 'Band Acc' column header")
	s.Contains(output, "FP Rate",
		"output must contain the 'FP Rate' column header")
	s.Contains(output, "FN Rate",
		"output must contain the 'FN Rate' column header")
	s.Contains(output, "JSON %",
		"output must contain the 'JSON %%' column header")
	s.Contains(output, "p50 ms",
		"output must contain the 'p50 ms' column header")
	s.Contains(output, "p95 ms",
		"output must contain the 'p95 ms' column header")
}

func (s *ReporterSuite) TestRenderTable_ContainsModelRow() {
	var buf bytes.Buffer
	report := s.helperReport()

	cuebench.RenderTable(&buf, report)
	output := buf.String()

	s.Contains(output, "neural-chat",
		"output must contain the model name in a data row")
	s.Contains(output, "72.5%",
		"output must contain the band accuracy formatted to 1 decimal place with %%")
}

func (s *ReporterSuite) TestRenderTable_ContainsFewShotSection() {
	var buf bytes.Buffer
	report := s.helperReport()

	cuebench.RenderTable(&buf, report)
	output := buf.String()

	s.Contains(output, "Few-Shot Calibration",
		"output must contain the 'Few-Shot Calibration' section header when ExampleCounts includes 5")
	s.Contains(output, "Cal Lift",
		"output must contain the 'Cal Lift' column in the few-shot section")
}
