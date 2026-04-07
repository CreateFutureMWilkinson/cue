package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// QueueEntry represents an item in the Ollama scoring queue.
type QueueEntry struct {
	ID         uuid.UUID
	MessageID  uuid.UUID // FK to messages table
	EnqueuedAt time.Time
	Status     string // "pending", "processing", "done", "failed"
}

// QueueRepository defines the contract for Ollama queue persistence.
type QueueRepository interface {
	Enqueue(ctx context.Context, messageID uuid.UUID) error
	DequeueOldest(ctx context.Context) (*QueueEntry, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID) error
	PendingCount(ctx context.Context) (int, error)
	PurgeCompleted(ctx context.Context) error
	PurgeOlderThan(ctx context.Context, cutoff time.Time) error
	PurgeAll(ctx context.Context) error
	ResetProcessing(ctx context.Context) (int64, error)
}
