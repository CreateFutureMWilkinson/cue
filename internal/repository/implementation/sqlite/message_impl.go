package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

const createMessagesTable = `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    source_account TEXT NOT NULL,
    channel TEXT NOT NULL,
    sender TEXT NOT NULL,
    message_id TEXT NOT NULL UNIQUE,
    raw_content TEXT NOT NULL,
    importance_score REAL NOT NULL,
    confidence_score REAL NOT NULL,
    status TEXT NOT NULL,
    reasoning TEXT NOT NULL DEFAULT '',
    user_rating INTEGER,
    user_feedback TEXT,
    vector_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_source_created ON messages(source, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
`

const maxMessagesPerSource = 100

// SQLiteMessageRepository implements repository.MessageRepository using SQLite.
type SQLiteMessageRepository struct {
	db *sql.DB
}

// NewSQLiteMessageRepository opens (or creates) a SQLite database at dbPath,
// enables WAL mode, creates the messages table if needed, and returns the repository.
func NewSQLiteMessageRepository(dbPath string) (*SQLiteMessageRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec(createMessagesTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("create messages table: %w", err)
	}

	return &SQLiteMessageRepository{db: db}, nil
}

// Insert inserts a message into the database. If a message with the same MessageID
// already exists, it updates the existing row (upsert). Before inserting, it enforces
// FIFO eviction: if the source already has >= 100 messages, the oldest is deleted.
func (r *SQLiteMessageRepository) Insert(ctx context.Context, msg *repository.Message) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// FIFO eviction: count messages for this source and delete oldest if at capacity.
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE source = ?", msg.Source).Scan(&count)
	if err != nil {
		return fmt.Errorf("count messages by source: %w", err)
	}

	if count >= maxMessagesPerSource {
		_, err = tx.ExecContext(ctx,
			"DELETE FROM messages WHERE id = (SELECT id FROM messages WHERE source = ? ORDER BY created_at ASC LIMIT 1)",
			msg.Source,
		)
		if err != nil {
			return fmt.Errorf("evict oldest message: %w", err)
		}
	}

	// Upsert the message.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (
			id, source, source_account, channel, sender, message_id,
			raw_content, importance_score, confidence_score, status, reasoning,
			user_rating, user_feedback, vector_id, created_at, updated_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			id = excluded.id,
			source = excluded.source,
			source_account = excluded.source_account,
			channel = excluded.channel,
			sender = excluded.sender,
			raw_content = excluded.raw_content,
			importance_score = excluded.importance_score,
			confidence_score = excluded.confidence_score,
			status = excluded.status,
			reasoning = excluded.reasoning,
			user_rating = excluded.user_rating,
			user_feedback = excluded.user_feedback,
			vector_id = excluded.vector_id,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			resolved_at = excluded.resolved_at
	`,
		msg.ID.String(),
		msg.Source,
		msg.SourceAccount,
		msg.Channel,
		msg.Sender,
		msg.MessageID,
		msg.RawContent,
		msg.ImportanceScore,
		msg.ConfidenceScore,
		msg.Status,
		msg.Reasoning,
		nullableInt(msg.UserRating),
		nullableString(msg.UserFeedback),
		nullableUUID(msg.VectorID),
		msg.CreatedAt.Format(time.RFC3339),
		msg.UpdatedAt.Format(time.RFC3339),
		nullableTime(msg.ResolvedAt),
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	return tx.Commit()
}

// Update updates an existing message by ID.
func (r *SQLiteMessageRepository) Update(ctx context.Context, msg *repository.Message) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE messages SET
			source = ?,
			source_account = ?,
			channel = ?,
			sender = ?,
			message_id = ?,
			raw_content = ?,
			importance_score = ?,
			confidence_score = ?,
			status = ?,
			reasoning = ?,
			user_rating = ?,
			user_feedback = ?,
			vector_id = ?,
			updated_at = ?,
			resolved_at = ?
		WHERE id = ?
	`,
		msg.Source,
		msg.SourceAccount,
		msg.Channel,
		msg.Sender,
		msg.MessageID,
		msg.RawContent,
		msg.ImportanceScore,
		msg.ConfidenceScore,
		msg.Status,
		msg.Reasoning,
		nullableInt(msg.UserRating),
		nullableString(msg.UserFeedback),
		nullableUUID(msg.VectorID),
		msg.UpdatedAt.Format(time.RFC3339),
		nullableTime(msg.ResolvedAt),
		msg.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	return nil
}

