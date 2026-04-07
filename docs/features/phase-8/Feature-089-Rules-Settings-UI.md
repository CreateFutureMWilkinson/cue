# Feature 089: Settings UI — Rules Tab

**Phase:** Phase-8-Feature-089
**Status:** Planned
**Packages:** `internal/ui/`, `internal/ui/presenter/`
**Depends on:** Feature 084

---

## Overview

Add a "Rules" tab to the Settings view for managing deterministic routing rules. Users can view, add, reorder, and delete rules through the UI.

## Tab Placement

New tab positioned after Calendar and before Audio:

```
Slack | Email | Calendar | Rules | Audio | Ollama
```

## Rule List View

Sorted by priority (ascending). Each row displays:

- **Priority number** (e.g., "0", "1", "2")
- **Source** ("email", "slack", "all")
- **Summary**: `[field] [matches|not matches] [pattern] → [action]`
- **Enabled toggle** (checkbox)
- **Up/Down buttons** for reordering (same pattern as wizard step 3 priority reordering)
- **Delete button**

Up/Down changes the priority values behind the scenes. First item's Up is disabled, last item's Down is disabled.

### Empty State

When no rules exist:

```
No routing rules configured. Messages will be queued for Ollama scoring.
Tap "Add Rule" to create deterministic routing rules.
```

## Add Rule Form

Shown when "Add Rule" button is tapped (replaces list, same pattern as account forms):

- **Source dropdown**: Email, Slack, All
- **Field dropdown**: updates options based on selected source
  - Email: sender, subject
  - Slack: sender, channel, content, message_type
  - All: sender
- **Pattern entry**: text field for Go regexp
- **Negate checkbox**: "Invert match (not matches)"
- **Action dropdown**: Notified, Ignored
- **Save / Cancel buttons**

### Validation on Save

- Pattern must compile as valid Go regexp
- Field must be valid for selected source
- Show inline error message on validation failure

## Queue Depth Indicator

Shown at the top of the Rules tab:

```
Ollama queue: 3 pending
```

If queue depth exceeds the configured warning threshold:

```
⚠ Ollama queue: 57 pending — consider adding more rules
```

## Presenter

```go
type RulesPresenter struct {
    ruleRepo repository.RoutingRuleRepository
    queueRepo repository.QueueRepository
}

func (p *RulesPresenter) ListRules(ctx context.Context) ([]*repository.RoutingRule, error)
func (p *RulesPresenter) SaveRule(ctx context.Context, rule *repository.RoutingRule) error
func (p *RulesPresenter) DeleteRule(ctx context.Context, id uuid.UUID) error
func (p *RulesPresenter) ReorderRule(ctx context.Context, id uuid.UUID, newPriority int) error
func (p *RulesPresenter) QueueDepth(ctx context.Context) (int, error)
```
