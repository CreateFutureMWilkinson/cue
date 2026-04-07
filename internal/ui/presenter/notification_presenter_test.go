package presenter_test

import (
	"context"
	"errors"
	"image/color"
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

// --- Expand/Collapse + Dismiss Suite ---

type NotificationPresenterExpandSuite struct {
	suite.Suite
	querier   *mockMessageQuerier
	updater   *mockMessageUpdater
	presenter *presenter.NotificationPresenter
}

func TestNotificationPresenterExpand(t *testing.T) {
	suite.Run(t, new(NotificationPresenterExpandSuite))
}

func (s *NotificationPresenterExpandSuite) SetupTest() {
	s.querier = &mockMessageQuerier{}
	s.updater = &mockMessageUpdater{}
	p, err := presenter.NewNotificationPresenter(s.querier, s.updater)
	s.Require().NoError(err)
	s.presenter = p
}

func (s *NotificationPresenterExpandSuite) TestDefaultIsCollapsed() {
	s.False(s.presenter.IsExpanded())
}

func (s *NotificationPresenterExpandSuite) TestToggleExpands() {
	s.presenter.ToggleExpanded()
	s.True(s.presenter.IsExpanded())
}

func (s *NotificationPresenterExpandSuite) TestToggleCollapses() {
	s.presenter.ToggleExpanded() // expand
	s.presenter.ToggleExpanded() // collapse
	s.False(s.presenter.IsExpanded())
}

func (s *NotificationPresenterExpandSuite) TestExpandedChangeCallbackFires() {
	var callbackValues []bool
	s.presenter.SetOnExpandedChange(func(expanded bool) {
		callbackValues = append(callbackValues, expanded)
	})

	s.presenter.ToggleExpanded() // expand -> callback(true)
	s.presenter.ToggleExpanded() // collapse -> callback(false)

	s.Require().Len(callbackValues, 2)
	s.True(callbackValues[0])
	s.False(callbackValues[1])
}

func (s *NotificationPresenterExpandSuite) TestDismissMarksResolved() {
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

	err = s.presenter.DismissMessage(context.Background(), msgID)
	s.Require().NoError(err)

	s.True(s.updater.called)
	s.Equal("Resolved", s.updater.updated.Status)
	s.NotNil(s.updater.updated.ResolvedAt)
}

func (s *NotificationPresenterExpandSuite) TestDismissRemovesFromList() {
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

	err = s.presenter.DismissMessage(context.Background(), msgID)
	s.Require().NoError(err)

	s.Empty(s.presenter.Messages())
}

func (s *NotificationPresenterExpandSuite) TestDismissUnknownIDReturnsError() {
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
	err = s.presenter.DismissMessage(context.Background(), unknownID)
	s.Error(err)
}

func (s *NotificationPresenterExpandSuite) TestDismissUpdaterErrorPropagates() {
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
	s.updater.err = errors.New("dismiss update failed")

	err := s.presenter.Refresh(context.Background())
	s.Require().NoError(err)

	err = s.presenter.DismissMessage(context.Background(), msgID)
	s.Error(err)
	s.Contains(err.Error(), "dismiss update failed")
}

// --- Cards() method tests (Feature 018-Hotfix-A) ---

type NotificationPresenterCardsSuite struct {
	suite.Suite
	querier   *mockMessageQuerier
	updater   *mockMessageUpdater
	presenter *presenter.NotificationPresenter
}

func TestNotificationPresenterCards(t *testing.T) {
	suite.Run(t, new(NotificationPresenterCardsSuite))
}

func (s *NotificationPresenterCardsSuite) SetupTest() {
	now := time.Now()
	s.querier = &mockMessageQuerier{
		messages: []*repository.Message{
			{
				ID:              uuid.New(),
				Source:          "slack",
				Sender:          "alice",
				Channel:         "general",
				RawContent:      "Server is on fire!",
				ImportanceScore: 9.0,
				ConfidenceScore: 0.95,
				Reasoning:       "Server outage detected with high urgency keywords",
				Status:          "Notified",
				CreatedAt:       now.Add(-5 * time.Minute),
			},
			{
				ID:              uuid.New(),
				Source:          "email",
				Sender:          "bob@example.com",
				Channel:         "inbox",
				RawContent:      "Quarterly report deadline tomorrow",
				ImportanceScore: 7.5,
				ConfidenceScore: 0.85,
				Reasoning:       "Upcoming deadline with moderate urgency",
				Status:          "Notified",
				CreatedAt:       now.Add(-10 * time.Minute),
			},
		},
	}
	s.updater = &mockMessageUpdater{}
	p, err := presenter.NewNotificationPresenter(s.querier, s.updater)
	s.Require().NoError(err)
	err = p.Refresh(context.Background())
	s.Require().NoError(err)
	s.presenter = p
}

func (s *NotificationPresenterCardsSuite) TestCardsReturnsNotificationCards() {
	s.T().Helper()

	cards := s.presenter.Cards()

	s.Require().Len(cards, 2, "Cards() should return one card per message")
	s.Equal("Server is on fire!", cards[0].FullContent)
	s.Equal("Quarterly report deadline tomorrow", cards[1].FullContent)
}

func (s *NotificationPresenterCardsSuite) TestCardsHasCorrectColors() {
	s.T().Helper()

	cards := s.presenter.Cards()
	s.Require().Len(cards, 2)

	// All tiers use dark card background
	expectedDarkCard := color.NRGBA{R: 0x2d, G: 0x2d, B: 0x2d, A: 0xff}

	// IS=9.0 -> red badge
	expectedHighBadge := color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff}
	s.Equal(expectedDarkCard, cards[0].CardColor, "IS=9.0 card should have dark background")
	s.Equal(expectedHighBadge, cards[0].BadgeColor, "IS=9.0 badge should be red")

	// IS=7.5 -> blue badge
	expectedLowBadge := color.NRGBA{R: 0x4a, G: 0x9e, B: 0xed, A: 0xff}
	s.Equal(expectedDarkCard, cards[1].CardColor, "IS=7.5 card should have dark background")
	s.Equal(expectedLowBadge, cards[1].BadgeColor, "IS=7.5 badge should be blue")
}

func (s *NotificationPresenterCardsSuite) TestCardsHasRelativeTime() {
	s.T().Helper()

	cards := s.presenter.Cards()
	s.Require().Len(cards, 2)

	s.NotEmpty(cards[0].RelativeTime, "first card should have non-empty RelativeTime")
	s.NotEmpty(cards[1].RelativeTime, "second card should have non-empty RelativeTime")
}

func (s *NotificationPresenterCardsSuite) TestSelectReturnsReasoning() {
	s.T().Helper()

	detail, err := s.presenter.Select(0)
	s.Require().NoError(err)

	s.Equal("Server outage detected with high urgency keywords", detail.Reasoning,
		"Select() should return Reasoning from the underlying message")
}
