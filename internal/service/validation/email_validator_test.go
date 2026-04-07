package validation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/service/validation"
)

type IMAPValidatorSuite struct {
	suite.Suite
	validator *validation.IMAPValidator
}

func TestIMAPValidator(t *testing.T) {
	suite.Run(t, new(IMAPValidatorSuite))
}

func (s *IMAPValidatorSuite) SetupTest() {
	s.validator = validation.NewIMAPValidator()
}

func (s *IMAPValidatorSuite) TestUnreachableHost() {
	// Port 1 on loopback should be unreachable / refused.
	err := s.validator.ValidateEmail(context.Background(), "127.0.0.1", 1, "user", "pass", "none")
	s.Error(err)
	s.Contains(err.Error(), "IMAP connection")
}

func (s *IMAPValidatorSuite) TestUnreachableHost_SSL() {
	err := s.validator.ValidateEmail(context.Background(), "127.0.0.1", 1, "user", "pass", "ssl_tls")
	s.Error(err)
	s.Contains(err.Error(), "IMAP TLS connection")
}

func (s *IMAPValidatorSuite) TestEmptyHost() {
	err := s.validator.ValidateEmail(context.Background(), "", 993, "user", "pass", "ssl_tls")
	s.Error(err)
	s.Contains(err.Error(), "host must not be empty")
}

func (s *IMAPValidatorSuite) TestEmptyUsername() {
	err := s.validator.ValidateEmail(context.Background(), "imap.example.com", 993, "", "pass", "ssl_tls")
	s.Error(err)
	s.Contains(err.Error(), "username must not be empty")
}

func (s *IMAPValidatorSuite) TestEmptyPassword() {
	err := s.validator.ValidateEmail(context.Background(), "imap.example.com", 993, "user", "", "ssl_tls")
	s.Error(err)
	s.Contains(err.Error(), "password must not be empty")
}
