package sqlite

import (
	"context"
	"database/sql"
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

func (r *SQLiteAuthTokenRepository) Create(_ context.Context, _ *repository.AuthToken) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) LookupByHash(_ context.Context, _ string) (*repository.AuthToken, error) {
	return nil, repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) List(_ context.Context) ([]repository.AuthToken, error) {
	return nil, repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) UpdateLabel(_ context.Context, _ uuid.UUID, _ string) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) Revoke(_ context.Context, _ uuid.UUID) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) CountActive(_ context.Context) (int, error) {
	return 0, repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) UpdateLastSeen(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteAuthTokenRepository) DeleteAll(_ context.Context) error {
	return repository.ErrNotImplemented
}
