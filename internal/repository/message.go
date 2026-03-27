package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Message represents a message event stored in the database.
type Message struct {
	ID              uuid.UUID
	Source          string    // "email" | "slack"
	SourceAccount   string
	Channel         string
	Sender          string
	MessageID       string    // Source-native message ID
	MessageType     string    // "message", "channel_join", etc.
	RawContent      string
	ImportanceScore float64   // 0–10
	ConfidenceScore float64   // 0.0–1.0
	Status          string    // "Pending", "Notified", "Buffered", "Ignored", "Resolved"
	Reasoning       string
	UserRating      *int      // nullable
	UserFeedback    *string   // nullable
	VectorID        *uuid.UUID // nullable
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ResolvedAt      *time.Time // nullable
}

// MessageRepository defines the contract for message persistence.
type MessageRepository interface {
	Insert(ctx context.Context, msg *Message) error
	Update(ctx context.Context, msg *Message) error
	QueryByStatus(ctx context.Context, status string) ([]*Message, error)
	QueryAll(ctx context.Context) ([]*Message, error)
	QueryOldestToNewest(ctx context.Context, limit int) ([]*Message, error)
	CountBySource(ctx context.Context, source string) (int, error)
}
