package presenter_test

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Suite ---

type NotificationCardSuite struct {
	suite.Suite
}

func TestNotificationCard(t *testing.T) {
	suite.Run(t, new(NotificationCardSuite))
}

// --- Helpers ---

func (s *NotificationCardSuite) makeMessage(is float64, cs float64, content string, createdAt time.Time) *repository.Message {
	return &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "general",
		Sender:          "alice",
		MessageID:       "msg-" + uuid.New().String(),
		MessageType:     "message",
		RawContent:      content,
		ImportanceScore: is,
		ConfidenceScore: cs,
		Status:          "Notified",
		Reasoning:       "test reasoning",
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

func parseHexColor(hex string) color.Color {
	var r, g, b uint8
	switch hex {
	case "#2d2d2d":
		r, g, b = 0x2d, 0x2d, 0x2d
	case "#ffc9c9":
		r, g, b = 0xff, 0xc9, 0xc9
	case "#ef4444":
		r, g, b = 0xef, 0x44, 0x44
	case "#ffd8a8":
		r, g, b = 0xff, 0xd8, 0xa8
	case "#f59e0b":
		r, g, b = 0xf5, 0x9e, 0x0b
	case "#dbe4ff":
		r, g, b = 0xdb, 0xe4, 0xff
	case "#4a9eed":
		r, g, b = 0x4a, 0x9e, 0xed
	}
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}
}

// --- Color Tier Tests ---

func (s *NotificationCardSuite) TestHighImportanceRedCard() {
	now := time.Now()
	msg := s.makeMessage(9.0, 0.9, "Server is down", now.Add(-2*time.Minute))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	card := cards[0]
	s.Equal(parseHexColor("#2d2d2d"), card.CardColor, "IS>=9 should have dark card background")
	s.Equal(parseHexColor("#ef4444"), card.BadgeColor, "IS>=9 should have red badge")
}

func (s *NotificationCardSuite) TestMidImportanceOrangeCard() {
	now := time.Now()
	msg := s.makeMessage(8.5, 0.85, "Deployment pending", now.Add(-3*time.Minute))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	card := cards[0]
	s.Equal(parseHexColor("#2d2d2d"), card.CardColor, "IS>=8 should have dark card background")
	s.Equal(parseHexColor("#f59e0b"), card.BadgeColor, "IS>=8 should have amber badge")
}

func (s *NotificationCardSuite) TestLowImportanceBlueCard() {
	now := time.Now()
	msg := s.makeMessage(7.2, 0.82, "New PR opened", now.Add(-10*time.Minute))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	card := cards[0]
	s.Equal(parseHexColor("#2d2d2d"), card.CardColor, "IS<8 should have dark card background")
	s.Equal(parseHexColor("#4a9eed"), card.BadgeColor, "IS<8 should have blue badge")
}

// --- Opacity Tests ---

func (s *NotificationCardSuite) TestOpacityScalesWithIS() {
	now := time.Now()

	tests := []struct {
		is              float64
		expectedOpacity float64
		tolerance       float64
		label           string
	}{
		{10.0, 0.4, 0.001, "IS=10 should be 0.4 opacity"},
		{9.0, 1.0/3.0 + 0.0667, 0.02, "IS=9 should be ~0.333"},
		{7.0, 0.2, 0.001, "IS=7 should be 0.2 opacity"},
	}

	for _, tc := range tests {
		msg := s.makeMessage(tc.is, 0.9, "test", now.Add(-1*time.Minute))
		cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)
		s.Require().Len(cards, 1, tc.label)
		// Linear interpolation: opacity = 0.2 + (IS - 7.0) / (10.0 - 7.0) * (0.4 - 0.2)
		expected := 0.2 + (tc.is-7.0)/3.0*0.2
		s.InDelta(expected, cards[0].Opacity, 0.001, tc.label)
	}
}

// --- Relative Time Tests ---

func (s *NotificationCardSuite) TestRelativeTimeMinutes() {
	now := time.Now()
	msg := s.makeMessage(9.0, 0.9, "alert", now.Add(-5*time.Minute))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	s.Equal("5m ago", cards[0].RelativeTime)
}

func (s *NotificationCardSuite) TestRelativeTimeHours() {
	now := time.Now()
	msg := s.makeMessage(9.0, 0.9, "alert", now.Add(-2*time.Hour))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	s.Equal("2h ago", cards[0].RelativeTime)
}

func (s *NotificationCardSuite) TestRelativeTimeJustNow() {
	now := time.Now()
	msg := s.makeMessage(9.0, 0.9, "alert", now.Add(-30*time.Second))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	s.Equal("<1m ago", cards[0].RelativeTime)
}

// --- Content Tests ---

func (s *NotificationCardSuite) TestMessagePreviewTruncated() {
	now := time.Now()
	longContent := strings.Repeat("a", 200)
	msg := s.makeMessage(9.0, 0.9, longContent, now.Add(-1*time.Minute))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	card := cards[0]

	s.Equal(longContent, card.FullContent, "FullContent should contain the entire message")
	s.Less(len(card.MessagePreview), len(longContent), "MessagePreview should be shorter than full content")
	s.LessOrEqual(len(card.MessagePreview), 80, "MessagePreview should be at most 80 characters")
	s.True(strings.HasSuffix(card.MessagePreview, "..."), "Truncated preview should end with ellipsis")
}

