package cuebench_test

import (
	"encoding/json"
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
