package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const createSlackAccountsTable = `
CREATE TABLE IF NOT EXISTS slack_accounts (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL,
    token TEXT NOT NULL,
    workspace_id TEXT NOT NULL UNIQUE,
    poll_interval_seconds INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

const createEmailAccountsTable = `
CREATE TABLE IF NOT EXISTS email_accounts (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL,
    imap_host TEXT NOT NULL,
    imap_port INTEGER NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password_env TEXT NOT NULL,
    poll_interval_seconds INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

const slackAccountColumns = "id, enabled, token, workspace_id, poll_interval_seconds, created_at, updated_at"
const emailAccountColumns = "id, enabled, imap_host, imap_port, username, password_env, poll_interval_seconds, created_at, updated_at"

// SQLiteServiceConfigRepository implements repository.ServiceConfigRepository using SQLite.
type SQLiteServiceConfigRepository struct {
	db *sql.DB
}

// NewSQLiteServiceConfigRepository creates a new ServiceConfigRepository backed by SQLite.
// It creates the slack_accounts and email_accounts tables if they do not exist.
func NewSQLiteServiceConfigRepository(db *sql.DB) (*SQLiteServiceConfigRepository, error) {
	if _, err := db.Exec(createSlackAccountsTable); err != nil {
		return nil, fmt.Errorf("create slack_accounts table: %w", err)
	}

	if _, err := db.Exec(createEmailAccountsTable); err != nil {
		return nil, fmt.Errorf("create email_accounts table: %w", err)
	}

	// Migrate existing databases: rename bot_token → token.
	// Silently ignored if the column was already named token (new databases).
	_, _ = db.Exec("ALTER TABLE slack_accounts RENAME COLUMN bot_token TO token")

	return &SQLiteServiceConfigRepository{db: db}, nil
}

// --- Slack Account Methods ---

// UpsertSlackAccount inserts or updates a Slack account by primary key.
// On conflict with the same ID, all fields except created_at are updated.
func (r *SQLiteServiceConfigRepository) UpsertSlackAccount(ctx context.Context, acct *repository.SlackAccount) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO slack_accounts (id, enabled, token, workspace_id, poll_interval_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			token = excluded.token,
			workspace_id = excluded.workspace_id,
			poll_interval_seconds = excluded.poll_interval_seconds,
			updated_at = excluded.updated_at
	`,
		acct.ID.String(),
		boolToInt(acct.Enabled),
		acct.Token,
		acct.WorkspaceID,
		acct.PollIntervalSeconds,
		acct.CreatedAt.Format(time.RFC3339),
		acct.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upsert slack account: %w", err)
	}
	return nil
}

// GetSlackAccount retrieves a Slack account by ID. Returns ErrNotFound if not found.
func (r *SQLiteServiceConfigRepository) GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+slackAccountColumns+`
		FROM slack_accounts WHERE id = ?
	`, id.String())

	acct, err := scanSlackAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("slack account %s: %w", id, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get slack account: %w", err)
	}

	return acct, nil
}

// DeleteSlackAccount deletes a Slack account by ID. Returns nil if not found (idempotent).
func (r *SQLiteServiceConfigRepository) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM slack_accounts WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("delete slack account: %w", err)
	}
	return nil
}

// ListSlackAccounts returns all Slack accounts. Returns an empty non-nil slice if none exist.
func (r *SQLiteServiceConfigRepository) ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+slackAccountColumns+`
		FROM slack_accounts
	`)
	if err != nil {
		return nil, fmt.Errorf("list slack accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*repository.SlackAccount, 0)
	for rows.Next() {
		acct, err := scanSlackAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan slack account: %w", err)
		}
		accounts = append(accounts, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate slack accounts: %w", err)
	}

	return accounts, nil
}

// --- Email Account Methods ---

// UpsertEmailAccount inserts or updates an email account by primary key.
// On conflict with the same ID, all fields except created_at are updated.
func (r *SQLiteServiceConfigRepository) UpsertEmailAccount(ctx context.Context, acct *repository.EmailAccount) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO email_accounts (id, enabled, imap_host, imap_port, username, password_env, poll_interval_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			imap_host = excluded.imap_host,
			imap_port = excluded.imap_port,
			username = excluded.username,
			password_env = excluded.password_env,
			poll_interval_seconds = excluded.poll_interval_seconds,
			updated_at = excluded.updated_at
	`,
		acct.ID.String(),
		boolToInt(acct.Enabled),
		acct.IMAPHost,
		acct.IMAPPort,
		acct.Username,
		acct.PasswordEnv,
		acct.PollIntervalSeconds,
		acct.CreatedAt.Format(time.RFC3339),
		acct.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upsert email account: %w", err)
	}
	return nil
}

// GetEmailAccount retrieves an email account by ID. Returns ErrNotFound if not found.
func (r *SQLiteServiceConfigRepository) GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+emailAccountColumns+`
		FROM email_accounts WHERE id = ?
	`, id.String())

	acct, err := scanEmailAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("email account %s: %w", id, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get email account: %w", err)
	}

	return acct, nil
}

// DeleteEmailAccount deletes an email account by ID. Returns nil if not found (idempotent).
func (r *SQLiteServiceConfigRepository) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM email_accounts WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("delete email account: %w", err)
	}
	return nil
}

// ListEmailAccounts returns all email accounts. Returns an empty non-nil slice if none exist.
func (r *SQLiteServiceConfigRepository) ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+emailAccountColumns+`
		FROM email_accounts
	`)
	if err != nil {
		return nil, fmt.Errorf("list email accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*repository.EmailAccount, 0)
	for rows.Next() {
		acct, err := scanEmailAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan email account: %w", err)
		}
		accounts = append(accounts, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate email accounts: %w", err)
	}

	return accounts, nil
}

// boolToInt converts a bool to an integer (0 or 1) for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// scanSlackAccount scans a database row into a SlackAccount struct.
// It can be used with both QueryRow and Rows.Scan.
func scanSlackAccount(scanner interface {
	Scan(dest ...any) error
}) (*repository.SlackAccount, error) {
	var (
		acct         repository.SlackAccount
		idStr        string
		enabled      int
		createdAtStr string
		updatedAtStr string
	)

	err := scanner.Scan(&idStr, &enabled, &acct.Token, &acct.WorkspaceID, &acct.PollIntervalSeconds, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, err
	}

	acct.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse slack account ID: %w", err)
	}

	acct.Enabled = enabled != 0

	acct.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	acct.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &acct, nil
}

// scanEmailAccount scans a database row into an EmailAccount struct.
// It can be used with both QueryRow and Rows.Scan.
func scanEmailAccount(scanner interface {
	Scan(dest ...any) error
}) (*repository.EmailAccount, error) {
	var (
		acct         repository.EmailAccount
		idStr        string
		enabled      int
		createdAtStr string
		updatedAtStr string
	)

	err := scanner.Scan(&idStr, &enabled, &acct.IMAPHost, &acct.IMAPPort, &acct.Username, &acct.PasswordEnv, &acct.PollIntervalSeconds, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, err
	}

	acct.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse email account ID: %w", err)
	}

	acct.Enabled = enabled != 0

	acct.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	acct.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &acct, nil
}
