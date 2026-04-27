package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

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

// ruleToItem converts a repository.RoutingRule to the JSON response type.
func ruleToItem(r *repository.RoutingRule) ruleItem {
	item := ruleItem{
		ID:             r.ID.String(),
		Name:           r.Name,
		Priority:       r.Priority,
		SourceType:     r.SourceType,
		ChannelPattern: r.ChannelPattern,
		ContentPattern: r.ContentPattern,
		MessageType:    r.MessageType,
		Action:         r.Action,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      r.UpdatedAt.Format(time.RFC3339),
	}
	if r.SourceAccount != nil {
		s := r.SourceAccount.String()
		item.SourceAccount = &s
	}
	return item
}

// writeRuleError writes the appropriate HTTP error for a rules operation error.
func writeRuleError(w http.ResponseWriter, err error) {
	if isNotFound(err) {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, repository.ErrInvalidRoutingRule) {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSONError(w, http.StatusInternalServerError, err.Error())
}

// --- Handlers ---

// ListRulesHandler returns an http.HandlerFunc for GET /api/v1/rules.
// Supports ?source_type= and ?source_account= query parameters.
//
// @Summary      List routing rules
// @Description  Returns rules in priority order. Optional filters: source_type
// @Description  (slack/email) or source_account (UUID). source_account takes
// @Description  precedence when both are supplied.
// @Tags         rules
// @Produce      json
// @Param        source_type     query     string  false  "slack | email"
// @Param        source_account  query     string  false  "Account UUID"
// @Success      200             {object}  map[string]any
// @Failure      400             {object}  map[string]string
// @Failure      500             {object}  map[string]string
// @Router       /api/v1/rules [get]
func ListRulesHandler(mgr RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var (
			rules []*repository.RoutingRule
			err   error
		)

		if sa := r.URL.Query().Get("source_account"); sa != "" {
			acctID, parseErr := uuid.Parse(sa)
			if parseErr != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid source_account UUID")
				return
			}
			rules, err = mgr.ListRulesBySourceAccount(ctx, acctID)
		} else if st := r.URL.Query().Get("source_type"); st != "" {
			rules, err = mgr.ListRulesBySourceType(ctx, st)
		} else {
			rules, err = mgr.ListRules(ctx)
		}

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		items := make([]ruleItem, 0, len(rules))
		for _, rule := range rules {
			items = append(items, ruleToItem(rule))
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"rules": items,
			"count": len(items),
		})
	}
}

// GetRuleHandler returns an http.HandlerFunc for GET /api/v1/rules/{id}.
//
// @Summary      Get routing rule
// @Tags         rules
// @Produce      json
// @Param        id   path      string  true  "Rule UUID"
// @Success      200  {object}  handler.ruleItem
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/rules/{id} [get]
func GetRuleHandler(mgr RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseRuleID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid rule ID")
			return
		}

		rule, err := mgr.GetRule(r.Context(), id)
		if err != nil {
			writeRuleError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, ruleToItem(rule))
	}
}

// CreateRuleHandler returns an http.HandlerFunc for POST /api/v1/rules.
//
// @Summary      Create routing rule
// @Description  Adds a new rule to the deterministic-rules pipeline. Patterns
// @Description  use Go regexp syntax. The action determines routing: NOTIFIED,
// @Description  BUFFERED, or IGNORED.
// @Tags         rules
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createRuleRequest  true  "Rule fields"
// @Success      201      {object}  handler.ruleItem
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/rules [post]
func CreateRuleHandler(mgr RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		now := time.Now().UTC()
		rule := &repository.RoutingRule{
			ID:             uuid.New(),
			Name:           req.Name,
			SourceType:     req.SourceType,
			ChannelPattern: req.ChannelPattern,
			ContentPattern: req.ContentPattern,
			MessageType:    req.MessageType,
			Action:         req.Action,
			Enabled:        true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if req.SourceAccount != nil {
			acctID, err := uuid.Parse(*req.SourceAccount)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid source_account UUID")
				return
			}
			rule.SourceAccount = &acctID
		}

		if err := mgr.SaveRule(r.Context(), rule); err != nil {
			writeRuleError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, ruleToItem(rule))
	}
}

// UpdateRuleHandler returns an http.HandlerFunc for PUT /api/v1/rules/{id}.
//
// @Summary      Replace routing rule
// @Description  Full replacement of a rule's content fields. Use PATCH to
// @Description  toggle enabled or change priority.
// @Tags         rules
// @Accept       json
// @Produce      json
// @Param        id       path      string                     true  "Rule UUID"
// @Param        request  body      handler.createRuleRequest  true  "Rule fields"
// @Success      200      {object}  handler.ruleItem
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/rules/{id} [put]
func UpdateRuleHandler(mgr RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseRuleID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid rule ID")
			return
		}

		var req createRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		now := time.Now().UTC()
		rule := &repository.RoutingRule{
			ID:             id,
			Name:           req.Name,
			SourceType:     req.SourceType,
			ChannelPattern: req.ChannelPattern,
			ContentPattern: req.ContentPattern,
			MessageType:    req.MessageType,
			Action:         req.Action,
			Enabled:        true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if req.SourceAccount != nil {
			acctID, parseErr := uuid.Parse(*req.SourceAccount)
			if parseErr != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid source_account UUID")
				return
			}
			rule.SourceAccount = &acctID
		}

		if err := mgr.SaveRule(r.Context(), rule); err != nil {
			writeRuleError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, ruleToItem(rule))
	}
}

// PatchRuleHandler returns an http.HandlerFunc for PATCH /api/v1/rules/{id}.
//
// @Summary      Patch routing rule (priority / enabled)
// @Description  Reorders the rule (sets priority) and/or toggles its enabled
// @Description  flag. At least one of priority or enabled must be present;
// @Description  400 if the body has neither.
// @Tags         rules
// @Accept       json
// @Param        id       path  string                     true  "Rule UUID"
// @Param        request  body  handler.patchRuleRequest   true  "Fields to patch"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/rules/{id} [patch]
func PatchRuleHandler(mgr RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseRuleID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid rule ID")
			return
		}

		var req patchRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if req.Priority == nil && req.Enabled == nil {
			writeJSONError(w, http.StatusBadRequest, "no fields to patch")
			return
		}

		ctx := r.Context()

		if req.Priority != nil {
			if err := mgr.ReorderRule(ctx, id, *req.Priority); err != nil {
				writeRuleError(w, err)
				return
			}
		}

		if req.Enabled != nil {
			if err := mgr.ToggleRule(ctx, id, *req.Enabled); err != nil {
				writeRuleError(w, err)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteRuleHandler returns an http.HandlerFunc for DELETE /api/v1/rules/{id}.
//
// @Summary      Delete routing rule
// @Tags         rules
// @Param        id   path  string  true  "Rule UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/rules/{id} [delete]
func DeleteRuleHandler(mgr RulesManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseRuleID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid rule ID")
			return
		}

		if err := mgr.DeleteRule(r.Context(), id); err != nil {
			writeRuleError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
