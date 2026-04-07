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
		ID:        uuid.New(),
		Priority:  0,
		Source:    "slack",
		Field:     "channel",
		Pattern:   "^general$",
		Action:    "notified",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (s *RoutingRuleSuite) TestValidateValidRule() {
	r := s.validRule()
	err := r.Validate()
	s.NoError(err)
}

func (s *RoutingRuleSuite) TestValidateInvalidSource() {
	r := s.validRule()
	r.Source = "ftp"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateInvalidFieldForEmail() {
	r := s.validRule()
	r.Source = "email"
	r.Field = "channel"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateInvalidFieldForSlack() {
	r := s.validRule()
	r.Source = "slack"
	r.Field = "subject"
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

func (s *RoutingRuleSuite) TestValidateInvalidRegex() {
	r := s.validRule()
	r.Pattern = "[invalid"
	err := r.Validate()
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSuite) TestValidateValidEmailFields() {
	for _, field := range []string{"sender", "subject"} {
		r := s.validRule()
		r.Source = "email"
		r.Field = field
		err := r.Validate()
		s.NoError(err, "field %q should be valid for email", field)
	}
}

func (s *RoutingRuleSuite) TestValidateValidSlackFields() {
	for _, field := range []string{"sender", "channel", "content", "message_type"} {
		r := s.validRule()
		r.Source = "slack"
		r.Field = field
		err := r.Validate()
		s.NoError(err, "field %q should be valid for slack", field)
	}
}
