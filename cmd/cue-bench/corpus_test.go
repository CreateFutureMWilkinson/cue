package cuebench_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cuebench "github.com/CreateFutureMWilkinson/cue/cmd/cue-bench"
	"github.com/stretchr/testify/suite"
)

type CorpusSuite struct {
	suite.Suite
}

func TestCorpus(t *testing.T) { suite.Run(t, new(CorpusSuite)) }

func (s *CorpusSuite) TestLoadCorpusFromValidFile() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "test_corpus.json")

	rating := 8
	entries := []cuebench.CorpusEntry{
		{
			ID:           "test-01",
			Source:       "slack",
			Sender:       "alice",
			Channel:      "#general",
			Content:      "Hello world",
			ExpectedBand: "ignored",
			Tags:         []string{"greeting"},
			UserRating:   nil,
		},
		{
			ID:           "test-02",
			Source:       "email",
			Sender:       "bob@example.com",
			Channel:      "inbox",
			Content:      "Server is down",
			ExpectedBand: "notified",
			Tags:         []string{"outage"},
			UserRating:   &rating,
		},
	}

	data, err := json.Marshal(entries)
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(path, data, 0644))

	result, err := cuebench.LoadCorpus(path)
	s.Require().NoError(err)
	s.Require().Len(result, 2)
	s.Equal("test-01", result[0].ID)
	s.Equal("test-02", result[1].ID)
	s.Nil(result[0].UserRating)
	s.Require().NotNil(result[1].UserRating)
	s.Equal(8, *result[1].UserRating)
}

func (s *CorpusSuite) TestLoadCorpusFromEmbeddedDefault() {
	result, err := cuebench.LoadCorpus("")
	s.Require().NoError(err)
	s.Greater(len(result), 0, "embedded corpus should contain entries")

	for i, entry := range result {
		s.NotEmptyf(entry.ID, "entry %d should have a non-empty ID", i)
	}
}

func (s *CorpusSuite) TestLoadCorpusMalformedJSON() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "bad.json")
	s.Require().NoError(os.WriteFile(path, []byte(`[{"id": broken`), 0644))

	result, err := cuebench.LoadCorpus(path)
	s.Error(err)
	s.Nil(result)
}

func (s *CorpusSuite) TestScoredAndRatedEntriesSplit() {
	r5 := 5
	r9 := 9
	entries := []cuebench.CorpusEntry{
		{ID: "unrated-1", UserRating: nil},
		{ID: "rated-1", UserRating: &r5},
		{ID: "unrated-2", UserRating: nil},
		{ID: "rated-2", UserRating: &r9},
		{ID: "unrated-3", UserRating: nil},
	}

	scored := cuebench.ScoredEntries(entries)
	rated := cuebench.RatedEntries(entries)

	s.Require().Len(scored, 3, "ScoredEntries should return entries with nil UserRating")
	s.Require().Len(rated, 2, "RatedEntries should return entries with non-nil UserRating")

	s.Equal("unrated-1", scored[0].ID)
	s.Equal("unrated-2", scored[1].ID)
	s.Equal("unrated-3", scored[2].ID)

	s.Equal("rated-1", rated[0].ID)
	s.Equal("rated-2", rated[1].ID)
}

// ---------- ExampleSelectionSuite ----------

type ExampleSelectionSuite struct {
	suite.Suite
}

func TestExampleSelection(t *testing.T) { suite.Run(t, new(ExampleSelectionSuite)) }

// intPtr is a helper that returns a pointer to the given int.
func intPtr(v int) *int { return &v }

func (s *ExampleSelectionSuite) TestSelectExamples_PrefersByTagOverlap() {
	entry := cuebench.CorpusEntry{
		ID:     "target",
		Source: "slack",
		Tags:   []string{"outage", "critical"},
	}

	pool := []cuebench.CorpusEntry{
		{
			ID:         "overlap-0",
			Source:     "slack",
			Tags:       []string{"greeting"},
			UserRating: intPtr(5),
		},
		{
			ID:         "overlap-2",
			Source:     "slack",
			Tags:       []string{"outage", "critical"},
			UserRating: intPtr(7),
		},
		{
			ID:         "overlap-1",
			Source:     "slack",
			Tags:       []string{"outage"},
			UserRating: intPtr(6),
		},
	}

	result := cuebench.SelectExamples(entry, pool, 3, 42)
	s.Require().Len(result, 3, "should return all 3 rated entries")
	s.Equal("overlap-2", result[0].ID, "entry with 2-tag overlap should be first")
	s.Equal("overlap-1", result[1].ID, "entry with 1-tag overlap should be second")
	s.Equal("overlap-0", result[2].ID, "entry with 0-tag overlap should be last")
}

