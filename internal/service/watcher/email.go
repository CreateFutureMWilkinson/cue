package watcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

const (
	SourceEmail        = "email"
	MessageTypeMention = "mention"
)

// EmailMessage represents an email fetched from IMAP.
type EmailMessage struct {
	UID       uint32
	MessageID string
	From      string
	Subject   string
	Folder    string
	Body      string
	To        []string
	CC        []string
	BCC       []string
}

// EmailAPI defines the interface for interacting with an IMAP email server.
type EmailAPI interface {
	FetchNewMessages(ctx context.Context, lastUID uint32) ([]EmailMessage, error)
}

// EmailWatcher polls an email account for new messages and converts them to repository messages.
type EmailWatcher struct {
	api      EmailAPI
	username string
	lastUID  uint32
}

// NewEmailWatcher creates a new EmailWatcher with the given API client and configuration.
func NewEmailWatcher(api EmailAPI, cfg config.EmailConfig) (*EmailWatcher, error) {
	if api == nil {
		return nil, fmt.Errorf("api must not be nil")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("username must not be empty")
	}
	return &EmailWatcher{
		api:      api,
		username: cfg.Username,
	}, nil
}

// Poll fetches new email messages and returns them as repository messages.
func (w *EmailWatcher) Poll(ctx context.Context) ([]*repository.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	emails, err := w.api.FetchNewMessages(ctx, w.lastUID)
	if err != nil {
		return nil, fmt.Errorf("fetching new messages: %w", err)
	}

	var result []*repository.Message
	for _, email := range emails {
		msg := w.convertEmailMessage(email)
		result = append(result, msg)

		w.updateLastUID(email.UID)
	}

	return result, nil
}

func (w *EmailWatcher) convertEmailMessage(email EmailMessage) *repository.Message {
	msgType := MessageTypeMsg
	if w.isMentioned(email) {
		msgType = MessageTypeMention
	}

	content := email.Subject + "\n" + email.Body

	return &repository.Message{
		ID:            uuid.New(),
		Source:        SourceEmail,
		SourceAccount: w.username,
		Channel:       email.Folder,
		Sender:        email.From,
		MessageID:     email.MessageID,
		MessageType:   msgType,
		RawContent:    content,
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}
}

func (w *EmailWatcher) isMentioned(email EmailMessage) bool {
	lower := strings.ToLower(w.username)
	return containsAddress(email.To, lower) ||
		containsAddress(email.CC, lower) ||
		containsAddress(email.BCC, lower)
}

func containsAddress(addrs []string, target string) bool {
	for _, addr := range addrs {
		if strings.ToLower(addr) == target {
			return true
		}
	}
	return false
}

// updateLastUID advances the high-water mark if uid is newer.
func (w *EmailWatcher) updateLastUID(uid uint32) {
	if uid > w.lastUID {
		w.lastUID = uid
	}
}
