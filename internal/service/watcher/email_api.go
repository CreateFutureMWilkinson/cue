package watcher

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// IMAPClient is a real IMAP client that connects to an IMAP server.
type IMAPClient struct {
	host       string
	port       int
	username   string
	password   string
	encryption string
}

// NewIMAPClient creates a new IMAPClient with the given credentials.
func NewIMAPClient(host string, port int, username, password, encryption string) (*IMAPClient, error) {
	if host == "" {
		return nil, fmt.Errorf("host must not be empty")
	}
	if port <= 0 {
		return nil, fmt.Errorf("port must be greater than zero")
	}
	if username == "" {
		return nil, fmt.Errorf("username must not be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password must not be empty")
	}
	return &IMAPClient{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		encryption: encryption,
	}, nil
}

// Encryption returns the configured encryption mode.
func (c *IMAPClient) Encryption() string {
	return c.encryption
}

// FetchNewMessages connects to the IMAP server, authenticates, searches for
// messages with UID > lastUID and returns them as EmailMessage values.
func (c *IMAPClient) FetchNewMessages(ctx context.Context, lastUID uint32) ([]EmailMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var imapClient *imapclient.Client
	switch c.encryption {
	case "ssl_tls":
		tlsConf := &tls.Config{ServerName: c.host}
		dialer := &net.Dialer{}
		tlsConn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, tlsConf)
		if dialErr != nil {
			return nil, fmt.Errorf("TLS connecting to IMAP server %s: %w", addr, dialErr)
		}
		imapClient = imapclient.New(tlsConn, nil)
		if err := imapClient.WaitGreeting(); err != nil {
			imapClient.Close() // #nosec G104 -- best-effort cleanup; the greeting error is the actionable one
			return nil, fmt.Errorf("waiting for IMAP greeting: %w", err)
		}
	case "starttls":
		dialer := &net.Dialer{}
		plainConn, dialErr := dialer.DialContext(ctx, "tcp", addr)
		if dialErr != nil {
			return nil, fmt.Errorf("connecting to IMAP server %s: %w", addr, dialErr)
		}
		opts := &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: c.host},
		}
		var startTLSErr error
		imapClient, startTLSErr = imapclient.NewStartTLS(plainConn, opts)
		if startTLSErr != nil {
			return nil, fmt.Errorf("STARTTLS upgrade: %w", startTLSErr)
		}
	default: // "none" or unrecognized
		dialer := &net.Dialer{}
		plainConn, dialErr := dialer.DialContext(ctx, "tcp", addr)
		if dialErr != nil {
			return nil, fmt.Errorf("connecting to IMAP server %s: %w", addr, dialErr)
		}
		imapClient = imapclient.New(plainConn, nil)
		if err := imapClient.WaitGreeting(); err != nil {
			imapClient.Close() // #nosec G104 -- best-effort cleanup; the greeting error is the actionable one
			return nil, fmt.Errorf("waiting for IMAP greeting: %w", err)
		}
	}
	defer imapClient.Close()

	if err := imapClient.Login(c.username, c.password).Wait(); err != nil {
		return nil, fmt.Errorf("IMAP login: %w", err)
	}

	if _, err := imapClient.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("selecting INBOX: %w", err)
	}

	// Build UID search criteria for UIDs > lastUID.
	var uidSet imap.UIDSet
	uidSet.AddRange(imap.UID(lastUID+1), 0) // 0 means "*"

	criteria := &imap.SearchCriteria{
		UID: []imap.UIDSet{uidSet},
	}

	searchData, err := imapClient.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("IMAP UID SEARCH: %w", err)
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return nil, nil
	}

	var fetchSet imap.UIDSet
	for _, uid := range uids {
		fetchSet.AddNum(uid)
	}

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText, Peek: true},
		},
	}

	fetchCmd := imapClient.Fetch(fetchSet, fetchOptions)

	var messages []EmailMessage
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			fetchCmd.Close() // #nosec G104 -- best-effort cleanup; the collect error on the next line is the actionable one
			return nil, fmt.Errorf("collecting FETCH data: %w", err)
		}

		messages = append(messages, emailMessageFromBuffer(buf))
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("closing FETCH command: %w", err)
	}

	return messages, nil
}

// emailMessageFromBuffer converts a FetchMessageBuffer into an EmailMessage.
func emailMessageFromBuffer(buf *imapclient.FetchMessageBuffer) EmailMessage {
	email := EmailMessage{
		UID:    uint32(buf.UID),
		Folder: "INBOX",
	}

	if buf.Envelope != nil {
		env := buf.Envelope
		email.MessageID = env.MessageID
		email.Subject = env.Subject

		if len(env.From) > 0 {
			email.From = env.From[0].Addr()
		}
		for _, addr := range env.To {
			email.To = append(email.To, addr.Addr())
		}
		for _, addr := range env.Cc {
			email.CC = append(email.CC, addr.Addr())
		}
		for _, addr := range env.Bcc {
			email.BCC = append(email.BCC, addr.Addr())
		}
	}

	for _, section := range buf.BodySection {
		email.Body = strings.TrimSpace(string(section.Bytes))
		break
	}

	return email
}

// ensure IMAPClient satisfies EmailAPI at compile time.
var _ EmailAPI = (*IMAPClient)(nil)
