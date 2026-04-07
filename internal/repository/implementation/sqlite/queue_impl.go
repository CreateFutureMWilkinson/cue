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
	id := uuid.New()
	enqueuedAt := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO ollama_queue (id, message_id, enqueued_at, status) VALUES (?, ?, ?, ?)",
		id.String(), messageID.String(), enqueuedAt, "pending",
	)
	if err != nil {
		return fmt.Errorf("enqueue message %s: %w", messageID, err)
	}
	return nil
}

func (r *SQLiteQueueRepository) DequeueOldest(ctx context.Context) (*repository.QueueEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var idStr, messageIDStr, enqueuedAtStr string
	err = tx.QueryRowContext(ctx,
		"SELECT id, message_id, enqueued_at FROM ollama_queue WHERE status = 'pending' ORDER BY enqueued_at ASC LIMIT 1",
	).Scan(&idStr, &messageIDStr, &enqueuedAtStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select oldest pending: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE ollama_queue SET status = 'processing' WHERE id = ?", idStr,
	)
	if err != nil {
		return nil, fmt.Errorf("update status to processing: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse queue entry id: %w", err)
	}
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse message id: %w", err)
	}
	enqueuedAt, err := time.Parse(time.RFC3339, enqueuedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse enqueued_at: %w", err)
	}

	return &repository.QueueEntry{
		ID:         id,
		MessageID:  messageID,
		EnqueuedAt: enqueuedAt,
		Status:     "processing",
	}, nil
}

func (r *SQLiteQueueRepository) MarkDone(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		"UPDATE ollama_queue SET status = 'done' WHERE id = ?", id.String(),
	)
	if err != nil {
		return fmt.Errorf("mark done %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark done rows affected: %w", err)
	}
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *SQLiteQueueRepository) MarkFailed(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		"UPDATE ollama_queue SET status = 'failed' WHERE id = ?", id.String(),
	)
	if err != nil {
		return fmt.Errorf("mark failed %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark failed rows affected: %w", err)
	}
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
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
