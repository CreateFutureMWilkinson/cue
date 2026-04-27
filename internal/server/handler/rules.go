package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// ErrNotImplemented is returned by stub handlers that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

// RulesManager is the subset of routing-rule operations needed by rule handlers.
type RulesManager interface {
	ListRules(ctx context.Context) ([]*repository.RoutingRule, error)
	ListRulesBySourceType(ctx context.Context, sourceType string) ([]*repository.RoutingRule, error)
	ListRulesBySourceAccount(ctx context.Context, accountID uuid.UUID) ([]*repository.RoutingRule, error)
	GetRule(ctx context.Context, id uuid.UUID) (*repository.RoutingRule, error)
	SaveRule(ctx context.Context, rule *repository.RoutingRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
	ReorderRule(ctx context.Context, id uuid.UUID, newPriority int) error
	ToggleRule(ctx context.Context, id uuid.UUID, enabled bool) error
}

// --- JSON response types ---

type ruleItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Priority       int     `json:"priority"`
	SourceType     string  `json:"source_type"`
	SourceAccount  *string `json:"source_account,omitempty"`
	ChannelPattern string  `json:"channel_pattern"`
	ContentPattern string  `json:"content_pattern"`
	MessageType    string  `json:"message_type"`
	Action         string  `json:"action"`
	Enabled        bool    `json:"enabled"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// --- JSON request types ---

type createRuleRequest struct {
	Name           string  `json:"name"`
	SourceType     string  `json:"source_type"`
	SourceAccount  *string `json:"source_account,omitempty"`
	ChannelPattern string  `json:"channel_pattern"`
	ContentPattern string  `json:"content_pattern"`
	MessageType    string  `json:"message_type"`
	Action         string  `json:"action"`
}

type patchRuleRequest struct {
	Priority *int  `json:"priority,omitempty"`
	Enabled  *bool `json:"enabled,omitempty"`
}

// --- Helpers ---

// parseRuleID extracts and parses the {id} path parameter as a UUID.
func parseRuleID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}

// --- Handlers (stubs) ---

// ListRulesHandler returns an http.HandlerFunc for GET /api/v1/rules.
// Supports ?source_type= and ?source_account= query parameters.
func ListRulesHandler(_ RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// GetRuleHandler returns an http.HandlerFunc for GET /api/v1/rules/{id}.
func GetRuleHandler(_ RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// CreateRuleHandler returns an http.HandlerFunc for POST /api/v1/rules.
func CreateRuleHandler(_ RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// UpdateRuleHandler returns an http.HandlerFunc for PUT /api/v1/rules/{id}.
func UpdateRuleHandler(_ RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// PatchRuleHandler returns an http.HandlerFunc for PATCH /api/v1/rules/{id}.
func PatchRuleHandler(_ RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// DeleteRuleHandler returns an http.HandlerFunc for DELETE /api/v1/rules/{id}.
func DeleteRuleHandler(_ RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}
