package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MetricsSuite struct {
	suite.Suite
}

func TestMetrics(t *testing.T) { suite.Run(t, new(MetricsSuite)) }

func (s *MetricsSuite) TestDeriveBand_NotifiedWhenHighISAndHighCS() {
	band := DeriveBand(8.0, 0.9)
	s.Equal("notified", band, "IS=8.0, CS=0.9 should route to notified")
}

func (s *MetricsSuite) TestDeriveBand_BufferedWhenHighISAndLowCS() {
	band := DeriveBand(7.5, 0.5)
	s.Equal("buffered", band, "IS=7.5, CS=0.5 should route to buffered")
}

func (s *MetricsSuite) TestDeriveBand_IgnoredWhenLowIS() {
	band := DeriveBand(4.0, 0.9)
	s.Equal("ignored", band, "IS=4.0, CS=0.9 should route to ignored")
}

func (s *MetricsSuite) TestDeriveBand_BoundaryIS7CS08_Notified() {
	band := DeriveBand(7.0, 0.8)
	s.Equal("notified", band, "IS=7.0, CS=0.8 should route to notified (boundary)")
}

func (s *MetricsSuite) TestDeriveBand_BoundaryIS7CS079_Buffered() {
	band := DeriveBand(7.0, 0.79)
	s.Equal("buffered", band, "IS=7.0, CS=0.79 should route to buffered (just below confidence threshold)")
}

// AggregateMetricsSuite tests the CalcMetrics function.
type AggregateMetricsSuite struct {
	suite.Suite
}

func TestAggregateMetrics(t *testing.T) { suite.Run(t, new(AggregateMetricsSuite)) }

func (s *AggregateMetricsSuite) TestCalcMetrics_BandAccuracy() {
	results := []RunResult{
		{BandCorrect: true},
		{BandCorrect: true},
		{BandCorrect: false},
	}
	m := CalcMetrics(results)
	s.InDelta(66.67, m.BandAccuracy, 0.01, "2/3 correct should be ~66.67%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_FalsePositiveRate() {
	results := []RunResult{
		{ExpectedBand: "ignored", Band: "notified"}, // false positive
		{ExpectedBand: "ignored", Band: "ignored"},  // correct
	}
	m := CalcMetrics(results)
	s.InDelta(50.0, m.FalsePositiveRate, 0.01, "1/2 expected-ignored got notified should be 50%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_FalseNegativeRate() {
	results := []RunResult{
		{ExpectedBand: "notified", Band: "ignored"},  // false negative
		{ExpectedBand: "notified", Band: "notified"}, // correct
	}
	m := CalcMetrics(results)
	s.InDelta(50.0, m.FalseNegativeRate, 0.01, "1/2 expected-notified got ignored should be 50%%")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_JSONCompliance() {
	results := []RunResult{
		{JSONValid: true},
		{JSONValid: true},
		{JSONValid: true},
		{JSONValid: false},
	}
	m := CalcMetrics(results)
	s.InDelta(75.0, m.JSONCompliance, 0.01, "3/4 valid JSON should be 75%%")
}

// CalibrationLiftSuite tests the CalibrationLift function.
type CalibrationLiftSuite struct {
	suite.Suite
}

func TestCalibrationLift(t *testing.T) { suite.Run(t, new(CalibrationLiftSuite)) }

func (s *CalibrationLiftSuite) TestCalibrationLift_PositiveLift() {
	base := AggregateMetrics{BandAccuracy: 75.0}
	full := AggregateMetrics{BandAccuracy: 82.0}
	lift := CalibrationLift(base, full)
	s.InDelta(7.0, lift, 0.001, "82.0 - 75.0 should yield positive lift of 7.0")
}

func (s *CalibrationLiftSuite) TestCalibrationLift_ZeroLift() {
	base := AggregateMetrics{BandAccuracy: 80.0}
	full := AggregateMetrics{BandAccuracy: 80.0}
	lift := CalibrationLift(base, full)
	s.InDelta(0.0, lift, 0.001, "equal band accuracy should yield zero lift")
}

func (s *CalibrationLiftSuite) TestCalibrationLift_NegativeLift() {
	base := AggregateMetrics{BandAccuracy: 85.0}
	full := AggregateMetrics{BandAccuracy: 80.0}
	lift := CalibrationLift(base, full)
	s.InDelta(-5.0, lift, 0.001, "80.0 - 85.0 should yield negative lift of -5.0")
}

func (s *AggregateMetricsSuite) TestCalcMetrics_Percentiles() {
	results := make([]RunResult, 10)
	for i := 0; i < 10; i++ {
		results[i] = RunResult{InferenceMs: int64((i + 1) * 10)}
	}
	m := CalcMetrics(results)
	s.InDelta(50, float64(m.P50Ms), 10, "P50 of [10..100] should be ~50ms")
	s.InDelta(95, float64(m.P95Ms), 10, "P95 of [10..100] should be ~95-100ms")
}
