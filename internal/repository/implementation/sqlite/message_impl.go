package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

const (
	queryCountBySource        = "SELECT COUNT(*) FROM messages WHERE source = ?"
	queryDeleteOldestBySource = "DELETE FROM messages WHERE id = (SELECT id FROM messages WHERE source = ? ORDER BY created_at ASC LIMIT 1)"
	querySelectByID           = "SELECT " + messageColumnsStr + " FROM messages WHERE id = ?"
	querySelectByStatus       = "SELECT " + messageColumnsStr + " FROM messages WHERE status = ?"
	querySelectAll            = "SELECT " + messageColumnsStr + " FROM messages"
	querySelectOldestLimit    = "SELECT " + messageColumnsStr + " FROM messages ORDER BY created_at ASC LIMIT ?"
)

const messageColumnsStr = "id, source, source_account, channel, sender, message_id, message_type, source_cursor, " +
	"raw_content, importance_score, confidence_score, status, reasoning, " +
	"user_rating, user_feedback, vector_id, scoring_model, examples_used, created_at, updated_at, resolved_at"

// SQLiteMessageRepository implements repository.MessageRepository using SQLite.
type SQLiteMessageRepository struct {
	db                   *sql.DB
	maxMessagesPerSource int
}

// NewSQLiteMessageRepository opens (or creates) a SQLite database at dbPath,
// enables WAL mode, creates the messages table if needed, and returns the repository.
// maxMessagesPerSource controls the FIFO eviction threshold per source and must be > 0.
func NewSQLiteMessageRepository(dbPath string, maxMessagesPerSource int) (*SQLiteMessageRepository, error) {
	if maxMessagesPerSource <= 0 {
		return nil, fmt.Errorf("maxMessagesPerSource must be greater than 0, got %d", maxMessagesPerSource)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createMessagesTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create messages table: %w", err)
	}

	// Migration: add message_type column (idempotent).
	_, err = db.Exec("ALTER TABLE messages ADD COLUMN message_type TEXT NOT NULL DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		_ = db.Close()
		return nil, fmt.Errorf("migrate message_type column: %w", err)
	}

	// Migration: add source_cursor column (idempotent).
	_, err = db.Exec("ALTER TABLE messages ADD COLUMN source_cursor TEXT NOT NULL DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		_ = db.Close()
		return nil, fmt.Errorf("migrate source_cursor column: %w", err)
	}

	// Migration: add scoring_model column (idempotent).
	_, err = db.Exec("ALTER TABLE messages ADD COLUMN scoring_model TEXT NOT NULL DEFAULT ''")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		_ = db.Close()
		return nil, fmt.Errorf("migrate scoring_model column: %w", err)
	}

	// Migration: add examples_used column (idempotent).
	_, err = db.Exec("ALTER TABLE messages ADD COLUMN examples_used INTEGER NOT NULL DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		_ = db.Close()
		return nil, fmt.Errorf("migrate examples_used column: %w", err)
	}

	return &SQLiteMessageRepository{db: db, maxMessagesPerSource: maxMessagesPerSource}, nil
}

// DB returns the underlying *sql.DB connection.
func (r *SQLiteMessageRepository) DB() *sql.DB {
	return r.db
}

