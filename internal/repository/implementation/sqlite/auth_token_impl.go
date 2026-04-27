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

const createAuthTokensTableSQL = `
CREATE TABLE IF NOT EXISTS auth_tokens (
    id          TEXT PRIMARY KEY,
    label       TEXT NOT NULL DEFAULT '',
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TEXT NOT NULL,
    last_seen   TEXT NOT NULL,
    revoked     INTEGER NOT NULL DEFAULT 0
);
`

// Compile-time check that SQLiteAuthTokenRepository satisfies AuthTokenRepository.
var _ repository.AuthTokenRepository = (*SQLiteAuthTokenRepository)(nil)

// SQLiteAuthTokenRepository implements repository.AuthTokenRepository using SQLite.
type SQLiteAuthTokenRepository struct {
	db *sql.DB
}

// NewSQLiteAuthTokenRepository creates a new AuthTokenRepository backed by SQLite.
// It creates the auth_tokens table if it does not exist.
func NewSQLiteAuthTokenRepository(db *sql.DB) (*SQLiteAuthTokenRepository, error) {
	if _, err := db.Exec(createAuthTokensTableSQL); err != nil {
		return nil, fmt.Errorf("creating auth_tokens table: %w", err)
	}
	return &SQLiteAuthTokenRepository{db: db}, nil
}

func (r *SQLiteAuthTokenRepository) Create(ctx context.Context, token *repository.AuthToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO auth_tokens (id, label, token_hash, created_at, last_seen, revoked)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		token.ID.String(),
		token.Label,
		token.TokenHash,
		token.CreatedAt.Format(time.RFC3339),
		token.LastSeen.Format(time.RFC3339),
		boolToInt(token.Revoked),
	)
	if err != nil {
		return fmt.Errorf("inserting auth token: %w", err)
	}
	return nil
}

func (r *SQLiteAuthTokenRepository) LookupByHash(ctx context.Context, hash string) (*repository.AuthToken, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, label, token_hash, created_at, last_seen, revoked
		 FROM auth_tokens WHERE token_hash = ?`, hash)

	token, err := scanAuthToken(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("auth token with hash: %w", repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("looking up auth token by hash: %w", err)
	}
	return token, nil
}

func (r *SQLiteAuthTokenRepository) List(ctx context.Context) ([]repository.AuthToken, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, label, token_hash, created_at, last_seen, revoked
		 FROM auth_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing auth tokens: %w", err)
	}
	defer rows.Close()

	var tokens []repository.AuthToken
	for rows.Next() {
		token, err := scanAuthToken(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning auth token: %w", err)
		}
		tokens = append(tokens, *token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating auth tokens: %w", err)
	}
	return tokens, nil
}

func (r *SQLiteAuthTokenRepository) UpdateLabel(ctx context.Context, id uuid.UUID, label string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_tokens SET label = ? WHERE id = ?`,
		label, id.String())
	if err != nil {
		return fmt.Errorf("updating auth token label: %w", err)
	}
	return nil
}

func (r *SQLiteAuthTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_tokens SET revoked = 1 WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("revoking auth token: %w", err)
	}
	return nil
}

func (r *SQLiteAuthTokenRepository) CountActive(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM auth_tokens WHERE revoked = 0`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting active auth tokens: %w", err)
	}
	return count, nil
}

func (r *SQLiteAuthTokenRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_tokens SET last_seen = ? WHERE id = ?`,
		t.Format(time.RFC3339), id.String())
	if err != nil {
		return fmt.Errorf("updating auth token last_seen: %w", err)
	}
	return nil
}

func (r *SQLiteAuthTokenRepository) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auth_tokens`)
	if err != nil {
		return fmt.Errorf("deleting all auth tokens: %w", err)
	}
	return nil
}

func scanAuthToken(scanner interface{ Scan(dest ...any) error }) (*repository.AuthToken, error) {
	var (
		token        repository.AuthToken
		idStr        string
		createdAtStr string
		lastSeenStr  string
		revoked      int
	)

	err := scanner.Scan(
		&idStr,
		&token.Label,
		&token.TokenHash,
		&createdAtStr,
		&lastSeenStr,
		&revoked,
	)
	if err != nil {
		return nil, err
	}

	token.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parsing auth token ID: %w", err)
	}

	token.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing auth token created_at: %w", err)
	}

	token.LastSeen, err = time.Parse(time.RFC3339, lastSeenStr)
	if err != nil {
		return nil, fmt.Errorf("parsing auth token last_seen: %w", err)
	}

	token.Revoked = revoked != 0

	return &token, nil
}
