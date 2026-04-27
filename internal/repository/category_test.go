package repository_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// CategoryNormalizationSuite covers the pure helpers
// NormalizeCategoryKey and PresentCategoryName defined by
// Feature 109 Decision 3.
type CategoryNormalizationSuite struct {
	suite.Suite
}

func TestCategoryNormalization(t *testing.T) {
	suite.Run(t, new(CategoryNormalizationSuite))
}

func (s *CategoryNormalizationSuite) TestNormalizeCategoryKey() {
	cases := []struct {
		name        string
		input       string
		want        string
		wantErr     bool
		errContains string
	}{
		{name: "single word lowercased", input: "FOOBAR", want: "foobar"},
		{name: "single space becomes underscore", input: "foo bar", want: "foo_bar"},
		{name: "mixed case with space", input: "foo BAR", want: "foo_bar"},
		{name: "trim and collapse whitespace", input: "  Foo   Bar  ", want: "foo_bar"},
		{name: "underscore rejected", input: "foo_bar", wantErr: true, errContains: "underscores not allowed"},
		{name: "non-alnum punctuation rejected", input: "foo!", wantErr: true},
		{name: "empty rejected", input: "", wantErr: true},
		{name: "whitespace-only rejected", input: "   ", wantErr: true},
		{name: "65 chars rejected", input: strings.Repeat("a", 65), wantErr: true},
		{name: "64 chars accepted", input: strings.Repeat("a", 64), want: strings.Repeat("a", 64)},
		{name: "digits allowed", input: "team 42", want: "team_42"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got, err := repository.NormalizeCategoryKey(tc.input)
			if tc.wantErr {
				s.Require().Error(err, "expected error for input %q", tc.input)
				if tc.errContains != "" {
					s.Contains(err.Error(), tc.errContains)
				}
				return
			}
			s.Require().NoError(err, "unexpected error for input %q", tc.input)
			s.Equal(tc.want, got)
		})
	}
}

func (s *CategoryNormalizationSuite) TestPresentCategoryName() {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "single word", key: "foobar", want: "Foobar"},
		{name: "two words", key: "foo_bar", want: "Foo Bar"},
		{name: "acronym mechanically titled", key: "api_docs", want: "Api Docs"},
		{name: "empty is no-op", key: "", want: ""},
		{name: "three words", key: "alpha_beta_gamma", want: "Alpha Beta Gamma"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := repository.PresentCategoryName(tc.key)
			s.Equal(tc.want, got)
		})
	}
}