// Insert inserts a message into the database. If a message with the same MessageID
// already exists, it updates the existing row (upsert). Before inserting, it enforces
// FIFO eviction: if the source already has >= maxMessagesPerSource messages, the oldest is deleted.
func (r *SQLiteMessageRepository) Insert(ctx context.Context, msg *repository.Message) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := r.evictOldestIfNeeded(ctx, tx, msg.Source); err != nil {
		return err
	}

	// Upsert the message.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (
			id, source, source_account, channel, sender, message_id, message_type, source_cursor,
			raw_content, importance_score, confidence_score, status, reasoning,
			user_rating, user_feedback, vector_id, scoring_model, examples_used,
			created_at, updated_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			id = excluded.id,
			source = excluded.source,
			source_account = excluded.source_account,
			channel = excluded.channel,
			sender = excluded.sender,
			message_type = excluded.message_type,
			source_cursor = excluded.source_cursor,
			raw_content = excluded.raw_content,
			importance_score = excluded.importance_score,
			confidence_score = excluded.confidence_score,
			status = excluded.status,
			reasoning = excluded.reasoning,
			user_rating = excluded.user_rating,
			user_feedback = excluded.user_feedback,
			vector_id = excluded.vector_id,
			scoring_model = excluded.scoring_model,
			examples_used = excluded.examples_used,
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
		msg.MessageType,
		msg.SourceCursor,
		msg.RawContent,
		msg.ImportanceScore,
		msg.ConfidenceScore,
		msg.Status,
		msg.Reasoning,
		nullable(msg.UserRating),
		nullable(msg.UserFeedback),
		nullableUUID(msg.VectorID),
		msg.ScoringModel,
		msg.ExamplesUsed,
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
			message_type = ?,
			source_cursor = ?,
			raw_content = ?,
			importance_score = ?,
			confidence_score = ?,
			status = ?,
			reasoning = ?,
			user_rating = ?,
			user_feedback = ?,
			vector_id = ?,
			scoring_model = ?,
			examples_used = ?,
			updated_at = ?,
			resolved_at = ?
		WHERE id = ?
	`,
		msg.Source,
		msg.SourceAccount,
		msg.Channel,
		msg.Sender,
		msg.MessageID,
		msg.MessageType,
		msg.SourceCursor,
		msg.RawContent,
		msg.ImportanceScore,
		msg.ConfidenceScore,
		msg.Status,
		msg.Reasoning,
		nullable(msg.UserRating),
		nullable(msg.UserFeedback),
		nullableUUID(msg.VectorID),
		msg.ScoringModel,
		msg.ExamplesUsed,
		msg.UpdatedAt.Format(time.RFC3339),
		nullableTime(msg.ResolvedAt),
		msg.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	return nil
}

// QueryByID returns a single message by its UUID, or repository.ErrNotFound if not found.
func (r *SQLiteMessageRepository) QueryByID(ctx context.Context, id uuid.UUID) (*repository.Message, error) {
	rows, err := r.db.QueryContext(ctx, querySelectByID, id.String())
	if err != nil {
		return nil, fmt.Errorf("query by id: %w", err)
	}
	defer rows.Close()

	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, repository.ErrNotFound
	}
	return msgs[0], nil
}

// QueryByStatus returns all messages with the given status.
func (r *SQLiteMessageRepository) QueryByStatus(ctx context.Context, status string) ([]*repository.Message, error) {
	filter := repository.MessageFilter{
		Status: status,
	}
	msg, _, err := r.QueryFiltered(ctx, filter)
	return msg, err
}

// QueryAll returns all messages in the database.
func (r *SQLiteMessageRepository) QueryAll(ctx context.Context) ([]*repository.Message, error) {
	rows, err := r.db.QueryContext(ctx, querySelectAll)
	if err != nil {
		return nil, fmt.Errorf("query all: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// QueryOldestToNewest returns up to limit messages ordered by created_at ascending.
func (r *SQLiteMessageRepository) QueryOldestToNewest(ctx context.Context, limit int) ([]*repository.Message, error) {
	rows, err := r.db.QueryContext(ctx, querySelectOldestLimit, limit)
	if err != nil {
		return nil, fmt.Errorf("query oldest to newest: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// CountBySource returns the number of messages for the given source.
func (r *SQLiteMessageRepository) CountBySource(ctx context.Context, source string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, queryCountBySource, source).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by source: %w", err)
	}
	return count, nil
}

// ExistsByMessageID checks whether a message with the given source-native message ID exists.
func (r *SQLiteMessageRepository) ExistsByMessageID(ctx context.Context, messageID string) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx, "SELECT 1 FROM messages WHERE message_id = ? LIMIT 1", messageID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("exists by message id: %w", err)
	}
	return true, nil
}

// MaxSourceCursor returns the maximum source_cursor value for a given source + sourceAccount + channel.
// Returns empty string if no matching records exist.
func (r *SQLiteMessageRepository) MaxSourceCursor(ctx context.Context, source, sourceAccount, channel string) (string, error) {
	var cursor sql.NullString
	err := r.db.QueryRowContext(ctx,
		"SELECT MAX(source_cursor) FROM messages WHERE source = ? AND source_account = ? AND channel = ?",
		source, sourceAccount, channel,
	).Scan(&cursor)
	if err != nil {
		return "", fmt.Errorf("max source cursor: %w", err)
	}
	if !cursor.Valid {
		return "", nil
	}
	return cursor.String, nil
}

// DistinctChannels returns distinct channel names for a given source and sourceAccount.
func (r *SQLiteMessageRepository) DistinctChannels(ctx context.Context, source, sourceAccount string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT DISTINCT channel FROM messages WHERE source = ? AND source_account = ?",
		source, sourceAccount,
	)
	if err != nil {
		return nil, fmt.Errorf("distinct channels: %w", err)
	}
	defer rows.Close()

	var channels []string
	for rows.Next() {
		var ch string
		if err := rows.Scan(&ch); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate channels: %w", err)
	}
	return channels, nil
}

// QueryFiltered returns messages matching the given filter criteria, plus the total count
// of matching messages (before limit/offset) for pagination.
func (r *SQLiteMessageRepository) QueryFiltered(ctx context.Context, filter repository.MessageFilter) ([]*repository.Message, int, error) {
	where := "WHERE 1=1"
	var args []any

	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Source != "" {
		where += " AND source = ?"
		args = append(args, filter.Source)
	}
	if filter.Channel != "" {
		where += " AND channel = ?"
		args = append(args, filter.Channel)
	}
	if filter.Since != nil {
		where += " AND created_at > ?"
		args = append(args, filter.Since.Format(time.RFC3339))
	}

	// Count total matching rows (before limit/offset).
	var total int
	countQuery := "SELECT COUNT(*) FROM messages " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("query filtered count: %w", err)
	}

	// Clamp limit.
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	selectQuery := "SELECT " + messageColumnsStr + " FROM messages " + where +
		" ORDER BY created_at DESC LIMIT ? OFFSET ?"
	selectArgs := make([]any, len(args), len(args)+2)
	copy(selectArgs, args)
	selectArgs = append(selectArgs, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query filtered: %w", err)
	}
	defer rows.Close()

	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, 0, err
	}

	return msgs, total, nil
}

// evictOldestIfNeeded performs FIFO eviction for the given source if at capacity.
func (r *SQLiteMessageRepository) evictOldestIfNeeded(ctx context.Context, tx *sql.Tx, source string) error {
	var count int
	err := tx.QueryRowContext(ctx, queryCountBySource, source).Scan(&count)
	if err != nil {
		return fmt.Errorf("count messages by source: %w", err)
	}

	if count >= r.maxMessagesPerSource {
		_, err = tx.ExecContext(ctx, queryDeleteOldestBySource, source)
		if err != nil {
			return fmt.Errorf("evict oldest message: %w", err)
		}
	}
	return nil
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
		scoringModel string
		examplesUsed int
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
		&msg.MessageType,
		&msg.SourceCursor,
		&msg.RawContent,
		&msg.ImportanceScore,
		&msg.ConfidenceScore,
		&msg.Status,
		&msg.Reasoning,
		&userRating,
		&userFeedback,
		&vectorIDStr,
		&scoringModel,
		&examplesUsed,
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

	msg.ScoringModel = scoringModel
	msg.ExamplesUsed = examplesUsed

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

// nullable converts a pointer to a value suitable for SQL (nil becomes NULL).
// For non-string types, it returns the dereferenced value directly.
// For special types, it applies the appropriate converter function.
func nullable[T any](v *T) any {
	if v == nil {
		return nil
	}
	return *v
}

// nullableUUID converts *uuid.UUID to a value suitable for SQL (nil becomes NULL).
func nullableUUID(v *uuid.UUID) any {
	if v == nil {
		return nil
	}
	return v.String()
}

// nullableTime converts *time.Time to a value suitable for SQL (nil becomes NULL).
func nullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.Format(time.RFC3339)
}
