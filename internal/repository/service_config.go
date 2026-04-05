package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SlackAccount represents a configured Slack workspace for monitoring.
type SlackAccount struct {
	ID                  uuid.UUID
	Enabled             bool
	BotToken            string
	WorkspaceID         string
	PollIntervalSeconds int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// EmailAccount represents a configured email account for monitoring.
type EmailAccount struct {
	ID                  uuid.UUID
	Enabled             bool
	IMAPHost            string
	IMAPPort            int
	Username            string
	PasswordEnv         string
	PollIntervalSeconds int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ServiceConfigRepository defines the contract for service configuration persistence.
// Get methods return ErrNotFound for unknown IDs. Delete is idempotent (no-op for unknown IDs).
// Upsert inserts or updates based on primary key.
type ServiceConfigRepository interface {
	// Slack accounts
	ListSlackAccounts(ctx context.Context) ([]*SlackAccount, error)
	GetSlackAccount(ctx context.Context, id uuid.UUID) (*SlackAccount, error)
	UpsertSlackAccount(ctx context.Context, acct *SlackAccount) error
	DeleteSlackAccount(ctx context.Context, id uuid.UUID) error

	// Email accounts
	ListEmailAccounts(ctx context.Context) ([]*EmailAccount, error)
	GetEmailAccount(ctx context.Context, id uuid.UUID) (*EmailAccount, error)
	UpsertEmailAccount(ctx context.Context, acct *EmailAccount) error
	DeleteEmailAccount(ctx context.Context, id uuid.UUID) error
}
