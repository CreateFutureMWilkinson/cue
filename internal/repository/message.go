package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Message represents a message event stored in the database.
type Message struct {
	ID              uuid.UUID
	Source          string // "email" | "slack"
	SourceAccount   string
	Channel         string
	Sender          string
	MessageID       string // Source-native message ID
	MessageType     string // "message", "channel_join", etc.
	SourceCursor    string // Source-native cursor (Slack ts, IMAP UID)
	RawContent      string
	ImportanceScore float64 // 0–10
	ConfidenceScore float64 // 0.0–1.0
	Status          string  // "Pending", "Notified", "Buffered", "Ignored", "Resolved"
	Reasoning       string
	UserRating      *int       // nullable
	UserFeedback    *string    // nullable
	VectorID        *uuid.UUID // nullable
	ScoringModel    string     // Model used for scoring (e.g. "neural-chat")
	ExamplesUsed    int        // Number of few-shot examples used in prompt
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ResolvedAt      *time.Time // nullable
}

// MessageFilter specifies optional criteria for filtering messages.
// Zero-value fields are ignored (treated as "no filter").
type MessageFilter struct {
	Status  string     // optional: filter by status
	Source  string     // optional: filter by source ("slack", "email")
	Channel string     // optional: filter by channel name
	Since   *time.Time // optional: only messages created after this time
	Limit   int        // page size (default 50, max 200)
	Offset  int        // pagination offset
}

// MessageRepository defines the contract for message persistence.
type MessageRepository interface {
	Insert(ctx context.Context, msg *Message) error
	Update(ctx context.Context, msg *Message) error
	QueryByID(ctx context.Context, id uuid.UUID) (*Message, error)
	QueryByStatus(ctx context.Context, status string) ([]*Message, error)
	QueryFiltered(ctx context.Context, filter MessageFilter) ([]*Message, int, error)
	QueryAll(ctx context.Context) ([]*Message, error)
	QueryOldestToNewest(ctx context.Context, limit int) ([]*Message, error)
	CountBySource(ctx context.Context, source string) (int, error)
	ExistsByMessageID(ctx context.Context, messageID string) (bool, error)
	MaxSourceCursor(ctx context.Context, source, sourceAccount, channel string) (string, error)
	DistinctChannels(ctx context.Context, source, sourceAccount string) ([]string, error)
}
