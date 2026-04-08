package cuebench

import (
	"errors"
	"math"
	"sort"
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

// CalcMetrics computes aggregate statistics from a slice of RunResult.
func CalcMetrics(results []RunResult) AggregateMetrics {
	n := len(results)
	if n == 0 {
		return AggregateMetrics{}
	}

	var bandCorrectCount int
	var jsonValidCount int
	var fpNumerator, fpDenominator int
	var fnNumerator, fnDenominator int
	inferenceMs := make([]int64, 0, n)

	for _, r := range results {
		if r.BandCorrect {
			bandCorrectCount++
		}
		if r.JSONValid {
			jsonValidCount++
		}
		if r.ExpectedBand == "ignored" {
			fpDenominator++
			if r.Band == "notified" {
				fpNumerator++
			}
		}
		if r.ExpectedBand == "notified" {
			fnDenominator++
			if r.Band == "ignored" {
				fnNumerator++
			}
		}
		inferenceMs = append(inferenceMs, r.InferenceMs)
	}

	sort.Slice(inferenceMs, func(i, j int) bool {
		return inferenceMs[i] < inferenceMs[j]
	})

	var fpr float64
	if fpDenominator > 0 {
		fpr = float64(fpNumerator) / float64(fpDenominator) * 100
	}

	var fnr float64
	if fnDenominator > 0 {
		fnr = float64(fnNumerator) / float64(fnDenominator) * 100
	}

	p50 := inferenceMs[n/2]
	p95Idx := int(math.Ceil(float64(n)*0.95)) - 1
	p95 := inferenceMs[p95Idx]

	return AggregateMetrics{
		BandAccuracy:      float64(bandCorrectCount) / float64(n) * 100,
		FalsePositiveRate: fpr,
		FalseNegativeRate: fnr,
		JSONCompliance:    float64(jsonValidCount) / float64(n) * 100,
		P50Ms:             p50,
		P95Ms:             p95,
	}
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