// QueryByStatus returns all messages with the given status.
func (r *SQLiteMessageRepository) QueryByStatus(ctx context.Context, status string) ([]*repository.Message, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+messageColumns()+" FROM messages WHERE status = ?", status)
	if err != nil {
		return nil, fmt.Errorf("query by status: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// QueryAll returns all messages in the database.
func (r *SQLiteMessageRepository) QueryAll(ctx context.Context) ([]*repository.Message, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+messageColumns()+" FROM messages")
	if err != nil {
		return nil, fmt.Errorf("query all: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// QueryOldestToNewest returns up to limit messages ordered by created_at ascending.
func (r *SQLiteMessageRepository) QueryOldestToNewest(ctx context.Context, limit int) ([]*repository.Message, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+messageColumns()+" FROM messages ORDER BY created_at ASC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("query oldest to newest: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// CountBySource returns the number of messages for the given source.
func (r *SQLiteMessageRepository) CountBySource(ctx context.Context, source string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE source = ?", source).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by source: %w", err)
	}
	return count, nil
}

// messageColumns returns the column list for SELECT queries.
func messageColumns() string {
	return `id, source, source_account, channel, sender, message_id,
		raw_content, importance_score, confidence_score, status, reasoning,
		user_rating, user_feedback, vector_id, created_at, updated_at, resolved_at`
}

// scanMessages scans rows into a slice of Message pointers.
func scanMessages(rows *sql.Rows) ([]*repository.Message, error) {
	var messages []*repository.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return messages, nil
}

// scanMessage scans a single row into a Message.
func scanMessage(rows *sql.Rows) (*repository.Message, error) {
	var (
		msg          repository.Message
		idStr        string
		userRating   sql.NullInt64
		userFeedback sql.NullString
		vectorIDStr  sql.NullString
		createdAtStr string
		updatedAtStr string
		resolvedAt   sql.NullString
	)

	err := rows.Scan(
		&idStr,
		&msg.Source,
		&msg.SourceAccount,
		&msg.Channel,
		&msg.Sender,
		&msg.MessageID,
		&msg.RawContent,
		&msg.ImportanceScore,
		&msg.ConfidenceScore,
		&msg.Status,
		&msg.Reasoning,
		&userRating,
		&userFeedback,
		&vectorIDStr,
		&createdAtStr,
		&updatedAtStr,
		&resolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan message: %w", err)
	}

	msg.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse message ID: %w", err)
	}

	msg.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	msg.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	if userRating.Valid {
		r := int(userRating.Int64)
		msg.UserRating = &r
	}

	if userFeedback.Valid {
		msg.UserFeedback = &userFeedback.String
	}

	if vectorIDStr.Valid {
		vid, err := uuid.Parse(vectorIDStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse vector_id: %w", err)
		}
		msg.VectorID = &vid
	}

	if resolvedAt.Valid {
		t, err := time.Parse(time.RFC3339, resolvedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse resolved_at: %w", err)
		}
		msg.ResolvedAt = &t
	}

	return &msg, nil
}

// nullableInt converts *int to a value suitable for SQL (nil becomes NULL).
func nullableInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// nullableString converts *string to a value suitable for SQL (nil becomes NULL).
func nullableString(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// nullableUUID converts *uuid.UUID to a value suitable for SQL (nil becomes NULL).
func nullableUUID(v *uuid.UUID) interface{} {
	if v == nil {
		return nil
	}
	return v.String()
}

// nullableTime converts *time.Time to a value suitable for SQL (nil becomes NULL).
func nullableTime(v *time.Time) interface{} {
	if v == nil {
		return nil
	}
	return v.Format(time.RFC3339)
}
