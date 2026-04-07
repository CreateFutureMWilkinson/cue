package repository

import (
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
