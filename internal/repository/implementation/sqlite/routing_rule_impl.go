package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const createRoutingRulesTableSQL = `
CREATE TABLE IF NOT EXISTS routing_rules (
    id TEXT PRIMARY KEY,
    priority INTEGER NOT NULL,
    source TEXT NOT NULL,
    field TEXT NOT NULL,
    negate INTEGER NOT NULL DEFAULT 0,
    pattern TEXT NOT NULL,
    action TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_routing_rules_priority ON routing_rules(priority);
CREATE INDEX IF NOT EXISTS idx_routing_rules_source ON routing_rules(source);
`

const upsertRoutingRuleSQL = `
INSERT INTO routing_rules (id, priority, source, field, negate, pattern, action, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    priority = excluded.priority,
    source = excluded.source,
    field = excluded.field,
    negate = excluded.negate,
    pattern = excluded.pattern,
    action = excluded.action,
    enabled = excluded.enabled,
    updated_at = excluded.updated_at
`

// Compile-time check that SQLiteRoutingRuleRepository satisfies RoutingRuleRepository.
var _ repository.RoutingRuleRepository = (*SQLiteRoutingRuleRepository)(nil)

// SQLiteRoutingRuleRepository implements repository.RoutingRuleRepository using SQLite.
type SQLiteRoutingRuleRepository struct {
	db *sql.DB
}

// NewSQLiteRoutingRuleRepository creates a new RoutingRuleRepository backed by SQLite.
// It creates the routing_rules table if it does not exist.
func NewSQLiteRoutingRuleRepository(db *sql.DB) (*SQLiteRoutingRuleRepository, error) {
	if _, err := db.Exec(createRoutingRulesTableSQL); err != nil {
		return nil, fmt.Errorf("creating routing_rules table: %w", err)
	}
	return &SQLiteRoutingRuleRepository{db: db}, nil
}

func (r *SQLiteRoutingRuleRepository) ListRules(ctx context.Context) ([]*repository.RoutingRule, error) {
	return nil, repository.ErrNotImplemented
}

func (r *SQLiteRoutingRuleRepository) ListRulesBySource(ctx context.Context, source string) ([]*repository.RoutingRule, error) {
	return nil, repository.ErrNotImplemented
}

func (r *SQLiteRoutingRuleRepository) GetRule(ctx context.Context, id uuid.UUID) (*repository.RoutingRule, error) {
	return nil, repository.ErrNotImplemented
}

// UpsertRule validates the rule and inserts or updates it in the database.
func (r *SQLiteRoutingRuleRepository) UpsertRule(ctx context.Context, rule *repository.RoutingRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, upsertRoutingRuleSQL,
		rule.ID.String(),
		rule.Priority,
		rule.Source,
		rule.Field,
		boolToInt(rule.Negate),
		rule.Pattern,
		rule.Action,
		boolToInt(rule.Enabled),
		rule.CreatedAt.Format(time.RFC3339),
		rule.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upserting routing rule: %w", err)
	}
	return nil
}

func (r *SQLiteRoutingRuleRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return repository.ErrNotImplemented
}
