package repository

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// RoutingRule represents a user-defined deterministic routing rule.
// Rules use multi-pattern matching: source_type + optional source_account,
// channel_pattern, content_pattern, and message_type filters. All set
// fields must match (AND logic).
type RoutingRule struct {
	ID             uuid.UUID
	Name           string     // optional, user-friendly label
	Priority       int        // evaluation order (lower = higher priority)
	SourceType     string     // required: "slack" or "email"
	SourceAccount  *uuid.UUID // optional FK → service config account
	ChannelPattern string     // optional regex for channel matching
	ContentPattern string     // optional regex for content matching
	MessageType    string     // optional pre-filter (e.g., "channel_join")
	Action         string     // "notified" or "ignored"
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RoutingRuleRepository defines the contract for routing rule persistence.
// Get returns ErrNotFound for unknown IDs. Delete is idempotent (no-op for unknown IDs).
// List methods return rules sorted by priority ascending.
type RoutingRuleRepository interface {
	ListRules(ctx context.Context) ([]*RoutingRule, error)
	ListRulesBySourceType(ctx context.Context, sourceType string) ([]*RoutingRule, error)
	ListRulesBySourceAccount(ctx context.Context, accountID uuid.UUID) ([]*RoutingRule, error)
	GetRule(ctx context.Context, id uuid.UUID) (*RoutingRule, error)
	UpsertRule(ctx context.Context, rule *RoutingRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
}

// Validate checks that all fields of the routing rule are valid.
func (r *RoutingRule) Validate() error {
	// Validate source type
	if r.SourceType != "slack" && r.SourceType != "email" {
		return fmt.Errorf("invalid source type %q: %w", r.SourceType, ErrInvalidRoutingRule)
	}

	// Validate action
	if r.Action != "notified" && r.Action != "ignored" {
		return fmt.Errorf("invalid action %q: %w", r.Action, ErrInvalidRoutingRule)
	}

	// Validate priority
	if r.Priority < 0 {
		return fmt.Errorf("negative priority %d: %w", r.Priority, ErrInvalidRoutingRule)
	}

	// Validate channel pattern regex
	if r.ChannelPattern != "" {
		if _, err := regexp.Compile(r.ChannelPattern); err != nil {
			return fmt.Errorf("invalid channel pattern %q: %w", r.ChannelPattern, ErrInvalidRoutingRule)
		}
	}

	// Validate content pattern regex
	if r.ContentPattern != "" {
		if _, err := regexp.Compile(r.ContentPattern); err != nil {
			return fmt.Errorf("invalid content pattern %q: %w", r.ContentPattern, ErrInvalidRoutingRule)
		}
	}

	return nil
}
