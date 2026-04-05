package presenter

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const (
	truncateLen    = 15
	previewLen     = 80
	statusNotified = "Notified"
	statusResolved = "Resolved"
)

type NotificationRow struct {
	Source  string
	Sender  string
	Channel string
	Preview string
}

type NotificationDetail struct {
	ID              uuid.UUID
	Content         string
	ImportanceScore float64
	ConfidenceScore float64
	CreatedAt       time.Time
}

type NotificationPresenter struct {
	querier  MessageQuerier
	updater  MessageUpdater
	messages []*repository.Message
}

func NewNotificationPresenter(querier MessageQuerier, updater MessageUpdater) (*NotificationPresenter, error) {
	if querier == nil {
		return nil, fmt.Errorf("querier must not be nil")
	}
	if updater == nil {
		return nil, fmt.Errorf("updater must not be nil")
	}
	return &NotificationPresenter{
		querier: querier,
		updater: updater,
	}, nil
}

func (p *NotificationPresenter) Refresh(ctx context.Context) error {
	msgs, err := p.querier.QueryByStatus(ctx, statusNotified)
	if err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.After(msgs[j].CreatedAt)
	})
	p.messages = msgs
	return nil
}

func (p *NotificationPresenter) Messages() []NotificationRow {
	rows := make([]NotificationRow, len(p.messages))
	for i, m := range p.messages {
		rows[i] = NotificationRow{
			Source:  truncate(m.Source, truncateLen),
			Sender:  truncate(m.Sender, truncateLen),
			Channel: truncate(m.Channel, truncateLen),
			Preview: truncate(m.RawContent, previewLen),
		}
	}
	return rows
}

func (p *NotificationPresenter) Select(index int) (*NotificationDetail, error) {
	if index < 0 || index >= len(p.messages) {
		return nil, fmt.Errorf("select: index %d out of range [0, %d)", index, len(p.messages))
	}
	m := p.messages[index]
	return &NotificationDetail{
		ID:              m.ID,
		Content:         m.RawContent,
		ImportanceScore: m.ImportanceScore,
		ConfidenceScore: m.ConfidenceScore,
		CreatedAt:       m.CreatedAt,
	}, nil
}

func (p *NotificationPresenter) Resolve(ctx context.Context, id uuid.UUID) error {
	idx := -1
	for i, m := range p.messages {
		if m.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("resolve: message %s not found", id)
	}

	msg := p.messages[idx]
	now := time.Now()
	msg.Status = statusResolved
	msg.ResolvedAt = &now

	if err := p.updater.Update(ctx, msg); err != nil {
		// Undo in-memory mutation so the presenter stays consistent on failure.
		msg.Status = statusNotified
		msg.ResolvedAt = nil
		return fmt.Errorf("resolve: %w", err)
	}

	p.messages = append(p.messages[:idx], p.messages[idx+1:]...)
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
