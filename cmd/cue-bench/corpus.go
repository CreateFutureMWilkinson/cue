package cuebench

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
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

// candidateWithScore holds a corpus entry with its computed similarity scores
// for selection ranking.
type candidateWithScore struct {
	entry       CorpusEntry
	tagScore    int // number of overlapping tags
	sourceScore int // 1 if same source, 0 otherwise
}

// SelectExamples selects up to n rated entries from pool to serve as few-shot
// examples for entry. Selection prefers tag overlap, then source match, then
// seeded random tiebreak. Only entries with non-nil UserRating are eligible.
func SelectExamples(entry CorpusEntry, pool []CorpusEntry, n int, seed int64) []CorpusEntry {
	// Filter to rated-only candidates.
	candidates := make([]CorpusEntry, 0, len(pool))
	for _, candidate := range pool {
		if candidate.UserRating != nil {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Build a tag set for the target entry for O(1) lookups.
	targetTags := make(map[string]struct{}, len(entry.Tags))
	for _, tag := range entry.Tags {
		targetTags[tag] = struct{}{}
	}

	// Score each candidate.
	scoredCandidates := make([]candidateWithScore, len(candidates))
	for i, candidate := range candidates {
		tagOverlaps := 0
		for _, tag := range candidate.Tags {
			if _, exists := targetTags[tag]; exists {
				tagOverlaps++
			}
		}

		sourceMatch := 0
		if candidate.Source == entry.Source {
			sourceMatch = 1
		}

		scoredCandidates[i] = candidateWithScore{
			entry:       candidate,
			tagScore:    tagOverlaps,
			sourceScore: sourceMatch,
		}
	}

	// Seeded shuffle for tiebreaking: shuffle first, then stable-sort so that
	// equal (tagScore, sourceScore) groups retain the shuffled order.
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(scoredCandidates), func(i, j int) {
		scoredCandidates[i], scoredCandidates[j] = scoredCandidates[j], scoredCandidates[i]
	})

	sort.SliceStable(scoredCandidates, func(i, j int) bool {
		if scoredCandidates[i].tagScore != scoredCandidates[j].tagScore {
			return scoredCandidates[i].tagScore > scoredCandidates[j].tagScore
		}
		return scoredCandidates[i].sourceScore > scoredCandidates[j].sourceScore
	})

	// Cap at n.
	if n > len(scoredCandidates) {
		n = len(scoredCandidates)
	}

	result := make([]CorpusEntry, n)
	for i := 0; i < n; i++ {
		result[i] = scoredCandidates[i].entry
	}
	return result
}
