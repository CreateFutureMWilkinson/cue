package watcher_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/stretchr/testify/suite"
)

// mockEmailAPI implements watcher.EmailAPI for testing.
type mockEmailAPI struct {
	messages    []watcher.EmailMessage
	messagesErr error
}

func (m *mockEmailAPI) FetchNewMessages(ctx context.Context, lastUID uint32) ([]watcher.EmailMessage, error) {
	if m.messagesErr != nil {
		return nil, m.messagesErr
	}
	// Filter messages with UID > lastUID to simulate IMAP behavior
	var result []watcher.EmailMessage
	for _, msg := range m.messages {
		if msg.UID > lastUID {
			result = append(result, msg)
		}
	}
	return result, nil
}

type EmailWatcherSuite struct {
	suite.Suite
}

func TestEmailWatcher(t *testing.T) {
	suite.Run(t, new(EmailWatcherSuite))
}

// --- Constructor tests ---

func (s *EmailWatcherSuite) TestNewEmailWatcher_ValidConfig() {
	api := &mockEmailAPI{}
	cfg := config.EmailConfig{
		Enabled:             true,
		IMAPHost:            "imap.gmail.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		PasswordEnv:         "CUE_EMAIL_PASSWORD",
		PollIntervalSeconds: 600,
	}

	w, err := watcher.NewEmailWatcher(api, cfg)
	s.NoError(err)
	s.NotNil(w)
}

func (s *EmailWatcherSuite) TestNewEmailWatcher_NilAPI() {
	cfg := config.EmailConfig{
		Enabled:  true,
		IMAPHost: "imap.gmail.com",
		IMAPPort: 993,
		Username: "user@example.com",
	}

	w, err := watcher.NewEmailWatcher(nil, cfg)
	s.Error(err)
	s.Nil(w)
	s.Contains(err.Error(), "api")
}

func (s *EmailWatcherSuite) TestNewEmailWatcher_EmptyUsername() {
	api := &mockEmailAPI{}
	cfg := config.EmailConfig{
		Enabled:  true,
		IMAPHost: "imap.gmail.com",
		IMAPPort: 993,
		Username: "",
	}

	w, err := watcher.NewEmailWatcher(api, cfg)
	s.Error(err)
	s.Nil(w)
	s.Contains(err.Error(), "username")
}

// --- Poll: basic message fetching ---

func (s *EmailWatcherSuite) TestPoll_FetchesNewMessages() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "Hello", Folder: "INBOX", Body: "Hi there", To: []string{"team@example.com"}},
			{UID: 2, From: "bob@example.com", Subject: "Meeting", Folder: "INBOX", Body: "Let's meet", To: []string{"team@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Len(msgs, 2)

	for _, msg := range msgs {
		s.Equal("email", msg.Source)
		s.Equal("user@example.com", msg.SourceAccount)
		s.Equal("Pending", msg.Status)
		s.Equal("message", msg.MessageType)
	}
}

func (s *EmailWatcherSuite) TestPoll_MessageFieldsPopulated() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, MessageID: "abc123@example.com", From: "alice@example.com", Subject: "Important", Folder: "INBOX", Body: "Please review", To: []string{"team@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)

	msg := msgs[0]
	s.Equal("email", msg.Source)
	s.Equal("user@example.com", msg.SourceAccount)
	s.Equal("INBOX", msg.Channel)
	s.Equal("alice@example.com", msg.Sender)
	s.Equal("abc123@example.com", msg.MessageID)
	s.Equal("message", msg.MessageType)
	s.Contains(msg.RawContent, "Important")
	s.Contains(msg.RawContent, "Please review")
	s.Equal("Pending", msg.Status)
	s.False(msg.ID.String() == "00000000-0000-0000-0000-000000000000")
	s.False(msg.CreatedAt.IsZero())
}

// --- Poll: @mention detection ---

func (s *EmailWatcherSuite) TestPoll_DetectsMentionInTo() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "Hi", Folder: "INBOX", Body: "content", To: []string{"user@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)
	s.Equal("mention", msgs[0].MessageType)
}

func (s *EmailWatcherSuite) TestPoll_DetectsMentionInCC() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "FYI", Folder: "INBOX", Body: "content", To: []string{"other@example.com"}, CC: []string{"user@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)
	s.Equal("mention", msgs[0].MessageType)
}

func (s *EmailWatcherSuite) TestPoll_DetectsMentionInBCC() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "Secret", Folder: "INBOX", Body: "content", To: []string{"other@example.com"}, BCC: []string{"user@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)
	s.Equal("mention", msgs[0].MessageType)
}

func (s *EmailWatcherSuite) TestPoll_NoMentionWhenUserNotInRecipients() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "Other", Folder: "INBOX", Body: "content", To: []string{"other@example.com"}, CC: []string{"another@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)
	s.Equal("message", msgs[0].MessageType)
}

func (s *EmailWatcherSuite) TestPoll_MentionDetectionIsCaseInsensitive() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "Hi", Folder: "INBOX", Body: "content", To: []string{"User@Example.COM"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)
	s.Equal("mention", msgs[0].MessageType)
}

