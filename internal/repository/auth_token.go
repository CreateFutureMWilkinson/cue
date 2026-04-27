package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuthToken represents a hashed API token for TOFU pairing authentication.
type AuthToken struct {
	ID        uuid.UUID
	Label     string
	TokenHash string // SHA-256 hex digest
	CreatedAt time.Time
	LastSeen  time.Time
	Revoked   bool
}

// AuthTokenRepository defines the contract for auth token persistence.
// LookupByHash returns ErrNotFound for unknown hashes. DeleteAll is idempotent.
type AuthTokenRepository interface {
	Create(ctx context.Context, token *AuthToken) error
	LookupByHash(ctx context.Context, hash string) (*AuthToken, error)
	List(ctx context.Context) ([]AuthToken, error)
	UpdateLabel(ctx context.Context, id uuid.UUID, label string) error
	Revoke(ctx context.Context, id uuid.UUID) error
	CountActive(ctx context.Context) (int, error)
	UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error
	DeleteAll(ctx context.Context) error
}
