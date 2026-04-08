package cuebench

import (
	"errors"
	"math"
	"slices"
)

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

// percentile calculates the specified percentile from sorted latency data.
// p should be between 0.0 and 1.0 (e.g., 0.5 for median, 0.95 for 95th percentile).
func percentile(sortedLatencies []int64, p float64) int64 {
	if len(sortedLatencies) == 0 {
		return 0
	}

	if p == 0.5 {
		// Median: use standard definition
		return sortedLatencies[len(sortedLatencies)/2]
	}

	// For other percentiles: use ceiling approach
	idx := int(math.Ceil(float64(len(sortedLatencies))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sortedLatencies) {
		idx = len(sortedLatencies) - 1
	}
	return sortedLatencies[idx]
}

// CalcMetrics computes aggregate statistics from a slice of RunResult.
func CalcMetrics(results []RunResult) AggregateMetrics {
	totalResults := len(results)
	if totalResults == 0 {
		return AggregateMetrics{}
	}

	var correctBandCount int
	var validJSONCount int
	var falsePositives, ignoredMessages int
	var falseNegatives, notifiedMessages int
	inferenceLatencies := make([]int64, 0, totalResults)

	for _, result := range results {
		if result.BandCorrect {
			correctBandCount++
		}
		if result.JSONValid {
			validJSONCount++
		}

		// Count false positives: ignored messages incorrectly notified
		if result.ExpectedBand == "ignored" {
			ignoredMessages++
			if result.Band == "notified" {
				falsePositives++
			}
		}

		// Count false negatives: notified messages incorrectly ignored
		if result.ExpectedBand == "notified" {
			notifiedMessages++
			if result.Band == "ignored" {
				falseNegatives++
			}
		}

		inferenceLatencies = append(inferenceLatencies, result.InferenceMs)
	}

	slices.Sort(inferenceLatencies)

	var falsePositiveRate float64
	if ignoredMessages > 0 {
		falsePositiveRate = float64(falsePositives) / float64(ignoredMessages) * 100
	}

	var falseNegativeRate float64
	if notifiedMessages > 0 {
		falseNegativeRate = float64(falseNegatives) / float64(notifiedMessages) * 100
	}

	return AggregateMetrics{
		BandAccuracy:      float64(correctBandCount) / float64(totalResults) * 100,
		FalsePositiveRate: falsePositiveRate,
		FalseNegativeRate: falseNegativeRate,
		JSONCompliance:    float64(validJSONCount) / float64(totalResults) * 100,
		P50Ms:             percentile(inferenceLatencies, 0.5),
		P95Ms:             percentile(inferenceLatencies, 0.95),
	}
}

// CalibrationLift returns the difference in BandAccuracy between a full
// (few-shot) run and a base (zero-shot) run. Positive values indicate
// that calibration examples improve routing accuracy.
func CalibrationLift(base, full AggregateMetrics) float64 {
	return 0
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
