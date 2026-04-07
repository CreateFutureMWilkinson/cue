package repository

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// RoutingRule represents a user-defined deterministic routing rule.
type RoutingRule struct {
	ID        uuid.UUID
	Priority  int
	Source    string // "email" or "slack"
	Field     string // Field to match against (source-dependent)
	Negate    bool   // true = "not matches", false = "matches"
	Pattern   string // Go regexp pattern
	Action    string // "notified" or "ignored"
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RoutingRuleRepository defines the contract for routing rule persistence.
// Get returns ErrNotFound for unknown IDs. Delete is idempotent (no-op for unknown IDs).
// List methods return rules sorted by priority ascending.
type RoutingRuleRepository interface {
	ListRules(ctx context.Context) ([]*RoutingRule, error)
	ListRulesBySource(ctx context.Context, source string) ([]*RoutingRule, error)
	GetRule(ctx context.Context, id uuid.UUID) (*RoutingRule, error)
	UpsertRule(ctx context.Context, rule *RoutingRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
}

// validSourceFields maps each source to its set of valid fields.
var validSourceFields = map[string]map[string]bool{
	"email": {"sender": true, "subject": true},
	"slack": {"sender": true, "channel": true, "content": true, "message_type": true},
}

// Validate checks that all fields of the routing rule are valid.
func (r *RoutingRule) Validate() error {
	// Validate source and field combination
	fields, ok := validSourceFields[r.Source]
	if !ok {
		return fmt.Errorf("invalid source %q: %w", r.Source, ErrInvalidRoutingRule)
	}

	if !fields[r.Field] {
		return fmt.Errorf("invalid field %q for source %q: %w", r.Field, r.Source, ErrInvalidRoutingRule)
	}

	// Validate action
	if r.Action != "notified" && r.Action != "ignored" {
		return fmt.Errorf("invalid action %q: %w", r.Action, ErrInvalidRoutingRule)
	}

	// Validate priority
	if r.Priority < 0 {
		return fmt.Errorf("negative priority %d: %w", r.Priority, ErrInvalidRoutingRule)
	}

	// Validate regex pattern
	if _, err := regexp.Compile(r.Pattern); err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", r.Pattern, ErrInvalidRoutingRule)
	}

	return nil
}
