package presenter

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

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
	return p.ruleRepo.ListRules(ctx)
}

// SaveRule persists a new or updated routing rule.
func (p *RulesPresenter) SaveRule(ctx context.Context, rule *repository.RoutingRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}
	return p.ruleRepo.UpsertRule(ctx, rule)
}

// DeleteRule removes a routing rule by ID.
func (p *RulesPresenter) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return p.ruleRepo.DeleteRule(ctx, id)
}

// ReorderRule moves a rule to a new priority, shifting others as needed.
func (p *RulesPresenter) ReorderRule(ctx context.Context, id uuid.UUID, newPriority int) error {
	rules, err := p.ruleRepo.ListRules(ctx)
	if err != nil {
		return err
	}

	// Find the target rule and its current priority.
	var target *repository.RoutingRule
	for _, r := range rules {
		if r.ID == id {
			target = r
			break
		}
	}
	if target == nil {
		return repository.ErrNotFound
	}

	oldPriority := target.Priority
	if oldPriority == newPriority {
		return nil
	}

	// Shift affected rules and set the target's new priority.
	target.Priority = newPriority
	var changed []*repository.RoutingRule
	changed = append(changed, target)

	for _, r := range rules {
		if r.ID == id {
			continue
		}
		if oldPriority > newPriority {
			// Moving up: shift rules in [newPriority, oldPriority) down by 1.
			if r.Priority >= newPriority && r.Priority < oldPriority {
				r.Priority++
				changed = append(changed, r)
			}
		} else {
			// Moving down: shift rules in (oldPriority, newPriority] up by 1.
			if r.Priority > oldPriority && r.Priority <= newPriority {
				r.Priority--
				changed = append(changed, r)
			}
		}
	}

	// Sort by priority to upsert in order.
	sort.Slice(changed, func(i, j int) bool {
		return changed[i].Priority < changed[j].Priority
	})

	for _, r := range changed {
		if err := p.ruleRepo.UpsertRule(ctx, r); err != nil {
			return err
		}
	}

	return nil
}

// ToggleRule enables or disables a routing rule.
func (p *RulesPresenter) ToggleRule(ctx context.Context, id uuid.UUID, enabled bool) error {
	rule, err := p.ruleRepo.GetRule(ctx, id)
	if err != nil {
		return err
	}
	rule.Enabled = enabled
	return p.ruleRepo.UpsertRule(ctx, rule)
}

// QueueDepth returns the number of pending items in the Ollama queue.
func (p *RulesPresenter) QueueDepth(ctx context.Context) (int, error) {
	return p.queueRepo.PendingCount(ctx)
}

// QueueWarningThreshold returns the threshold above which a warning is shown.
func (p *RulesPresenter) QueueWarningThreshold() int {
	return p.warnAt
}
