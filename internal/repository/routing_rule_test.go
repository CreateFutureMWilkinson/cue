package repository_test

import (
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type RoutingRuleSuite struct {
	suite.Suite
}

func TestRoutingRule(t *testing.T) {
	suite.Run(t, new(RoutingRuleSuite))
}

// validRule returns a valid routing rule for testing purposes.
func (s *RoutingRuleSuite) validRule() *repository.RoutingRule {
	return &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "^general$",
		Action:         "notified",
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func (s *RoutingRuleSuite) TestValidateValidRule() {
	r := s.validRule()
	err := r.Validate()
	s.NoError(err)
}

func (s *RoutingRuleSuite) TestValidateInvalidSourceType() {
	r := s.validRule()
	r.SourceType = "ftp"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateInvalidAction() {
	r := s.validRule()
	r.Action = "buffered"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateNegativePriority() {
	r := s.validRule()
	r.Priority = -1
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateInvalidChannelPatternRegex() {
	r := s.validRule()
	r.ChannelPattern = "[invalid"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateInvalidContentPatternRegex() {
	r := s.validRule()
	r.ContentPattern = "[invalid"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateValidSourceTypes() {
	for _, st := range []string{"slack", "email"} {
		r := s.validRule()
		r.SourceType = st
		err := r.Validate()
		s.NoError(err, "source type %q should be valid", st)
	}
}

func (s *RoutingRuleSuite) TestValidateEmptyPatternsAreValid() {
	r := s.validRule()
	r.ChannelPattern = ""
	r.ContentPattern = ""
	err := r.Validate()
	s.NoError(err, "empty patterns should be valid")
}
