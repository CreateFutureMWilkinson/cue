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
	BandCorrect  bool   // Band == entry.ExpectedBand
}

// DeriveBand derives the routing band from importance score and confidence
// score using fixed production thresholds:
//
//	IS >= 7 AND CS >= 0.8  -> "notified"
//	IS >= 7 AND CS <  0.8  -> "buffered"
//	otherwise              -> "ignored"
func DeriveBand(is, cs float64) string {
	return "" // stub
}
