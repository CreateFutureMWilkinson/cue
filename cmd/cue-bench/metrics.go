package cuebench

import "errors"

// ErrNotImplemented is returned by stub functions that have not yet been
// replaced with real implementations.
var ErrNotImplemented = errors.New("not implemented")

// Routing threshold constants matching production values.
const (
	importanceThreshold = 7.0
	confidenceThreshold = 0.8
)

// RunResult holds per-message scoring output for one model x entry x
// example-count combination.
type RunResult struct {
	ModelName    string
	EntryID      string
	ExampleCount int
	InferenceMs  int64
	IS           float64
	CS           float64
	Reasoning    string
	JSONValid    bool
	Band         string // derived by DeriveBand
	ExpectedBand string // ground-truth band from calibration entry
	BandCorrect  bool   // Band == entry.ExpectedBand
}

// AggregateMetrics holds computed aggregate statistics over a set of RunResults.
type AggregateMetrics struct {
	BandAccuracy      float64 // % messages routed to correct band
	FalsePositiveRate float64 // % ignored messages incorrectly notified
	FalseNegativeRate float64 // % notified messages incorrectly ignored
	JSONCompliance    float64 // % responses that were valid JSON
	P50Ms             int64   // median inference time
	P95Ms             int64   // 95th percentile inference time
}

// CalcMetrics computes aggregate statistics from a slice of RunResult.
func CalcMetrics(results []RunResult) AggregateMetrics {
	return AggregateMetrics{}
}

// DeriveBand derives the routing band from importance score and confidence
// score using fixed production thresholds:
//
//	IS >= 7 AND CS >= 0.8  -> "notified"
//	IS >= 7 AND CS <  0.8  -> "buffered"
//	otherwise              -> "ignored"
func DeriveBand(is, cs float64) string {
	if is >= importanceThreshold && cs >= confidenceThreshold {
		return "notified"
	}
	if is >= importanceThreshold {
		return "buffered"
	}
	return "ignored"
}
