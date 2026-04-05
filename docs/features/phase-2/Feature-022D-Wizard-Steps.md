# Feature 022-Hotfix-D: Day Planner Wizard Steps 1-4

**Phase:** Phase-2-Feature-022-Hotfix-D
**Status:** Done
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Implements the Fyne UI for the 4-step day planner wizard. The `WizardView` renders step-specific content in the center column when `ViewWizard` is active, delegating all state transitions to the `WizardViewModel` interface (backed by `PlannerPresenter`).

## Design Decisions

### WizardViewModel Interface

A dedicated `WizardViewModel` interface combines read methods (from `PlannerViewModel`) with write/mutation methods needed to drive wizard interactions. This keeps the wizard view decoupled from the concrete `PlannerPresenter` and enables isolated testing with mocks.

Methods: `CurrentStep`, `AvailableTasks`, `Estimates`, `EstimateSummary`, `FocusSchedule`, `RecoverySchedule`, `SelectTask`, `AddTask`, `NextStep`, `PreviousStep`, `OverrideEstimate`, `ReorderTask`, `SelectSchedule`, `SelectedCount`.

### Cached State Pattern

`WizardView` uses a `buildState()` method (called from constructor and `Refresh()`) that reads the entire view model once and caches all computed fields. Accessors return cached values, avoiding repeated view model calls during rendering. This matches the pattern used in `PlannerView` and `TodoListView`.

### Step-Specific Content

Each wizard step renders distinct content:

| Step | Indicator | Primary Content | Navigation |
|---|---|---|---|
| 1 — Task Selection | "Step 1 of 4" | Checkbox list + category badges + inline add | Cancel, Next (disabled if 0 selected) |
| 2 — Estimates | "Step 2 of 4" | Estimate table + summary + overload warning | Back, Next |
| 3 — Priority | "Step 3 of 4" | Numbered task list + up/down reorder buttons | Back, Next |
| 4 — Schedule | "Step 4 of 4" | Two schedule cards (focus/recovery) with stats | Back, Select A/B |

### Schedule Card Stats

Each card displays: strategy name, focus block count, break count, and total focus time. Focus blocks are counted from `TimeBlockPreview` entries with `Type == "focus"`. Duration is formatted as `"Xh Ym"` (e.g., `"3h0m"`).

## Files

| File | Action |
|---|---|
| `internal/ui/wizard_view.go` | **New** — Day Planner Wizard steps 1-4 |
| `internal/ui/wizard_view_test.go` | **New** — wizard step rendering and navigation tests |

## Dependencies

- 022-A (center view router wiring) — wizard must display in center column via ViewWizard
- PlannerPresenter (Feature 022) — wizard delegates all state transitions to presenter

## API

### Types

- `WizardViewModel` — interface for wizard data source and mutations
- `WizardView` — Fyne component rendering wizard step content
- `TaskCheckboxItem` — view model for step 1 task checkboxes (ID, Title, Selected, Categories)
- `ScheduleCardStats` — view model for step 4 card stats (FocusBlocks, Breaks, TotalTime)

### Constructor

- `NewWizardView(vm WizardViewModel, router *CenterViewRouter) *WizardView`

### Accessors

- `Container()` — Fyne container
- `StepIndicator()` — "Step N of 4"
- `TaskCheckboxes()` — step 1 checkbox items
- `NextButtonEnabled()` — step 1 next button state
- `HasCancelButton()` — step 1 cancel presence
- `EstimateRows()` — step 2 estimate table data
- `SummaryText()` — step 2 "X of Y Pomodoros"
- `OverloadWarningVisible()` — step 2 overload state
- `PriorityList()` — step 3 ordered task titles
- `HasUpDownButtons()` — step 3 reorder buttons
- `ScheduleCards()` — step 4 card count
- `FocusCardStrategy()` / `RecoveryCardStrategy()` — strategy names
- `FocusCardStats()` / `RecoveryCardStats()` — card statistics
- `HasBackButton()` / `HasNextButton()` — navigation presence
- `Refresh()` — rebuild from view model

## Error Handling

No error paths — the wizard view is a pure projection of view model state. All error handling (Ollama failures, repository errors) is managed by the `PlannerPresenter`.

## Test Coverage

25 tests covering:
- Constructor (2): non-nil view and container
- Step 1 — Task Selection (7): step indicator, checkboxes, category badges, next enabled/disabled, cancel/next buttons
- Step 2 — Estimates (7): step indicator, estimate rows, summary text, overload hidden/visible, back/next buttons
- Step 3 — Priority (5): step indicator, numbered list, up/down buttons, back/next buttons
- Step 4 — Schedule (8): step indicator, two cards, focus/recovery strategy names, focus/recovery stats, back button, no next button
- Refresh (1): step indicator updates across steps

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 143s | 57,819 | 9625636 |
| GREEN | Implementer | 65s | 34,954 | f1a514f |
| REFACTOR | Refactorer | 80s | 38,462 | 2bc571b |
