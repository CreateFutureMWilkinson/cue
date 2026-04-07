package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/secret"
)

const createSlackAccountsTable = `
CREATE TABLE IF NOT EXISTS slack_accounts (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL,
    token_encrypted BLOB NOT NULL,
    workspace_id TEXT NOT NULL UNIQUE,
    poll_interval_seconds INTEGER NOT NULL,
    friendly_name TEXT NOT NULL DEFAULT '',
    web_url TEXT NOT NULL DEFAULT '',
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
    password_encrypted BLOB NOT NULL,
    poll_interval_seconds INTEGER NOT NULL,
    friendly_name TEXT NOT NULL DEFAULT '',
    web_url TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

const createCalendarAccountsTable = `
CREATE TABLE IF NOT EXISTS calendar_accounts (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL,
    name TEXT NOT NULL UNIQUE,
    ics_url_encrypted BLOB NOT NULL,
    poll_interval_seconds INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

const (
	slackAccountColumns    = "id, enabled, token_encrypted, workspace_id, username, poll_interval_seconds, friendly_name, web_url, created_at, updated_at"
	emailAccountColumns    = "id, enabled, imap_host, imap_port, username, password_encrypted, encryption, poll_interval_seconds, friendly_name, web_url, created_at, updated_at"
	calendarAccountColumns = "id, enabled, name, ics_url_encrypted, poll_interval_seconds, created_at, updated_at"
)

// SQLiteServiceConfigRepository implements repository.ServiceConfigRepository using SQLite.
type SQLiteServiceConfigRepository struct {
	db  *sql.DB
	enc secret.Encryptor
}

// NewSQLiteServiceConfigRepository creates a new ServiceConfigRepository backed by SQLite.
// It creates the slack_accounts, email_accounts, and calendar_accounts tables if they do not exist.
func NewSQLiteServiceConfigRepository(db *sql.DB, enc secret.Encryptor) (*SQLiteServiceConfigRepository, error) {
	if _, err := db.Exec(createSlackAccountsTable); err != nil {
		return nil, fmt.Errorf("creating slack_accounts table: %w", err)
	}

	if _, err := db.Exec(createEmailAccountsTable); err != nil {
		return nil, fmt.Errorf("creating email_accounts table: %w", err)
	}

	if _, err := db.Exec(createCalendarAccountsTable); err != nil {
		return nil, fmt.Errorf("creating calendar_accounts table: %w", err)
	}

	// Migrate existing databases: rename old column names to new encrypted column names.
	_, _ = db.Exec("ALTER TABLE slack_accounts RENAME COLUMN bot_token TO token_encrypted")
	_, _ = db.Exec("ALTER TABLE slack_accounts RENAME COLUMN token TO token_encrypted")
	_, _ = db.Exec("ALTER TABLE email_accounts RENAME COLUMN password_env TO password_encrypted")
	_, _ = db.Exec("ALTER TABLE email_accounts ADD COLUMN encryption TEXT NOT NULL DEFAULT 'ssl_tls'")
	_, _ = db.Exec("ALTER TABLE slack_accounts ADD COLUMN username TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE slack_accounts ADD COLUMN friendly_name TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE slack_accounts ADD COLUMN web_url TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE email_accounts ADD COLUMN friendly_name TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE email_accounts ADD COLUMN web_url TEXT NOT NULL DEFAULT ''")

	return &SQLiteServiceConfigRepository{db: db, enc: enc}, nil
}

// --- Slack Account Methods ---

// UpsertSlackAccount inserts or updates a Slack account by primary key.
// On conflict with the same ID, all fields except created_at are updated.
func (r *SQLiteServiceConfigRepository) UpsertSlackAccount(ctx context.Context, acct *repository.SlackAccount) error {
	encToken, err := r.enc.Encrypt([]byte(acct.Token))
	if err != nil {
		return fmt.Errorf("encrypting Slack token: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO slack_accounts (id, enabled, token_encrypted, workspace_id, username, poll_interval_seconds, friendly_name, web_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			token_encrypted = excluded.token_encrypted,
			workspace_id = excluded.workspace_id,
			username = excluded.username,
			poll_interval_seconds = excluded.poll_interval_seconds,
			friendly_name = excluded.friendly_name,
			web_url = excluded.web_url,
			updated_at = excluded.updated_at
	`,
		acct.ID.String(),
		boolToInt(acct.Enabled),
		encToken,
		acct.WorkspaceID,
		acct.Username,
		acct.PollIntervalSeconds,
		acct.FriendlyName,
		acct.WebURL,
		acct.CreatedAt.Format(time.RFC3339),
		acct.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upserting Slack account: %w", err)
	}
	return nil
}

// GetSlackAccount retrieves a Slack account by ID. Returns ErrNotFound if not found.
func (r *SQLiteServiceConfigRepository) GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+slackAccountColumns+`
		FROM slack_accounts WHERE id = ?
	`, id.String())

	acct, err := r.scanSlackAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("slack account %s: %w", id, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("getting Slack account: %w", err)
	}

	return acct, nil
}

// DeleteSlackAccount deletes a Slack account by ID. Returns nil if not found (idempotent).
func (r *SQLiteServiceConfigRepository) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM slack_accounts WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("deleting Slack account: %w", err)
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
		return nil, fmt.Errorf("listing Slack accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*repository.SlackAccount, 0)
	for rows.Next() {
		acct, err := r.scanSlackAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning Slack account: %w", err)
		}
		accounts = append(accounts, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating Slack accounts: %w", err)
	}

	return accounts, nil
}

// --- Email Account Methods ---

// UpsertEmailAccount inserts or updates an email account by primary key.
// On conflict with the same ID, all fields except created_at are updated.
func (r *SQLiteServiceConfigRepository) UpsertEmailAccount(ctx context.Context, acct *repository.EmailAccount) error {
	encPassword, err := r.enc.Encrypt([]byte(acct.Password))
	if err != nil {
		return fmt.Errorf("encrypting email password: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO email_accounts (id, enabled, imap_host, imap_port, username, password_encrypted, encryption, poll_interval_seconds, friendly_name, web_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			imap_host = excluded.imap_host,
			imap_port = excluded.imap_port,
			username = excluded.username,
			password_encrypted = excluded.password_encrypted,
			encryption = excluded.encryption,
			poll_interval_seconds = excluded.poll_interval_seconds,
			friendly_name = excluded.friendly_name,
			web_url = excluded.web_url,
			updated_at = excluded.updated_at
	`,
		acct.ID.String(),
		boolToInt(acct.Enabled),
		acct.IMAPHost,
		acct.IMAPPort,
		acct.Username,
		encPassword,
		acct.Encryption,
		acct.PollIntervalSeconds,
		acct.FriendlyName,
		acct.WebURL,
		acct.CreatedAt.Format(time.RFC3339),
		acct.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upserting email account: %w", err)
	}
	return nil
}

// GetEmailAccount retrieves an email account by ID. Returns ErrNotFound if not found.
func (r *SQLiteServiceConfigRepository) GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+emailAccountColumns+`
		FROM email_accounts WHERE id = ?
	`, id.String())

	acct, err := r.scanEmailAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("email account %s: %w", id, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("getting email account: %w", err)
	}

	return acct, nil
}

