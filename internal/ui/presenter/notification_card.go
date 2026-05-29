package presenter

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const (
	// Magic numbers for opacity calculation
	minOpacity    = 0.2
	maxOpacity    = 0.4
	minImportance = 7.0
	maxImportance = 10.0
	// Importance score thresholds for color tiers
	highImportanceThreshold = 9.0
	midImportanceThreshold  = 8.0
)

var (
	// Dark card background for all tiers — text readability on dark theme
	darkCardBackground = color.NRGBA{R: 0x2d, G: 0x2d, B: 0x2d, A: 0xff} // #2d2d2d

	// Badge colors remain tier-specific
	highImportanceBadgeColor = color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff} // Red badge
	midImportanceBadgeColor  = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff} // Amber badge
	lowImportanceBadgeColor  = color.NRGBA{R: 0x4a, G: 0x9e, B: 0xed, A: 0xff} // Blue badge
)

// NotificationCard is a view model for rendering a notification card in the UI.
type NotificationCard struct {
	ID              uuid.UUID
	ImportanceScore float64
	Source          string
	Channel         string
	Sender          string
	MessagePreview  string
	FullContent     string
	ConfidenceScore float64
	CreatedAt       time.Time
	RelativeTime    string
	CardColor       color.Color
	BadgeColor      color.Color
	Opacity         float64
	FriendlyName    string
	DisplayLine     string
	WebURL          string
}

// FormatDisplayLine produces a source-appropriate display line for a notification card.
// For email: returns the subject (first line of rawContent).
// For slack: returns "#channel: preview..." (truncated).
func FormatDisplayLine(source, channel, rawContent string) string {
	switch source {
	case "email":
		if idx := strings.Index(rawContent, "\n"); idx >= 0 {
			return rawContent[:idx]
		}
		return rawContent
	case "slack":
		prefix := "#" + channel + ": "
		maxLen := 29
		available := maxLen - len(prefix)
		if available <= 3 {
			return prefix + "..."
		}
		if len(rawContent) <= available {
			return prefix + rawContent
		}
		return prefix + rawContent[:available-3] + "..."
	default:
		return truncateWithEllipsis(rawContent, 28)
	}
}

// BuildNotificationCards converts repository messages into presentation-ready notification cards.
func BuildNotificationCards(messages []*repository.Message, now time.Time) []NotificationCard {
	cards := make([]NotificationCard, 0, len(messages))

	for _, msg := range messages {
		card := NotificationCard{
			ID:              msg.ID,
			ImportanceScore: msg.ImportanceScore,
			Source:          msg.Source,
			Channel:         msg.Channel,
			Sender:          msg.Sender,
			FullContent:     msg.RawContent,
			ConfidenceScore: msg.ConfidenceScore,
			CreatedAt:       msg.CreatedAt,
			MessagePreview:  buildMessagePreview(msg),
			RelativeTime:    formatRelativeTime(msg.CreatedAt, now),
		}

		card.CardColor, card.BadgeColor = colorTier(msg.ImportanceScore)
		card.Opacity = calculateOpacity(msg.ImportanceScore)

		cards = append(cards, card)
	}

	return cards
}

// buildMessagePreview returns the preview text shown on a notification card.
// Email messages surface the subject line; other sources truncate raw content.
func buildMessagePreview(msg *repository.Message) string {
	if msg.Source == "email" && msg.Subject != "" {
		return msg.Subject
	}
	return truncateWithEllipsis(msg.RawContent, messagePreviewLen)
}

// truncateWithEllipsis truncates the content to maxLen, adding "..." if truncation occurs.
// It preserves 3 characters for the ellipsis when truncating.
func truncateWithEllipsis(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	if maxLen <= 3 {
		return "..."
	}
	return content[:maxLen-3] + "..."
}

// formatRelativeTime converts a timestamp to human-readable relative time.
// Returns "<1m ago" for times less than a minute, "Xm ago" for minutes, or "Xh ago" for hours.
func formatRelativeTime(created time.Time, now time.Time) string {
	duration := now.Sub(created)
	if duration < time.Minute {
		return "<1m ago"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(duration.Hours()))
}

// colorTier determines card and badge colors based on importance score.
// All tiers use a dark card background; badge colors remain tier-specific.
func colorTier(importanceScore float64) (cardColor, badgeColor color.Color) {
	if importanceScore >= highImportanceThreshold {
		return darkCardBackground, highImportanceBadgeColor
	}
	if importanceScore >= midImportanceThreshold {
		return darkCardBackground, midImportanceBadgeColor
	}
	return darkCardBackground, lowImportanceBadgeColor
}

// calculateOpacity computes card opacity based on importance score using linear interpolation.
// Maps IS range [7.0, 10.0] to opacity range [0.2, 0.4] with bounds clamping.
func calculateOpacity(importanceScore float64) float64 {
	// Linear interpolation: opacity = minOpacity + (IS - minImportance) / (maxImportance - minImportance) * (maxOpacity - minOpacity)
	normalizedScore := (importanceScore - minImportance) / (maxImportance - minImportance)
	opacity := minOpacity + normalizedScore*(maxOpacity-minOpacity)

	// Clamp to valid range
	if opacity < minOpacity {
		return minOpacity
	}
	if opacity > maxOpacity {
		return maxOpacity
	}
	return opacity
}