// --- Poll: content includes subject and body ---

func (s *EmailWatcherSuite) TestPoll_RawContentIncludesSubjectAndBody() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 1, From: "alice@example.com", Subject: "Urgent: Server Down", Folder: "INBOX", Body: "The production server is unresponsive.", To: []string{"user@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)
	s.Contains(msgs[0].RawContent, "Urgent: Server Down")
	s.Contains(msgs[0].RawContent, "The production server is unresponsive.")
}

// --- Poll: tracks last UID to avoid reprocessing ---

func (s *EmailWatcherSuite) TestPoll_TracksLastUIDToAvoidReprocessing() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 10, From: "alice@example.com", Subject: "First", Folder: "INBOX", Body: "first msg", To: []string{"user@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")

	// First poll: gets the message
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Len(msgs, 1)

	// Second poll with same messages: should get nothing (already seen)
	msgs, err = w.Poll(context.Background())
	s.NoError(err)
	s.Empty(msgs)

	// Add a new message with higher UID
	api.messages = append(api.messages, watcher.EmailMessage{
		UID: 20, From: "bob@example.com", Subject: "Second", Folder: "INBOX", Body: "second msg", To: []string{"user@example.com"},
	})

	// Third poll: only gets the new message
	msgs, err = w.Poll(context.Background())
	s.NoError(err)
	s.Len(msgs, 1)
	s.Contains(msgs[0].RawContent, "Second")
}

// --- Poll: error handling ---

func (s *EmailWatcherSuite) TestPoll_FetchError_ReturnsError() {
	api := &mockEmailAPI{
		messagesErr: errors.New("imap connection lost"),
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.Error(err)
	s.Nil(msgs)
	s.Contains(err.Error(), "imap connection lost")
}

// --- Poll: context cancellation ---

func (s *EmailWatcherSuite) TestPoll_ContextCancelled_ReturnsError() {
	api := &mockEmailAPI{}

	w := s.mustNewWatcher(api, "user@example.com")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := w.Poll(ctx)
	s.Error(err)
}

// --- Poll: empty inbox ---

func (s *EmailWatcherSuite) TestPoll_NoMessages_ReturnsEmpty() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Empty(msgs)
}

// --- Poll: multiple UIDs updates to highest ---

func (s *EmailWatcherSuite) TestPoll_MultipleMessages_TracksHighestUID() {
	api := &mockEmailAPI{
		messages: []watcher.EmailMessage{
			{UID: 5, From: "a@example.com", Subject: "A", Folder: "INBOX", Body: "a", To: []string{"user@example.com"}},
			{UID: 15, From: "b@example.com", Subject: "B", Folder: "INBOX", Body: "b", To: []string{"user@example.com"}},
			{UID: 10, From: "c@example.com", Subject: "C", Folder: "INBOX", Body: "c", To: []string{"user@example.com"}},
		},
	}

	w := s.mustNewWatcher(api, "user@example.com")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Len(msgs, 3)

	// Second poll: nothing new since lastUID should be 15
	msgs, err = w.Poll(context.Background())
	s.NoError(err)
	s.Empty(msgs)
}

// --- Helpers ---

func (s *EmailWatcherSuite) mustNewWatcher(api watcher.EmailAPI, username string) *watcher.EmailWatcher {
	cfg := config.EmailConfig{
		Enabled:             true,
		IMAPHost:            "imap.gmail.com",
		IMAPPort:            993,
		Username:            username,
		PasswordEnv:         "CUE_EMAIL_PASSWORD",
		PollIntervalSeconds: 600,
	}
	w, err := watcher.NewEmailWatcher(api, cfg)
	s.Require().NoError(err)
	return w
}
