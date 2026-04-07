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

// Valid fields per source.
var validFields = map[string]map[string]bool{
	"email": {"sender": true, "subject": true},
	"slack": {"sender": true, "channel": true, "content": true, "message_type": true},
}

// Validate checks that all fields of the routing rule are valid.
func (r *RoutingRule) Validate() error {
	fields, ok := validFields[r.Source]
	if !ok {
		return fmt.Errorf("invalid source %q: %w", r.Source, ErrInvalidRoutingRule)
	}

	if !fields[r.Field] {
		return fmt.Errorf("invalid field %q for source %q: %w", r.Field, r.Source, ErrInvalidRoutingRule)
	}

	if r.Action != "notified" && r.Action != "ignored" {
		return fmt.Errorf("invalid action %q: %w", r.Action, ErrInvalidRoutingRule)
	}

	if r.Priority < 0 {
		return fmt.Errorf("negative priority %d: %w", r.Priority, ErrInvalidRoutingRule)
	}

	if _, err := regexp.Compile(r.Pattern); err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", r.Pattern, ErrInvalidRoutingRule)
	}

	return nil
}
