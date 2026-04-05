package presenter_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mocks ---

type mockMessageQuerier struct {
	messages []*repository.Message
	err      error
	called   bool
	status   string
}

func (m *mockMessageQuerier) QueryByStatus(_ context.Context, status string) ([]*repository.Message, error) {
	m.called = true
	m.status = status
	return m.messages, m.err
}

type mockMessageUpdater struct {
	updated *repository.Message
	err     error
	called  bool
}

func (m *mockMessageUpdater) Update(_ context.Context, msg *repository.Message) error {
	m.called = true
	m.updated = msg
	return m.err
}

// --- Suite ---

type NotificationPresenterSuite struct {
	suite.Suite
	querier   *mockMessageQuerier
	updater   *mockMessageUpdater
	presenter *presenter.NotificationPresenter
}

func TestNotificationPresenter(t *testing.T) {
	suite.Run(t, new(NotificationPresenterSuite))
}

func (s *NotificationPresenterSuite) SetupTest() {
	s.querier = &mockMessageQuerier{}
	s.updater = &mockMessageUpdater{}
	p, err := presenter.NewNotificationPresenter(s.querier, s.updater)
	s.Require().NoError(err)
	s.presenter = p
}

// --- Constructor validation ---

func (s *NotificationPresenterSuite) TestNewPresenterNilQuerierReturnsError() {
	_, err := presenter.NewNotificationPresenter(nil, s.updater)
	s.Error(err)
}

func (s *NotificationPresenterSuite) TestNewPresenterNilUpdaterReturnsError() {
	_, err := presenter.NewNotificationPresenter(s.querier, nil)
	s.Error(err)
}

// --- Refresh ---

func (s *NotificationPresenterSuite) TestRefreshPopulatesMessagesNewestFirst() {
	now := time.Now()
	older := now.Add(-10 * time.Minute)
	oldest := now.Add(-20 * time.Minute)

	s.querier.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "alice", Channel: "general", RawContent: "oldest msg", CreatedAt: oldest, Status: "Notified"},
		{ID: uuid.New(), Source: "email", Sender: "bob", Channel: "inbox", RawContent: "newer msg", CreatedAt: older, Status: "Notified"},
		{ID: uuid.New(), Source: "slack", Sender: "carol", Channel: "alerts", RawContent: "newest msg", CreatedAt: now, Status: "Notified"},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)
	s.True(s.querier.called)
	s.Equal("Notified", s.querier.status)

	rows := s.presenter.Messages()
	s.Require().Len(rows, 3)
	// Newest first
	s.Equal("carol", rows[0].Sender)
	s.Equal("bob", rows[1].Sender)
	s.Equal("alice", rows[2].Sender)
}

func (s *NotificationPresenterSuite) TestRefreshEmptyResultReturnsEmptyList() {
	s.querier.messages = []*repository.Message{}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	rows := s.presenter.Messages()
	s.Empty(rows)
}

func (s *NotificationPresenterSuite) TestRefreshPropagatesRepositoryError() {
	s.querier.err = errors.New("database exploded")

	err := s.presenter.Refresh(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "database exploded")
}

// --- Messages / Truncation ---

func (s *NotificationPresenterSuite) TestMessagesReturnsTruncatedRows() {
	s.querier.messages = []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack-workspace-very-long-name",
			Sender:     "extremely-long-sender-name",
			Channel:    "super-long-channel-name-here",
			RawContent: "This is a message preview that should be shown",
			CreatedAt:  time.Now(),
			Status:     "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	rows := s.presenter.Messages()
	s.Require().Len(rows, 1)

	s.Len(rows[0].Source, 15)
	s.Len(rows[0].Sender, 15)
	s.Len(rows[0].Channel, 15)
}

func (s *NotificationPresenterSuite) TestTruncationShorterThan15Unchanged() {
	s.querier.messages = []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Sender:     "alice",
			Channel:    "general",
			RawContent: "hello",
			CreatedAt:  time.Now(),
			Status:     "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	rows := s.presenter.Messages()
	s.Require().Len(rows, 1)
	s.Equal("slack", rows[0].Source)
	s.Equal("alice", rows[0].Sender)
	s.Equal("general", rows[0].Channel)
}

