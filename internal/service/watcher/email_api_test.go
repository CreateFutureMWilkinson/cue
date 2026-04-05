package watcher_test

import (
	"context"
	"net"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/stretchr/testify/suite"
)

type IMAPClientSuite struct {
	suite.Suite
}

func TestIMAPClient(t *testing.T) {
	suite.Run(t, new(IMAPClientSuite))
}

// --- Constructor validation tests ---

func (s *IMAPClientSuite) TestNewIMAPClient_EmptyHost() {
	s.T().Setenv("TEST_EMAIL_PW", "secret")

	client, err := watcher.NewIMAPClient("", 993, "user@example.com", "TEST_EMAIL_PW")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "host")
}

func (s *IMAPClientSuite) TestNewIMAPClient_ZeroPort() {
	s.T().Setenv("TEST_EMAIL_PW", "secret")

	client, err := watcher.NewIMAPClient("imap.example.com", 0, "user@example.com", "TEST_EMAIL_PW")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "port")
}

func (s *IMAPClientSuite) TestNewIMAPClient_EmptyUsername() {
	s.T().Setenv("TEST_EMAIL_PW", "secret")

	client, err := watcher.NewIMAPClient("imap.example.com", 993, "", "TEST_EMAIL_PW")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "username")
}

func (s *IMAPClientSuite) TestNewIMAPClient_PasswordEnvNotSet() {
	// UNSET_VAR_46 is never set in the environment
	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "UNSET_VAR_46")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "password")
}

func (s *IMAPClientSuite) TestNewIMAPClient_PasswordEnvEmpty() {
	s.T().Setenv("TEST_EMAIL_PW_EMPTY", "")

	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "TEST_EMAIL_PW_EMPTY")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "password")
}

func (s *IMAPClientSuite) TestNewIMAPClient_Valid() {
	s.T().Setenv("TEST_EMAIL_PW", "secret")

	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "TEST_EMAIL_PW")

	s.NoError(err)
	s.NotNil(client)
}

// --- FetchNewMessages error tests ---

func (s *IMAPClientSuite) TestFetchNewMessages_ConnectionRefused() {
	s.T().Setenv("TEST_EMAIL_PW", "secret")

	// Find an unused local port to guarantee a connection-refused response
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // close immediately so the port is unused

	client, err := watcher.NewIMAPClient("127.0.0.1", port, "user@example.com", "TEST_EMAIL_PW")
	s.Require().NoError(err)

	_, fetchErr := client.FetchNewMessages(context.Background(), 0)
	s.Error(fetchErr)
}

func (s *IMAPClientSuite) TestFetchNewMessages_ContextCancelled() {
	s.T().Setenv("TEST_EMAIL_PW", "secret")

	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "TEST_EMAIL_PW")
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling

	_, fetchErr := client.FetchNewMessages(ctx, 0)
	s.Error(fetchErr)
}