func (s *ExampleSelectionSuite) TestSelectExamples_PrefersSourceMatch() {
	entry := cuebench.CorpusEntry{
		ID:     "target",
		Source: "slack",
		Tags:   []string{"outage"},
	}

	pool := []cuebench.CorpusEntry{
		{
			ID:         "email-entry",
			Source:     "email",
			Tags:       []string{"outage"},
			UserRating: intPtr(5),
		},
		{
			ID:         "slack-entry",
			Source:     "slack",
			Tags:       []string{"outage"},
			UserRating: intPtr(5),
		},
	}

	result := cuebench.SelectExamples(entry, pool, 2, 42)
	s.Require().Len(result, 2, "should return both rated entries")
	s.Equal("slack-entry", result[0].ID, "same-source entry should rank first when tag overlap is equal")
}

func (s *ExampleSelectionSuite) TestSelectExamples_ReproducibleWithSeed() {
	entry := cuebench.CorpusEntry{
		ID:     "target",
		Source: "slack",
		Tags:   []string{"misc"},
	}

	// Build a pool of 10 entries all with identical tag overlap and source so
	// that ordering depends entirely on the seeded random tiebreak.
	pool := make([]cuebench.CorpusEntry, 10)
	for i := range pool {
		pool[i] = cuebench.CorpusEntry{
			ID:         fmt.Sprintf("entry-%02d", i),
			Source:     "slack",
			Tags:       []string{"misc"},
			UserRating: intPtr(5),
		}
	}

	resultA1 := cuebench.SelectExamples(entry, pool, 10, 99)
	resultA2 := cuebench.SelectExamples(entry, pool, 10, 99)
	s.Require().Len(resultA1, 10)
	s.Require().Len(resultA2, 10)

	// Same seed must produce identical order.
	for i := range resultA1 {
		s.Equal(resultA1[i].ID, resultA2[i].ID, "same seed should yield identical order at index %d", i)
	}

	// Different seed: collect IDs and verify at least one positional difference.
	resultB := cuebench.SelectExamples(entry, pool, 10, 7777)
	s.Require().Len(resultB, 10)

	differs := false
	for i := range resultA1 {
		if resultA1[i].ID != resultB[i].ID {
			differs = true
			break
		}
	}
	s.True(differs, "different seeds should (with high probability) produce different orderings")
}

func (s *ExampleSelectionSuite) TestSelectExamples_IgnoresUnratedEntries() {
	entry := cuebench.CorpusEntry{
		ID:     "target",
		Source: "slack",
		Tags:   []string{"outage"},
	}

	pool := []cuebench.CorpusEntry{
		{
			ID:         "rated-1",
			Source:     "slack",
			Tags:       []string{"outage"},
			UserRating: intPtr(7),
		},
		{
			ID:         "unrated-1",
			Source:     "slack",
			Tags:       []string{"outage"},
			UserRating: nil,
		},
		{
			ID:         "unrated-2",
			Source:     "email",
			Tags:       []string{"critical"},
			UserRating: nil,
		},
		{
			ID:         "rated-2",
			Source:     "email",
			Tags:       []string{"outage"},
			UserRating: intPtr(3),
		},
	}

	result := cuebench.SelectExamples(entry, pool, 10, 42)
	s.Require().Len(result, 2, "only rated entries should be returned")

	for _, r := range result {
		s.NotNil(r.UserRating, "every returned entry must have a non-nil UserRating")
		s.NotEqual("unrated-1", r.ID)
		s.NotEqual("unrated-2", r.ID)
	}
}

func (s *ExampleSelectionSuite) TestSelectExamples_RespectsMaxN() {
	entry := cuebench.CorpusEntry{
		ID:     "target",
		Source: "slack",
		Tags:   []string{"outage"},
	}

	pool := make([]cuebench.CorpusEntry, 10)
	for i := range pool {
		pool[i] = cuebench.CorpusEntry{
			ID:         fmt.Sprintf("pool-%02d", i),
			Source:     "slack",
			Tags:       []string{"outage"},
			UserRating: intPtr(5),
		}
	}

	result := cuebench.SelectExamples(entry, pool, 3, 42)
	s.Require().Len(result, 3, "should return exactly n entries when pool has more than n rated entries")
}