// DeleteEmailAccount deletes an email account by ID. Returns nil if not found (idempotent).
func (r *SQLiteServiceConfigRepository) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM email_accounts WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("deleting email account: %w", err)
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
		return nil, fmt.Errorf("listing email accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*repository.EmailAccount, 0)
	for rows.Next() {
		acct, err := r.scanEmailAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning email account: %w", err)
		}
		accounts = append(accounts, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating email accounts: %w", err)
	}

	return accounts, nil
}

// --- Calendar Account Methods ---

// UpsertCalendarAccount inserts or updates a calendar account by primary key.
func (r *SQLiteServiceConfigRepository) UpsertCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) error {
	encICSURL, err := r.enc.Encrypt([]byte(acct.ICSURL))
	if err != nil {
		return fmt.Errorf("encrypting calendar ICS URL: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO calendar_accounts (id, enabled, name, ics_url_encrypted, poll_interval_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			name = excluded.name,
			ics_url_encrypted = excluded.ics_url_encrypted,
			poll_interval_seconds = excluded.poll_interval_seconds,
			updated_at = excluded.updated_at
	`,
		acct.ID.String(),
		boolToInt(acct.Enabled),
		acct.Name,
		encICSURL,
		acct.PollIntervalSeconds,
		acct.CreatedAt.Format(time.RFC3339),
		acct.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upserting calendar account: %w", err)
	}
	return nil
}

// GetCalendarAccount retrieves a calendar account by ID. Returns ErrNotFound if not found.
func (r *SQLiteServiceConfigRepository) GetCalendarAccount(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+calendarAccountColumns+`
		FROM calendar_accounts WHERE id = ?
	`, id.String())

	acct, err := r.scanCalendarAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("calendar account %s: %w", id, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("getting calendar account: %w", err)
	}

	return acct, nil
}

// DeleteCalendarAccount deletes a calendar account by ID. Returns nil if not found (idempotent).
func (r *SQLiteServiceConfigRepository) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM calendar_accounts WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("deleting calendar account: %w", err)
	}
	return nil
}

// ListCalendarAccounts returns all calendar accounts. Returns an empty non-nil slice if none exist.
func (r *SQLiteServiceConfigRepository) ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+calendarAccountColumns+`
		FROM calendar_accounts
	`)
	if err != nil {
		return nil, fmt.Errorf("listing calendar accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*repository.CalendarAccount, 0)
	for rows.Next() {
		acct, err := r.scanCalendarAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning calendar account: %w", err)
		}
		accounts = append(accounts, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating calendar accounts: %w", err)
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

// scanSlackAccount scans a database row into a SlackAccount struct, decrypting the token.
func (r *SQLiteServiceConfigRepository) scanSlackAccount(scanner interface {
	Scan(dest ...any) error
}) (*repository.SlackAccount, error) {
	var (
		acct         repository.SlackAccount
		idStr        string
		enabled      int
		tokenEnc     []byte
		createdAtStr string
		updatedAtStr string
	)

	err := scanner.Scan(&idStr, &enabled, &tokenEnc, &acct.WorkspaceID, &acct.Username, &acct.PollIntervalSeconds, &acct.FriendlyName, &acct.WebURL, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, err
	}

	tokenBytes, err := r.enc.Decrypt(tokenEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypting Slack token: %w", err)
	}
	acct.Token = string(tokenBytes)

	acct.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parsing Slack account ID: %w", err)
	}

	acct.Enabled = enabled != 0

	acct.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at timestamp: %w", err)
	}

	acct.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at timestamp: %w", err)
	}

	return &acct, nil
}

// scanEmailAccount scans a database row into an EmailAccount struct, decrypting the password.
func (r *SQLiteServiceConfigRepository) scanEmailAccount(scanner interface {
	Scan(dest ...any) error
}) (*repository.EmailAccount, error) {
	var (
		acct         repository.EmailAccount
		idStr        string
		enabled      int
		passwordEnc  []byte
		createdAtStr string
		updatedAtStr string
	)

	err := scanner.Scan(&idStr, &enabled, &acct.IMAPHost, &acct.IMAPPort, &acct.Username, &passwordEnc, &acct.Encryption, &acct.PollIntervalSeconds, &acct.FriendlyName, &acct.WebURL, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, err
	}

	passwordBytes, err := r.enc.Decrypt(passwordEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypting email password: %w", err)
	}
	acct.Password = string(passwordBytes)

	acct.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parsing email account ID: %w", err)
	}

	acct.Enabled = enabled != 0

	acct.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at timestamp: %w", err)
	}

	acct.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at timestamp: %w", err)
	}

	return &acct, nil
}

// scanCalendarAccount scans a database row into a CalendarAccount struct, decrypting the ICS URL.
func (r *SQLiteServiceConfigRepository) scanCalendarAccount(scanner interface {
	Scan(dest ...any) error
}) (*repository.CalendarAccount, error) {
	var (
		acct         repository.CalendarAccount
		idStr        string
		enabled      int
		icsURLEnc    []byte
		createdAtStr string
		updatedAtStr string
	)

	err := scanner.Scan(&idStr, &enabled, &acct.Name, &icsURLEnc, &acct.PollIntervalSeconds, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, err
	}

	icsURLBytes, err := r.enc.Decrypt(icsURLEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypting calendar ICS URL: %w", err)
	}
	acct.ICSURL = string(icsURLBytes)

	acct.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parsing calendar account ID: %w", err)
	}

	acct.Enabled = enabled != 0

	acct.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at timestamp: %w", err)
	}

	acct.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at timestamp: %w", err)
	}

	return &acct, nil
}
