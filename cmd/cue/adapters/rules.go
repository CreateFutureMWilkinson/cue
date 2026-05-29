package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// RulesAdapter satisfies repository.RoutingRuleRepository on top of
// the SDK's RulesClient. It maps the wire RoutingRule DTO (with
// stringly-typed source_account, RFC3339 timestamps) onto the
// repository struct (uuid pointer, time.Time).
type RulesAdapter struct {
	client client.RulesClient
}

// NewRulesAdapter wraps the given SDK rules client.
func NewRulesAdapter(c client.RulesClient) *RulesAdapter {
	return &RulesAdapter{client: c}
}

// ListRules returns every routing rule the server reports.
func (a *RulesAdapter) ListRules(ctx context.Context) ([]*repository.RoutingRule, error) {
	dtos, err := a.client.ListRules(ctx, client.RuleFilter{})
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	out := make([]*repository.RoutingRule, 0, len(dtos))
	for i := range dtos {
		r, err := ruleDTOToRepo(dtos[i])
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// ListRulesBySourceType filters by source on the server side.
func (a *RulesAdapter) ListRulesBySourceType(ctx context.Context, sourceType string) ([]*repository.RoutingRule, error) {
	dtos, err := a.client.ListRules(ctx, client.RuleFilter{SourceType: sourceType})
	if err != nil {
		return nil, fmt.Errorf("list rules by source_type=%s: %w", sourceType, err)
	}
	return convertRules(dtos)
}

// ListRulesBySourceAccount filters by account on the server side.
func (a *RulesAdapter) ListRulesBySourceAccount(ctx context.Context, accountID uuid.UUID) ([]*repository.RoutingRule, error) {
	dtos, err := a.client.ListRules(ctx, client.RuleFilter{SourceAccount: accountID.String()})
	if err != nil {
		return nil, fmt.Errorf("list rules by source_account=%s: %w", accountID, err)
	}
	return convertRules(dtos)
}

// GetRule fetches a single rule by ID.
func (a *RulesAdapter) GetRule(ctx context.Context, id uuid.UUID) (*repository.RoutingRule, error) {
	dto, err := a.client.GetRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get rule %s: %w", id, err)
	}
	return ruleDTOToRepo(*dto)
}

// UpsertRule creates a new rule when ID is unset and otherwise replaces
// the rule with a full PUT. The server stamps timestamps and a fresh
// ID on Create; the adapter copies those back onto the supplied rule
// pointer so callers see the persisted shape.
func (a *RulesAdapter) UpsertRule(ctx context.Context, rule *repository.RoutingRule) error {
	if rule == nil {
		return fmt.Errorf("rules adapter: cannot upsert nil rule")
	}
	if rule.ID == uuid.Nil {
		req := client.CreateRuleRequest{
			Name:           rule.Name,
			SourceType:     rule.SourceType,
			SourceAccount:  uuidPtrToString(rule.SourceAccount),
			ChannelPattern: rule.ChannelPattern,
			ContentPattern: rule.ContentPattern,
			MessageType:    rule.MessageType,
			Action:         rule.Action,
		}
		dto, err := a.client.CreateRule(ctx, req)
		if err != nil {
			return fmt.Errorf("create rule: %w", err)
		}
		converted, err := ruleDTOToRepo(*dto)
		if err != nil {
			return err
		}
		*rule = *converted
		return nil
	}
	req := client.UpdateRuleRequest{
		Name:           rule.Name,
		Priority:       rule.Priority,
		SourceType:     rule.SourceType,
		SourceAccount:  uuidPtrToString(rule.SourceAccount),
		ChannelPattern: rule.ChannelPattern,
		ContentPattern: rule.ContentPattern,
		MessageType:    rule.MessageType,
		Action:         rule.Action,
		Enabled:        rule.Enabled,
	}
	dto, err := a.client.UpdateRule(ctx, rule.ID, req)
	if err != nil {
		return fmt.Errorf("update rule %s: %w", rule.ID, err)
	}
	converted, err := ruleDTOToRepo(*dto)
	if err != nil {
		return err
	}
	*rule = *converted
	return nil
}

// DeleteRule removes a routing rule by ID. The server treats unknown
// IDs as a no-op so the adapter follows that contract.
func (a *RulesAdapter) DeleteRule(ctx context.Context, id uuid.UUID) error {
	if err := a.client.DeleteRule(ctx, id); err != nil {
		return fmt.Errorf("delete rule %s: %w", id, err)
	}
	return nil
}

func convertRules(dtos []client.RoutingRule) ([]*repository.RoutingRule, error) {
	out := make([]*repository.RoutingRule, 0, len(dtos))
	for i := range dtos {
		r, err := ruleDTOToRepo(dtos[i])
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func ruleDTOToRepo(d client.RoutingRule) (*repository.RoutingRule, error) {
	r := &repository.RoutingRule{
		ID:             d.ID,
		Name:           d.Name,
		Priority:       d.Priority,
		SourceType:     d.SourceType,
		ChannelPattern: d.ChannelPattern,
		ContentPattern: d.ContentPattern,
		MessageType:    d.MessageType,
		Action:         d.Action,
		Enabled:        d.Enabled,
		CreatedAt:      parseRFC3339OrZero(d.CreatedAt),
		UpdatedAt:      parseRFC3339OrZero(d.UpdatedAt),
	}
	if d.SourceAccount != nil && *d.SourceAccount != "" {
		id, err := uuid.Parse(*d.SourceAccount)
		if err != nil {
			return nil, fmt.Errorf("parse source_account uuid %q: %w", *d.SourceAccount, err)
		}
		r.SourceAccount = &id
	}
	return r, nil
}

func uuidPtrToString(p *uuid.UUID) *string {
	if p == nil || *p == uuid.Nil {
		return nil
	}
	s := p.String()
	return &s
}

// QueueDepthAdapter satisfies the slice of repository.QueueRepository
// the rules presenter actually consumes — namely PendingCount, used to
// drive the queue-depth warning. Other QueueRepository methods are
// orchestrator-only (server-side) and are intentionally unsupported on
// the client.
//
// PendingCount is implemented by listing messages with Status="Pending"
// and returning the server's reported total.
type QueueDepthAdapter struct {
	client client.MessageClient
}

// NewQueueDepthAdapter wraps the SDK MessageClient.
func NewQueueDepthAdapter(c client.MessageClient) *QueueDepthAdapter {
	return &QueueDepthAdapter{client: c}
}

// Enqueue is server-side only; the client never enqueues.
func (q *QueueDepthAdapter) Enqueue(_ context.Context, _ uuid.UUID) error {
	return errClientQueueWriteUnsupported("Enqueue")
}

// DequeueOldest is server-side only.
func (q *QueueDepthAdapter) DequeueOldest(_ context.Context) (*repository.QueueEntry, error) {
	return nil, errClientQueueWriteUnsupported("DequeueOldest")
}

// MarkDone is server-side only.
func (q *QueueDepthAdapter) MarkDone(_ context.Context, _ uuid.UUID) error {
	return errClientQueueWriteUnsupported("MarkDone")
}

// MarkFailed is server-side only.
func (q *QueueDepthAdapter) MarkFailed(_ context.Context, _ uuid.UUID) error {
	return errClientQueueWriteUnsupported("MarkFailed")
}

// PendingCount returns the number of messages with Status="Pending".
// The server's list endpoint reports the matching total in its
// response envelope so the adapter does not need to walk the result
// slice itself.
func (q *QueueDepthAdapter) PendingCount(ctx context.Context) (int, error) {
	_, total, err := q.client.ListMessages(ctx, client.MessageFilter{Status: "Pending", Limit: 1})
	if err != nil {
		return 0, fmt.Errorf("list pending messages: %w", err)
	}
	return total, nil
}

// PurgeCompleted is server-side only.
func (q *QueueDepthAdapter) PurgeCompleted(_ context.Context) error {
	return errClientQueueWriteUnsupported("PurgeCompleted")
}

// PurgeOlderThan is server-side only.
func (q *QueueDepthAdapter) PurgeOlderThan(_ context.Context, _ time.Time) error {
	return errClientQueueWriteUnsupported("PurgeOlderThan")
}

// PurgeAll is server-side only.
func (q *QueueDepthAdapter) PurgeAll(_ context.Context) error {
	return errClientQueueWriteUnsupported("PurgeAll")
}

// ResetProcessing is server-side only.
func (q *QueueDepthAdapter) ResetProcessing(_ context.Context) (int64, error) {
	return 0, errClientQueueWriteUnsupported("ResetProcessing")
}

func errClientQueueWriteUnsupported(method string) error {
	return fmt.Errorf("queue depth adapter: %s is server-side only and unavailable on the client", method)
}
