package presenter

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// ErrNotImplemented is returned by stub implementations.
var ErrNotImplemented = errors.New("not implemented")

// RulesPresenter mediates between the Rules settings tab UI and the
// routing rule and queue repositories.
type RulesPresenter struct {
	ruleRepo  repository.RoutingRuleRepository
	queueRepo repository.QueueRepository
	warnAt    int
}

// NewRulesPresenter creates a RulesPresenter. warnAt is the queue depth
// threshold above which a warning is displayed.
func NewRulesPresenter(ruleRepo repository.RoutingRuleRepository, queueRepo repository.QueueRepository, warnAt int) *RulesPresenter {
	return &RulesPresenter{ruleRepo: ruleRepo, queueRepo: queueRepo, warnAt: warnAt}
}

// ListRules returns all routing rules sorted by priority.
func (p *RulesPresenter) ListRules(ctx context.Context) ([]*repository.RoutingRule, error) {
	return nil, ErrNotImplemented
}

// SaveRule persists a new or updated routing rule.
func (p *RulesPresenter) SaveRule(ctx context.Context, rule *repository.RoutingRule) error {
	return ErrNotImplemented
}

// DeleteRule removes a routing rule by ID.
func (p *RulesPresenter) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return ErrNotImplemented
}

// ReorderRule moves a rule to a new priority, shifting others as needed.
func (p *RulesPresenter) ReorderRule(ctx context.Context, id uuid.UUID, newPriority int) error {
	return ErrNotImplemented
}

// ToggleRule enables or disables a routing rule.
func (p *RulesPresenter) ToggleRule(ctx context.Context, id uuid.UUID, enabled bool) error {
	return ErrNotImplemented
}

// QueueDepth returns the number of pending items in the Ollama queue.
func (p *RulesPresenter) QueueDepth(ctx context.Context) (int, error) {
	return 0, ErrNotImplemented
}

// QueueWarningThreshold returns the threshold above which a warning is shown.
func (p *RulesPresenter) QueueWarningThreshold() int {
	return p.warnAt
}