// --- Field Population Test ---

func (s *NotificationCardSuite) TestCardFieldsPopulated() {
	now := time.Now()
	msgID := uuid.New()
	msg := &repository.Message{
		ID:              msgID,
		Source:          "email",
		SourceAccount:   "acct-1",
		Channel:         "inbox",
		Sender:          "bob@example.com",
		MessageID:       "email-123",
		MessageType:     "message",
		RawContent:      "Hello world",
		ImportanceScore: 8.0,
		ConfidenceScore: 0.75,
		Status:          "Notified",
		Reasoning:       "important",
		CreatedAt:       now.Add(-3 * time.Minute),
		UpdatedAt:       now.Add(-3 * time.Minute),
	}
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	card := cards[0]

	s.Equal(msgID, card.ID)
	s.Equal(8.0, card.ImportanceScore)
	s.Equal("email", card.Source)
	s.Equal("inbox", card.Channel)
	s.Equal("bob@example.com", card.Sender)
	s.Equal(0.75, card.ConfidenceScore)
	s.Equal("Hello world", card.FullContent)
	s.Equal(msg.CreatedAt, card.CreatedAt)
}

// --- Dark Card Background Tests ---

func (s *NotificationCardSuite) TestHighImportanceCardHasDarkBackground() {
	now := time.Now()
	msg := s.makeMessage(9.5, 0.95, "Production database is down", now.Add(-1*time.Minute))
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	card := cards[0]

	// Card background must be dark for readability (R, G, B each < 0x60)
	r, g, b, _ := card.CardColor.RGBA()
	// RGBA() returns pre-multiplied 16-bit values; shift to 8-bit for comparison
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	const darkThreshold = uint8(0x60)
	s.Less(r8, darkThreshold, "card background red channel should be dark (< 0x60), got 0x%02x", r8)
	s.Less(g8, darkThreshold, "card background green channel should be dark (< 0x60), got 0x%02x", g8)
	s.Less(b8, darkThreshold, "card background blue channel should be dark (< 0x60), got 0x%02x", b8)

	// Badge color should still be tier-specific red
	s.Equal(parseHexColor("#ef4444"), card.BadgeColor, "IS>=9 badge should remain red")
}

// --- WebURL Propagation ---

func (s *NotificationCardSuite) TestCardWebURLPopulatedFromMessage() {
	now := time.Now()
	const url = "https://acme.slack.com/archives/C123"
	msg := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "T1",
		Channel:         "general",
		Sender:          "alice",
		RawContent:      "ping",
		WebURL:          url,
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Status:          "Notified",
		CreatedAt:       now.Add(-1 * time.Minute),
	}
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	s.Equal(url, cards[0].WebURL, "card WebURL should mirror message WebURL")
}

// --- Email Subject Preview ---

func (s *NotificationCardSuite) TestEmailMessageUsesSubjectForPreview() {
	now := time.Now()
	msg := &repository.Message{
		ID:              uuid.New(),
		Source:          "email",
		SourceAccount:   "user@example.com",
		Channel:         "inbox",
		Sender:          "boss@example.com",
		Subject:         "Q4 deadline reminder",
		RawContent:      "Hi team, the deadline is Friday. Please send your sections...",
		ImportanceScore: 8.0,
		ConfidenceScore: 0.9,
		Status:          "Notified",
		CreatedAt:       now.Add(-1 * time.Minute),
	}
	cards := presenter.BuildNotificationCards([]*repository.Message{msg}, now)

	s.Require().Len(cards, 1)
	s.Equal("Q4 deadline reminder", cards[0].MessagePreview,
		"email preview should be the subject line, not the body")
}

// --- FormatDisplayLine Tests ---

func (s *NotificationCardSuite) TestFormatDisplayLineBySource() {
	tests := []struct {
		name       string
		source     string
		channel    string
		rawContent string
		expected   string
	}{
		{
			name:       "email returns subject line only",
			source:     "email",
			channel:    "inbox",
			rawContent: "Important Subject\n<html>body...</html>",
			expected:   "Important Subject",
		},
		{
			name:       "slack returns channel prefix with truncated preview",
			source:     "slack",
			channel:    "general",
			rawContent: "Hey everyone check this out",
			expected:   "#general: Hey everyone che...",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result := presenter.FormatDisplayLine(tc.source, tc.channel, tc.rawContent)
			s.Equal(tc.expected, result)
		})
	}
}

// --- Empty Input Test ---

func (s *NotificationCardSuite) TestEmptyMessagesReturnsEmptySlice() {
	now := time.Now()

	cards := presenter.BuildNotificationCards(nil, now)
	s.NotNil(cards, "nil input should return empty slice, not nil")
	s.Empty(cards)

	cards = presenter.BuildNotificationCards([]*repository.Message{}, now)
	s.NotNil(cards, "empty input should return empty slice, not nil")
	s.Empty(cards)
}
