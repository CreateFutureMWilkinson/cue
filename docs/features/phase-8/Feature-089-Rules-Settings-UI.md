# Feature 089: Settings UI — Rules Tab

**Phase:** Phase-8-Feature-089
**Status:** Planned
**Packages:** `internal/ui/`, `internal/ui/presenter/`
**Depends on:** Feature 084

---

## Overview

Add a "Rules" tab to the Settings view for managing deterministic routing rules. Users can view, add, reorder, and delete rules through the UI.

**UiSpec.md impact:** The Settings view already uses a tabbed layout with 5 tabs: Slack, Email, Calendar, Audio, Ollama (documented in `docs/guides/UiSpec.md`). This feature adds a new "Rules" tab to that existing tab structure. The docs commit for this feature must:
1. Update the Settings View acceptance criteria in UiSpec.md from "5 tabs" to "6 tabs" and add "Rules" to the tab name list
2. Add a `### Settings — Rules Tab` section with the acceptance criteria listed in the "UI Acceptance Tests" section below

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

## UI Acceptance Tests (Step 1 — Before TDD Micro-Loops)

Per the UI Feature Workflow in CLAUDE.md, UI acceptance tests must be written and committed **before** implementation begins. These tests go in `tests/ui/settings_acceptance_test.go` (extending the existing `SettingsAcceptanceSuite`), tagged with `//go:build ui_acceptance`, and run via `just test-ui`.

The existing test file uses `newSettingsView()` helpers and `uitest.FindWidget`/`uitest.RequireWidget` for assertions. The `NewSettingsView` constructor will need to accept a `RulesPresenter` (or the Rules tab content will need to be wired through an existing dependency). The helper `newSettingsView()` in `tests/ui/helpers_test.go` must be updated to provide the new dependency.

### Acceptance Criteria to Test

These map to UiSpec.md entries that will be added in the docs commit:

**Tab structure:**
- [ ] Settings view has 6 tabs (currently 5 — Slack, Email, Calendar, Audio, Ollama)
- [ ] Tab names are `["Slack", "Email", "Calendar", "Rules", "Audio", "Ollama"]` in that order
- [ ] Rules tab is at index 3

**Empty state:**
- [ ] Rules tab with no rules shows empty state text: "No routing rules configured"
- [ ] Rules tab with no rules shows "Add Rule" button

**Rule list:**
- [ ] Rules tab with pre-existing rules displays them sorted by priority
- [ ] Each rule row shows source, field, pattern, action
- [ ] Each rule row has an enabled checkbox
- [ ] Each rule row has Up/Down reorder buttons
- [ ] Each rule row has a Delete button
- [ ] First rule's Up button is disabled
- [ ] Last rule's Down button is disabled

**Add Rule form:**
- [ ] Tapping "Add Rule" replaces list with form
- [ ] Form has Source dropdown with options: Email, Slack, All
- [ ] Form has Field dropdown (options change based on Source selection)
- [ ] Selecting Email source shows fields: sender, subject
- [ ] Selecting Slack source shows fields: sender, channel, content, message_type
- [ ] Selecting All source shows fields: sender
- [ ] Form has Pattern text entry
- [ ] Form has Negate checkbox
- [ ] Form has Action dropdown: Notified, Ignored
- [ ] Form has Save and Cancel buttons
- [ ] Cancel returns to rule list without saving

**Validation:**
- [ ] Saving with invalid regexp shows inline error
- [ ] Saving with valid data persists rule and returns to list
- [ ] Newly saved rule appears in the list

**Queue depth indicator:**
- [ ] Queue depth label shown at top of Rules tab
- [ ] Queue depth shows warning text when exceeding threshold

**Reordering:**
- [ ] Tapping Down on a rule swaps it with the next rule
- [ ] Tapping Up on a rule swaps it with the previous rule

**Deletion:**
- [ ] Tapping Delete removes the rule from the list

### Commit

```
test(ui): failing acceptance tests for Rules settings tab
```

These tests will all FAIL initially (the Rules tab doesn't exist yet). TDD micro-loops then drive implementation until `just test-ui` passes.
