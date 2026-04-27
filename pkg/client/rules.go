package client

import (
	"context"

	"github.com/google/uuid"
)

// rulesPath is the base URL path for the routing-rules REST resource.
const rulesPath = "/api/v1/rules"

// RoutingRule mirrors the server's ruleItem DTO returned by
// /api/v1/rules routes. SourceAccount is a pointer because the server
// uses json:",omitempty" and may emit null/absent for global rules
// that do not target a specific source account.
type RoutingRule struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Priority       int       `json:"priority"`
	SourceType     string    `json:"source_type"`
	SourceAccount  *string   `json:"source_account,omitempty"`
	ChannelPattern string    `json:"channel_pattern"`
	ContentPattern string    `json:"content_pattern"`
	MessageType    string    `json:"message_type"`
	Action         string    `json:"action"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

// CreateRuleRequest is the POST body for creating a rule via
// POST /api/v1/rules. The server populates id, priority (default),
// enabled=true, created_at, and updated_at automatically.
type CreateRuleRequest struct {
	Name           string  `json:"name"`
	SourceType     string  `json:"source_type"`
	SourceAccount  *string `json:"source_account,omitempty"`
	ChannelPattern string  `json:"channel_pattern"`
	ContentPattern string  `json:"content_pattern"`
	MessageType    string  `json:"message_type"`
	Action         string  `json:"action"`
}

// UpdateRuleRequest is the PUT body for full replacement via
// PUT /api/v1/rules/{id}. Unlike CreateRuleRequest, it carries
// Priority and Enabled because the caller is replacing all fields.
type UpdateRuleRequest struct {
	Name           string  `json:"name"`
	Priority       int     `json:"priority"`
	SourceType     string  `json:"source_type"`
	SourceAccount  *string `json:"source_account,omitempty"`
	ChannelPattern string  `json:"channel_pattern"`
	ContentPattern string  `json:"content_pattern"`
	MessageType    string  `json:"message_type"`
	Action         string  `json:"action"`
	Enabled        bool    `json:"enabled"`
}

// PatchRuleRequest is the PATCH body for partial updates to priority
// and/or enabled. Both fields are optional pointers; nil fields are
// omitted from the outgoing JSON via omitempty so the server only
// applies fields the caller explicitly set.
type PatchRuleRequest struct {
	Priority *int  `json:"priority,omitempty"`
	Enabled  *bool `json:"enabled,omitempty"`
}

// RuleFilter captures the optional query parameters accepted by
// GET /api/v1/rules. Empty strings are omitted from the outgoing
// query string. SourceAccount should be a UUID string or "".
type RuleFilter struct {
	SourceType    string
	SourceAccount string
}

// RulesClient wraps /api/v1/rules routes: listing, creating, fetching,
// updating (full PUT), patching (partial priority/enabled), and deleting
// routing rules.
type RulesClient interface {
	ListRules(ctx context.Context, filter RuleFilter) ([]RoutingRule, error)
	GetRule(ctx context.Context, id uuid.UUID) (*RoutingRule, error)
	CreateRule(ctx context.Context, req CreateRuleRequest) (*RoutingRule, error)
	UpdateRule(ctx context.Context, id uuid.UUID, req UpdateRuleRequest) (*RoutingRule, error)
	PatchRule(ctx context.Context, id uuid.UUID, patch PatchRuleRequest) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
}

// rulesAdapter is the concrete RulesClient backed by an *APIClient.
type rulesAdapter struct {
	client *APIClient
}

// NewRulesClient returns a RulesClient backed by the given APIClient.
func NewRulesClient(c *APIClient) RulesClient {
	return &rulesAdapter{client: c}
}

// ListRules is a noop stub. Replaced in the GREEN phase.
func (a *rulesAdapter) ListRules(ctx context.Context, filter RuleFilter) ([]RoutingRule, error) {
	return nil, ErrNotImplemented
}

// GetRule is a noop stub. Replaced in the GREEN phase.
func (a *rulesAdapter) GetRule(ctx context.Context, id uuid.UUID) (*RoutingRule, error) {
	return nil, ErrNotImplemented
}

// CreateRule is a noop stub. Replaced in the GREEN phase.
func (a *rulesAdapter) CreateRule(ctx context.Context, req CreateRuleRequest) (*RoutingRule, error) {
	return nil, ErrNotImplemented
}

// UpdateRule is a noop stub. Replaced in the GREEN phase.
func (a *rulesAdapter) UpdateRule(ctx context.Context, id uuid.UUID, req UpdateRuleRequest) (*RoutingRule, error) {
	return nil, ErrNotImplemented
}

// PatchRule is a noop stub. Replaced in the GREEN phase.
func (a *rulesAdapter) PatchRule(ctx context.Context, id uuid.UUID, patch PatchRuleRequest) error {
	return ErrNotImplemented
}

// DeleteRule is a noop stub. Replaced in the GREEN phase.
func (a *rulesAdapter) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return ErrNotImplemented
}