func (s *NotificationPresenterSuite) TestTruncationExactly15Unchanged() {
	exactly15 := "123456789012345" // 15 chars
	s.Require().Len(exactly15, 15)

	s.querier.messages = []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     exactly15,
			Sender:     exactly15,
			Channel:    exactly15,
			RawContent: "msg",
			CreatedAt:  time.Now(),
			Status:     "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	rows := s.presenter.Messages()
	s.Require().Len(rows, 1)
	s.Equal(exactly15, rows[0].Source)
	s.Equal(exactly15, rows[0].Sender)
	s.Equal(exactly15, rows[0].Channel)
}

func (s *NotificationPresenterSuite) TestTruncationLongerThan15Truncates() {
	long := "1234567890123456789" // 19 chars
	s.Require().Greater(len(long), 15)

	s.querier.messages = []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     long,
			Sender:     long,
			Channel:    long,
			RawContent: "msg",
			CreatedAt:  time.Now(),
			Status:     "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	rows := s.presenter.Messages()
	s.Require().Len(rows, 1)
	s.Len(rows[0].Source, 15)
	s.Len(rows[0].Sender, 15)
	s.Len(rows[0].Channel, 15)
}

func (s *NotificationPresenterSuite) TestPreviewTruncatedTo80Chars() {
	longContent := strings.Repeat("x", 200)

	s.querier.messages = []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Sender:     "alice",
			Channel:    "general",
			RawContent: longContent,
			CreatedAt:  time.Now(),
			Status:     "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	rows := s.presenter.Messages()
	s.Require().Len(rows, 1)
	s.LessOrEqual(len(rows[0].Preview), 80)
}

// --- Select ---

func (s *NotificationPresenterSuite) TestSelectSetsExpandedMessage() {
	now := time.Now()
	msgID := uuid.New()

	s.querier.messages = []*repository.Message{
		{
			ID:              msgID,
			Source:          "slack",
			Sender:          "alice",
			Channel:         "general",
			RawContent:      "Full message content here with all the details",
			ImportanceScore: 8.5,
			ConfidenceScore: 0.92,
			CreatedAt:       now,
			Status:          "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	detail, err := s.presenter.Select(0)
	s.Require().NoError(err)
	s.Equal(msgID, detail.ID)
	s.Equal("Full message content here with all the details", detail.Content)
	s.InDelta(8.5, detail.ImportanceScore, 0.001)
	s.InDelta(0.92, detail.ConfidenceScore, 0.001)
	s.Equal(now.Unix(), detail.CreatedAt.Unix())
}

func (s *NotificationPresenterSuite) TestSelectInvalidIndexReturnsError() {
	s.querier.messages = []*repository.Message{
		{
			ID:        uuid.New(),
			Source:    "slack",
			Sender:    "alice",
			Channel:   "general",
			CreatedAt: time.Now(),
			Status:    "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	_, err = s.presenter.Select(-1)
	s.Error(err)

	_, err = s.presenter.Select(99)
	s.Error(err)
}

// --- Resolve ---

func (s *NotificationPresenterSuite) TestResolveUpdatesMessageAndRemovesFromList() {
	msgID := uuid.New()

	s.querier.messages = []*repository.Message{
		{
			ID:         msgID,
			Source:     "slack",
			Sender:     "alice",
			Channel:    "general",
			RawContent: "important message",
			CreatedAt:  time.Now(),
			Status:     "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)
	s.Require().Len(s.presenter.Messages(), 1)

	err = s.presenter.Resolve(context.Background(), msgID)
	s.Require().NoError(err)

	// Verify updater was called with correct status
	s.True(s.updater.called)
	s.Equal("Resolved", s.updater.updated.Status)
	s.NotNil(s.updater.updated.ResolvedAt)

	// Message removed from list
	s.Empty(s.presenter.Messages())
}

func (s *NotificationPresenterSuite) TestResolveUnknownIDReturnsError() {
	s.querier.messages = []*repository.Message{
		{
			ID:        uuid.New(),
			Source:    "slack",
			Sender:    "alice",
			Channel:   "general",
			CreatedAt: time.Now(),
			Status:    "Notified",
		},
	}

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	unknownID := uuid.New()
	err = s.presenter.Resolve(context.Background(), unknownID)
	s.Error(err)
}

func (s *NotificationPresenterSuite) TestResolveUpdaterErrorPropagates() {
	msgID := uuid.New()

	s.querier.messages = []*repository.Message{
		{
			ID:        msgID,
			Source:    "slack",
			Sender:    "alice",
			Channel:   "general",
			CreatedAt: time.Now(),
			Status:    "Notified",
		},
	}
	s.updater.err = errors.New("update failed")

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	err = s.presenter.Resolve(context.Background(), msgID)
	s.Error(err)
	s.Contains(err.Error(), "update failed")
}
