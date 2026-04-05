package presenter

import (
	"fmt"
	"image/color"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
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
			MessagePreview:  truncatePreview(msg.RawContent),
			RelativeTime:    relativeTime(msg.CreatedAt, now),
		}

		card.CardColor, card.BadgeColor = colorTier(msg.ImportanceScore)
		card.Opacity = opacity(msg.ImportanceScore)

		cards = append(cards, card)
	}

	return cards
}

func truncatePreview(content string) string {
	if len(content) > 80 {
		return content[:77] + "..."
	}
	return content
}

func relativeTime(created time.Time, now time.Time) string {
	d := now.Sub(created)
	if d < time.Minute {
		return "<1m ago"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}

func colorTier(is float64) (cardColor, badgeColor color.Color) {
	if is >= 9 {
		return color.NRGBA{R: 0xff, G: 0xc9, B: 0xc9, A: 0xff},
			color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff}
	}
	if is >= 8 {
		return color.NRGBA{R: 0xff, G: 0xd8, B: 0xa8, A: 0xff},
			color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}
	}
	return color.NRGBA{R: 0xdb, G: 0xe4, B: 0xff, A: 0xff},
		color.NRGBA{R: 0x4a, G: 0x9e, B: 0xed, A: 0xff}
}

func opacity(is float64) float64 {
	o := 0.2 + (is-7.0)/3.0*0.2
	if o < 0.2 {
		return 0.2
	}
	if o > 0.4 {
		return 0.4
	}
	return o
}
