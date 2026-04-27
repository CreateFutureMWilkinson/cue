package client

import (
	"context"
	"net/http"
	"net/url"

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

// ListRules issues GET /api/v1/rules with optional source_type and
// source_account query parameters encoded from filter. Empty fields
// are omitted from the outgoing query string.
func (a *rulesAdapter) ListRules(ctx context.Context, filter RuleFilter) ([]RoutingRule, error) {
	q := url.Values{}
	if filter.SourceType != "" {
		q.Set("source_type", filter.SourceType)
	}
	if filter.SourceAccount != "" {
		q.Set("source_account", filter.SourceAccount)
	}

	path := buildPath(rulesPath, q)

	var out struct {
		Rules []RoutingRule `json:"rules"`
		Count int           `json:"count"`
	}
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Rules, nil
}

// GetRule issues GET /api/v1/rules/{id} and decodes the ruleItem payload.
func (a *rulesAdapter) GetRule(ctx context.Context, id uuid.UUID) (*RoutingRule, error) {
	var rule RoutingRule
	if err := a.client.doJSON(ctx, http.MethodGet, rulesPath+"/"+id.String(), nil, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// CreateRule issues POST /api/v1/rules with req as the JSON body and
// decodes the server-populated ruleItem response.
func (a *rulesAdapter) CreateRule(ctx context.Context, req CreateRuleRequest) (*RoutingRule, error) {
	var rule RoutingRule
	if err := a.client.doJSON(ctx, http.MethodPost, rulesPath, req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateRule issues PUT /api/v1/rules/{id} with req as the full
// replacement JSON body and decodes the updated ruleItem response.
func (a *rulesAdapter) UpdateRule(ctx context.Context, id uuid.UUID, req UpdateRuleRequest) (*RoutingRule, error) {
	var rule RoutingRule
	if err := a.client.doJSON(ctx, http.MethodPut, rulesPath+"/"+id.String(), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// PatchRule issues PATCH /api/v1/rules/{id} with patch as the partial
// JSON body. The server returns 204 No Content on success. Nil fields
// on PatchRuleRequest are omitted from the body via omitempty.
func (a *rulesAdapter) PatchRule(ctx context.Context, id uuid.UUID, patch PatchRuleRequest) error {
	return a.client.doJSON(ctx, http.MethodPatch, rulesPath+"/"+id.String(), patch, nil)
}

// DeleteRule issues DELETE /api/v1/rules/{id}. The server returns 204
// No Content on success.
func (a *rulesAdapter) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return a.client.doJSON(ctx, http.MethodDelete, rulesPath+"/"+id.String(), nil, nil)
}
