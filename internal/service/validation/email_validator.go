package validation

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
)

// IMAPValidator validates IMAP credentials by connecting, logging in, and
// immediately logging out. It is stateless and safe for concurrent use.
type IMAPValidator struct{}

// NewIMAPValidator creates a new IMAPValidator.
func NewIMAPValidator() *IMAPValidator {
	return &IMAPValidator{}
}

// ValidateEmail connects to the given IMAP server, authenticates with the
// supplied credentials, and logs out. Any failure is returned as a
// human-readable error.
func (v *IMAPValidator) ValidateEmail(ctx context.Context, host string, port int, username, password, encryption string) error {
	if host == "" {
		return fmt.Errorf("IMAP validation: host must not be empty")
	}
	if username == "" {
		return fmt.Errorf("IMAP validation: username must not be empty")
	}
	if password == "" {
		return fmt.Errorf("IMAP validation: password must not be empty")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var imapClient *imapclient.Client

	switch encryption {
	case "ssl_tls":
		tlsConf := &tls.Config{ServerName: host}
		tlsConn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConf)
		if err != nil {
			return fmt.Errorf("IMAP TLS connection to %s: %w", addr, err)
		}
		imapClient = imapclient.New(tlsConn, nil)
		if err := imapClient.WaitGreeting(); err != nil {
			imapClient.Close() // #nosec G104 -- best-effort cleanup
			return fmt.Errorf("IMAP greeting from %s: %w", addr, err)
		}

	case "starttls":
		plainConn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("IMAP connection to %s: %w", addr, err)
		}
		opts := &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: host},
		}
		var startTLSErr error
		imapClient, startTLSErr = imapclient.NewStartTLS(plainConn, opts)
		if startTLSErr != nil {
			return fmt.Errorf("IMAP STARTTLS upgrade on %s: %w", addr, startTLSErr)
		}

	default: // "none" or unrecognised
		plainConn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("IMAP connection to %s: %w", addr, err)
		}
		imapClient = imapclient.New(plainConn, nil)
		if err := imapClient.WaitGreeting(); err != nil {
			imapClient.Close() // #nosec G104 -- best-effort cleanup
			return fmt.Errorf("IMAP greeting from %s: %w", addr, err)
		}
	}
	defer imapClient.Close()

	if err := imapClient.Login(username, password).Wait(); err != nil {
		return fmt.Errorf("IMAP login: %w", err)
	}

	if err := imapClient.Logout().Wait(); err != nil {
		return fmt.Errorf("IMAP logout: %w", err)
	}

	return nil
}
