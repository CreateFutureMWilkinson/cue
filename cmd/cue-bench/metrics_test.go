package cuebench_test

import (
	"testing"

	cuebench "github.com/CreateFutureMWilkinson/cue/cmd/cue-bench"
	"github.com/stretchr/testify/suite"
)

type MetricsSuite struct {
	suite.Suite
}

func TestMetrics(t *testing.T) { suite.Run(t, new(MetricsSuite)) }

func (s *MetricsSuite) TestDeriveBand_NotifiedWhenHighISAndHighCS() {
	band := cuebench.DeriveBand(8.0, 0.9)
	s.Equal("notified", band, "IS=8.0, CS=0.9 should route to notified")
}

func (s *MetricsSuite) TestDeriveBand_BufferedWhenHighISAndLowCS() {
	band := cuebench.DeriveBand(7.5, 0.5)
	s.Equal("buffered", band, "IS=7.5, CS=0.5 should route to buffered")
}

func (s *MetricsSuite) TestDeriveBand_IgnoredWhenLowIS() {
	band := cuebench.DeriveBand(4.0, 0.9)
	s.Equal("ignored", band, "IS=4.0, CS=0.9 should route to ignored")
}

func (s *MetricsSuite) TestDeriveBand_BoundaryIS7CS08_Notified() {
	band := cuebench.DeriveBand(7.0, 0.8)
	s.Equal("notified", band, "IS=7.0, CS=0.8 should route to notified (boundary)")
}

func (s *MetricsSuite) TestDeriveBand_BoundaryIS7CS079_Buffered() {
	band := cuebench.DeriveBand(7.0, 0.79)
	s.Equal("buffered", band, "IS=7.0, CS=0.79 should route to buffered (just below confidence threshold)")
}

// AggregateMetricsSuite tests the CalcMetrics function.
type AggregateMetricsSuite struct {
	suite.Suite
}

func TestAggregateMetrics(t *testing.T) { suite.Run(t, new(AggregateMetricsSuite)) }

func (s *AggregateMetricsSuite) TestCalcMetrics_BandAccuracy() {
	results := []cuebench.RunResult{
		{BandCorrect: true},
		{BandCorrect: true},
		{BandCorrect: false},
	}
	m := cuebench.CalcMetrics(results)
	s.InDelta(66.67, m.BandAccuracy, 0.01, "2/3 correct should be ~66.67%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_FalsePositiveRate() {
	results := []cuebench.RunResult{
		{ExpectedBand: "ignored", Band: "notified"}, // false positive
		{ExpectedBand: "ignored", Band: "ignored"},   // correct
	}
	m := cuebench.CalcMetrics(results)
	s.InDelta(50.0, m.FalsePositiveRate, 0.01, "1/2 expected-ignored got notified should be 50%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_FalseNegativeRate() {
	results := []cuebench.RunResult{
		{ExpectedBand: "notified", Band: "ignored"},  // false negative
		{ExpectedBand: "notified", Band: "notified"},  // correct
	}
	m := cuebench.CalcMetrics(results)
	s.InDelta(50.0, m.FalseNegativeRate, 0.01, "1/2 expected-notified got ignored should be 50%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_JSONCompliance() {
	results := []cuebench.RunResult{
		{JSONValid: true},
		{JSONValid: true},
		{JSONValid: true},
		{JSONValid: false},
	}
	m := cuebench.CalcMetrics(results)
	s.InDelta(75.0, m.JSONCompliance, 0.01, "3/4 valid JSON should be 75%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_Percentiles() {
	results := make([]cuebench.RunResult, 10)
	for i := 0; i < 10; i++ {
		results[i] = cuebench.RunResult{InferenceMs: int64((i + 1) * 10)}
	}
	m := cuebench.CalcMetrics(results)
	s.InDelta(50, float64(m.P50Ms), 10, "P50 of [10..100] should be ~50ms")
	s.InDelta(95, float64(m.P95Ms), 10, "P95 of [10..100] should be ~95-100ms")
}
