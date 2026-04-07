package watcher_test

import (
	"context"
	"net"
	"strings"
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
	client, err := watcher.NewIMAPClient("", 993, "user@example.com", "secret", "ssl_tls")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "host")
}

func (s *IMAPClientSuite) TestNewIMAPClient_ZeroPort() {
	client, err := watcher.NewIMAPClient("imap.example.com", 0, "user@example.com", "secret", "ssl_tls")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "port")
}

func (s *IMAPClientSuite) TestNewIMAPClient_EmptyUsername() {
	client, err := watcher.NewIMAPClient("imap.example.com", 993, "", "secret", "ssl_tls")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "username")
}

func (s *IMAPClientSuite) TestNewIMAPClient_EmptyPassword() {
	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "", "ssl_tls")

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "password")
}

func (s *IMAPClientSuite) TestNewIMAPClient_Valid() {
	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "secret", "ssl_tls")

	s.NoError(err)
	s.NotNil(client)
}

func (s *IMAPClientSuite) TestNewIMAPClient_StoresEncryption() {
	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "secret", "starttls")
	s.NoError(err)
	s.NotNil(client)
	s.Equal("starttls", client.Encryption())
}

// --- FetchNewMessages error tests ---

func (s *IMAPClientSuite) TestFetchNewMessages_ConnectionRefused() {
	// Find an unused local port to guarantee a connection-refused response
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // close immediately so the port is unused

	client, err := watcher.NewIMAPClient("127.0.0.1", port, "user@example.com", "secret", "ssl_tls")
	s.Require().NoError(err)

	_, fetchErr := client.FetchNewMessages(context.Background(), 0)
	s.Error(fetchErr)
}

func (s *IMAPClientSuite) TestFetchNewMessages_SSLTLSAttemptsTLS() {
	// Start a plain TCP listener that accepts and immediately closes
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	client, err := watcher.NewIMAPClient("127.0.0.1", port, "user@example.com", "secret", "ssl_tls")
	s.Require().NoError(err)

	_, fetchErr := client.FetchNewMessages(context.Background(), 0)
	s.Error(fetchErr)
	// When ssl_tls is used, the error should be TLS-related, not a plain TCP error
	s.Contains(strings.ToLower(fetchErr.Error()), "tls",
		"ssl_tls mode should attempt a TLS connection")
}

func (s *IMAPClientSuite) TestFetchNewMessages_ContextCancelled() {
	client, err := watcher.NewIMAPClient("imap.example.com", 993, "user@example.com", "secret", "ssl_tls")
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling

	_, fetchErr := client.FetchNewMessages(ctx, 0)
	s.Error(fetchErr)
}
