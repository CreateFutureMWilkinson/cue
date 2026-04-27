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

const createRoutingRulesTableSQL = `
CREATE TABLE IF NOT EXISTS routing_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    priority INTEGER NOT NULL,
    source_type TEXT NOT NULL,
    source_account TEXT,
    channel_pattern TEXT NOT NULL DEFAULT '',
    content_pattern TEXT NOT NULL DEFAULT '',
    message_type TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_routing_rules_priority ON routing_rules(priority);
CREATE INDEX IF NOT EXISTS idx_routing_rules_source_type ON routing_rules(source_type);
`

const (
	routingRuleColumns = "id, name, priority, source_type, source_account, channel_pattern, content_pattern, message_type, action, enabled, created_at, updated_at"
)

const upsertRoutingRuleSQL = `
INSERT INTO routing_rules (id, name, priority, source_type, source_account, channel_pattern, content_pattern, message_type, action, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    priority = excluded.priority,
    source_type = excluded.source_type,
    source_account = excluded.source_account,
    channel_pattern = excluded.channel_pattern,
    content_pattern = excluded.content_pattern,
    message_type = excluded.message_type,
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

	// Seed default rules if the table is empty.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM routing_rules").Scan(&count); err != nil {
		return nil, fmt.Errorf("counting routing rules: %w", err)
	}
	if count == 0 {
		now := time.Now().UTC().Format(time.RFC3339)
		defaultRules := []struct {
			name           string
			priority       int
			sourceType     string
			channelPattern string
			contentPattern string
			messageType    string
			action         string
		}{
			{name: "Channel Join", priority: 0, sourceType: "slack", messageType: "channel_join", action: "notified"},
			{name: "@mention", priority: 1, sourceType: "slack", contentPattern: "@username", action: "notified"},
		}
		for _, r := range defaultRules {
			_, err := db.Exec(upsertRoutingRuleSQL,
				uuid.New().String(),
				r.name,
				r.priority,
				r.sourceType,
				nil, // source_account
				r.channelPattern,
				r.contentPattern,
				r.messageType,
				r.action,
				boolToInt(true), // enabled
				now,
				now,
			)
			if err != nil {
				return nil, fmt.Errorf("seeding default routing rule (priority %d): %w", r.priority, err)
			}
		}
	}

	return &SQLiteRoutingRuleRepository{db: db}, nil
}

func (r *SQLiteRoutingRuleRepository) ListRules(ctx context.Context) ([]*repository.RoutingRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+routingRuleColumns+`
		 FROM routing_rules ORDER BY priority ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing routing rules: %w", err)
	}
	defer rows.Close()

	rules := make([]*repository.RoutingRule, 0)
	for rows.Next() {
		rule, err := r.scanRoutingRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning routing rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating routing rules: %w", err)
	}
	return rules, nil
}

func (r *SQLiteRoutingRuleRepository) ListRulesBySourceType(ctx context.Context, sourceType string) ([]*repository.RoutingRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+routingRuleColumns+`
		 FROM routing_rules WHERE source_type = ? ORDER BY priority ASC`, sourceType)
	if err != nil {
		return nil, fmt.Errorf("listing routing rules by source type: %w", err)
	}
	defer rows.Close()

	rules := make([]*repository.RoutingRule, 0)
	for rows.Next() {
		rule, err := r.scanRoutingRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning routing rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating routing rules: %w", err)
	}
	return rules, nil
}

func (r *SQLiteRoutingRuleRepository) ListRulesBySourceAccount(ctx context.Context, accountID uuid.UUID) ([]*repository.RoutingRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+routingRuleColumns+`
		 FROM routing_rules WHERE source_account = ? ORDER BY priority ASC`, accountID.String())
	if err != nil {
		return nil, fmt.Errorf("listing routing rules by source account: %w", err)
	}
	defer rows.Close()

	rules := make([]*repository.RoutingRule, 0)
	for rows.Next() {
		rule, err := r.scanRoutingRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning routing rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating routing rules: %w", err)
	}
	return rules, nil
}

// GetRule retrieves a routing rule by ID. Returns ErrNotFound if not found.
func (r *SQLiteRoutingRuleRepository) GetRule(ctx context.Context, id uuid.UUID) (*repository.RoutingRule, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+routingRuleColumns+`
		 FROM routing_rules WHERE id = ?`, id.String())

	rule, err := r.scanRoutingRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("routing rule %s: %w", id, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("getting routing rule: %w", err)
	}
	return rule, nil
}

func (r *SQLiteRoutingRuleRepository) scanRoutingRule(scanner interface {
	Scan(dest ...any) error
}) (*repository.RoutingRule, error) {
	var (
		rule             repository.RoutingRule
		idStr            string
		sourceAccountStr sql.NullString
		enabled          int
		createdAtStr     string
		updatedAtStr     string
	)

	err := scanner.Scan(
		&idStr,
		&rule.Name,
		&rule.Priority,
		&rule.SourceType,
		&sourceAccountStr,
		&rule.ChannelPattern,
		&rule.ContentPattern,
		&rule.MessageType,
		&rule.Action,
		&enabled,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	rule.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parsing routing rule ID: %w", err)
	}

	if sourceAccountStr.Valid {
		parsed, err := uuid.Parse(sourceAccountStr.String)
		if err != nil {
			return nil, fmt.Errorf("parsing source_account UUID: %w", err)
		}
		rule.SourceAccount = &parsed
	}

	rule.Enabled = enabled != 0

	rule.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}

	rule.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}

	return &rule, nil
}

// UpsertRule validates the rule and inserts or updates it in the database.
func (r *SQLiteRoutingRuleRepository) UpsertRule(ctx context.Context, rule *repository.RoutingRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}

	var sourceAccount any
	if rule.SourceAccount != nil {
		sourceAccount = rule.SourceAccount.String()
	}

	_, err := r.db.ExecContext(ctx, upsertRoutingRuleSQL,
		rule.ID.String(),
		rule.Name,
		rule.Priority,
		rule.SourceType,
		sourceAccount,
		rule.ChannelPattern,
		rule.ContentPattern,
		rule.MessageType,
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
	_, err := r.db.ExecContext(ctx, `DELETE FROM routing_rules WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("deleting routing rule: %w", err)
	}
	return nil
}
