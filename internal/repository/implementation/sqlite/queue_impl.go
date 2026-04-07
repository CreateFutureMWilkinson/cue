package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const createOllamaQueueTableSQL = `
CREATE TABLE IF NOT EXISTS ollama_queue (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id),
    enqueued_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);
CREATE INDEX IF NOT EXISTS idx_ollama_queue_status_enqueued ON ollama_queue(status, enqueued_at);
`

// Compile-time check that SQLiteQueueRepository satisfies QueueRepository.
var _ repository.QueueRepository = (*SQLiteQueueRepository)(nil)

// SQLiteQueueRepository implements repository.QueueRepository using SQLite.
type SQLiteQueueRepository struct {
	db *sql.DB
}

// NewSQLiteQueueRepository creates a new QueueRepository backed by SQLite.
// It creates the ollama_queue table if it does not exist.
func NewSQLiteQueueRepository(db *sql.DB) (*SQLiteQueueRepository, error) {
	if _, err := db.Exec(createOllamaQueueTableSQL); err != nil {
		return nil, fmt.Errorf("creating ollama_queue table: %w", err)
	}
	return &SQLiteQueueRepository{db: db}, nil
}

func (r *SQLiteQueueRepository) Enqueue(ctx context.Context, messageID uuid.UUID) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) DequeueOldest(ctx context.Context) (*repository.QueueEntry, error) {
	return nil, repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) MarkDone(ctx context.Context, id uuid.UUID) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) MarkFailed(ctx context.Context, id uuid.UUID) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) PendingCount(ctx context.Context) (int, error) {
	return 0, repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) PurgeCompleted(ctx context.Context) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) PurgeOlderThan(ctx context.Context, cutoff time.Time) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) PurgeAll(ctx context.Context) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteQueueRepository) ResetProcessing(ctx context.Context) (int64, error) {
	return 0, repository.ErrNotImplemented
}
