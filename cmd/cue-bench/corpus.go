package cuebench

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed corpus.json
var defaultCorpusJSON []byte

// CorpusEntry represents a single benchmark corpus message with expected
// routing band, optional tags, and an optional user rating for calibration.
type CorpusEntry struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Sender       string   `json:"sender"`
	Channel      string   `json:"channel"`
	Content      string   `json:"content"`
	ExpectedBand string   `json:"expected_band"`
	Tags         []string `json:"tags"`
	UserRating   *int     `json:"user_rating"`
}

// LoadCorpus reads a JSON corpus file from the given path and returns the
// parsed entries. If path is empty, the embedded corpus.json is used.
func LoadCorpus(path string) ([]CorpusEntry, error) {
	var data []byte
	if path == "" {
		data = defaultCorpusJSON
	} else {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read corpus file: %w", err)
		}
	}

	var entries []CorpusEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse corpus JSON: %w", err)
	}
	return entries, nil
}

// ScoredEntries returns the subset of entries where UserRating is nil
// (scored by the system but not yet rated by a human).
func ScoredEntries(entries []CorpusEntry) []CorpusEntry {
	result := make([]CorpusEntry, 0, len(entries))
	for _, e := range entries {
		if e.UserRating == nil {
			result = append(result, e)
		}
	}
	return result
}

// RatedEntries returns the subset of entries where UserRating is non-nil
// (a human has provided a rating).
func RatedEntries(entries []CorpusEntry) []CorpusEntry {
	result := make([]CorpusEntry, 0, len(entries))
	for _, e := range entries {
		if e.UserRating != nil {
			result = append(result, e)
		}
	}
	return result
}
