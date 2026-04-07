package sqlite

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// Compile-time check that SQLiteRoutingRuleRepository satisfies RoutingRuleRepository.
var _ repository.RoutingRuleRepository = (*SQLiteRoutingRuleRepository)(nil)

// SQLiteRoutingRuleRepository implements repository.RoutingRuleRepository using SQLite.
type SQLiteRoutingRuleRepository struct {
	db *sql.DB
}

// NewSQLiteRoutingRuleRepository creates a new RoutingRuleRepository backed by SQLite.
// It creates the routing_rules table if it does not exist.
func NewSQLiteRoutingRuleRepository(db *sql.DB) (*SQLiteRoutingRuleRepository, error) {
	return nil, repository.ErrNotImplemented
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

func (r *SQLiteRoutingRuleRepository) UpsertRule(ctx context.Context, rule *repository.RoutingRule) error {
	return repository.ErrNotImplemented
}

func (r *SQLiteRoutingRuleRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return repository.ErrNotImplemented
}
