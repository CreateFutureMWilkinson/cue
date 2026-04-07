package repository

import (
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

// Validate checks that all fields of the routing rule are valid.
func (r *RoutingRule) Validate() error {
	return ErrNotImplemented
}
