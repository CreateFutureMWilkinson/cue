package cuebench_test

import (
	"bytes"
	"encoding/json"
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

// --- JSON Reporter Suite ---

type JSONReporterSuite struct {
	suite.Suite
}

func TestJSONReporter(t *testing.T) { suite.Run(t, new(JSONReporterSuite)) }

// helperRunResults builds a slice of RunResult for use across JSON reporter tests.
func (s *JSONReporterSuite) helperRunResults() []cuebench.RunResult {
	return []cuebench.RunResult{
		{
			ModelName:    "neural-chat",
			EntryID:      "entry-001",
			ExampleCount: 0,
			InferenceMs:  120,
			IS:           8.5,
			CS:           0.9,
			Reasoning:    "server outage mentioned",
			JSONValid:    true,
			Band:         "notified",
			ExpectedBand: "notified",
			BandCorrect:  true,
		},
		{
			ModelName:    "neural-chat",
			EntryID:      "entry-002",
			ExampleCount: 0,
			InferenceMs:  95,
			IS:           3.0,
			CS:           0.7,
			Reasoning:    "casual message",
			JSONValid:    true,
			Band:         "ignored",
			ExpectedBand: "ignored",
			BandCorrect:  true,
		},
	}
}

// helperJSONReport builds a BenchReport populated with RunResults for JSON tests.
func (s *JSONReporterSuite) helperJSONReport() cuebench.BenchReport {
	return cuebench.BenchReport{
		BaselineName:  "neural-chat",
		CorpusStats:   "2 messages, 1 run(s)",
		Results:       map[string]map[int]cuebench.AggregateMetrics{},
		ModelOrder:    []string{"neural-chat"},
		ExampleCounts: []int{0},
		RunResults:    s.helperRunResults(),
	}
}

func (s *JSONReporterSuite) TestRenderJSON_ProducesValidJSON() {
	var buf bytes.Buffer
	report := s.helperJSONReport()

	err := cuebench.RenderJSON(&buf, report)
	s.Require().NoError(err, "RenderJSON must not return an error")

	var decoded []map[string]any
	err = json.Unmarshal(buf.Bytes(), &decoded)
	s.NoError(err, "RenderJSON output must be valid JSON that unmarshals into []map[string]any")
}

func (s *JSONReporterSuite) TestRenderJSON_ContainsRunResults() {
	var buf bytes.Buffer
	report := s.helperJSONReport()

	err := cuebench.RenderJSON(&buf, report)
	s.Require().NoError(err, "RenderJSON must not return an error")

	var decoded []map[string]any
	err = json.Unmarshal(buf.Bytes(), &decoded)
	s.Require().NoError(err, "output must be valid JSON")

	s.Equal(len(report.RunResults), len(decoded),
		"JSON array length must match len(report.RunResults)")
}

func (s *JSONReporterSuite) TestRenderJSON_FieldsPresent() {
	var buf bytes.Buffer
	report := s.helperJSONReport()

	err := cuebench.RenderJSON(&buf, report)
	s.Require().NoError(err, "RenderJSON must not return an error")

	var decoded []map[string]any
	err = json.Unmarshal(buf.Bytes(), &decoded)
	s.Require().NoError(err, "output must be valid JSON")
	s.Require().NotEmpty(decoded, "JSON array must not be empty")

	first := decoded[0]
	expectedKeys := []string{"model_name", "entry_id", "band"}
	for _, key := range expectedKeys {
		_, exists := first[key]
		s.True(exists, "first JSON element must contain key %q", key)
	}
}
