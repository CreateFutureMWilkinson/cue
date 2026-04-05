package presenter

import (
	"fmt"
	"image/color"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const (
	// Message preview length for notification cards
	cardPreviewLen = 80
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
	// Color scheme for high importance (IS >= 9) - Red tier
	highImportanceCardColor  = color.NRGBA{R: 0xff, G: 0xc9, B: 0xc9, A: 0xff} // Light red background
	highImportanceBadgeColor = color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff} // Red badge

	// Color scheme for mid importance (IS >= 8) - Orange tier
	midImportanceCardColor  = color.NRGBA{R: 0xff, G: 0xd8, B: 0xa8, A: 0xff} // Light orange background
	midImportanceBadgeColor = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff} // Amber badge

	// Color scheme for low importance (IS < 8) - Blue tier
	lowImportanceCardColor  = color.NRGBA{R: 0xdb, G: 0xe4, B: 0xff, A: 0xff} // Light blue background
	lowImportanceBadgeColor = color.NRGBA{R: 0x4a, G: 0x9e, B: 0xed, A: 0xff} // Blue badge
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
			MessagePreview:  truncateWithEllipsis(msg.RawContent, cardPreviewLen),
			RelativeTime:    formatRelativeTime(msg.CreatedAt, now),
		}

		card.CardColor, card.BadgeColor = colorTier(msg.ImportanceScore)
		card.Opacity = calculateOpacity(msg.ImportanceScore)

		cards = append(cards, card)
	}

	return cards
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
// Uses a three-tier color scheme: red (IS>=9), orange (IS>=8), blue (IS<8).
func colorTier(importanceScore float64) (cardColor, badgeColor color.Color) {
	if importanceScore >= highImportanceThreshold {
		return highImportanceCardColor, highImportanceBadgeColor
	}
	if importanceScore >= midImportanceThreshold {
		return midImportanceCardColor, midImportanceBadgeColor
	}
	return lowImportanceCardColor, lowImportanceBadgeColor
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
