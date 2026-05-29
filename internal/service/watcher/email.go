package watcher

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// EmailWatcherConfig holds the configuration needed by EmailWatcher.
type EmailWatcherConfig struct {
	Username string
}

const (
	SourceEmail = "email"
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
func NewEmailWatcher(api EmailAPI, cfg EmailWatcherConfig) (*EmailWatcher, error) {
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

// SetLastUID seeds the UID high-water mark so that the next Poll only fetches
// messages newer than uid. Used at startup to resume from the last stored cursor.
func (w *EmailWatcher) SetLastUID(uid uint32) {
	w.lastUID = uid
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
	content := email.Subject + "\n" + email.Body

	return &repository.Message{
		ID:            uuid.New(),
		Source:        SourceEmail,
		SourceAccount: w.username,
		Channel:       email.Folder,
		Sender:        email.From,
		MessageID:     email.MessageID,
		MessageType:   MessageTypeMsg,
		SourceCursor:  strconv.FormatUint(uint64(email.UID), 10),
		Subject:       email.Subject,
		RawContent:    content,
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}
}

// SourceInfo returns the source type and account identifier for this watcher.
func (w *EmailWatcher) SourceInfo() (string, string) {
	return SourceEmail, w.username
}

// SeedCursor parses cursor as a uint32 UID and seeds the high-water mark.
func (w *EmailWatcher) SeedCursor(channel string, cursor string) {
	uid, err := strconv.ParseUint(cursor, 10, 32)
	if err != nil {
		return
	}
	w.SetLastUID(uint32(uid))
}

// updateLastUID advances the high-water mark if uid is newer.
func (w *EmailWatcher) updateLastUID(uid uint32) {
	if uid > w.lastUID {
		w.lastUID = uid
	}
}
