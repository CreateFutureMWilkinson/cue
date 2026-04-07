# Feature 073 — PlannerView Navigation Buttons Not Wired

**Phase:** Phase-6-Feature-073
**Type:** Bugfix
**Severity:** High
**Depends on:** 071

## Problem

The `PlannerView` has four navigation/control buttons — Next, Back, Complete Task, and Abandon Plan — that are initialized with empty `func() {}` callbacks and never get wired to actual presenter methods.

## Root Cause

`planner_view.go:118-121`:
```go
v.nextBtn = widget.NewButton("Next", func() {})
v.backBtn = widget.NewButton("Back", func() {})
v.completeTaskBtn = widget.NewButton("Complete Task", func() {})
v.abandonBtn = widget.NewButton("Abandon Plan", func() {})
```

The `AppBinder` only wires `focusRail.SetOnDone` — it does not wire these PlannerView buttons. There is no mechanism to inject callbacks for these buttons.

## Solution

### 1. Setter methods on PlannerView

Added four setter methods that update the Fyne button's `OnTapped` callback:

- `SetOnNext(fn func())` — wires Next button
- `SetOnBack(fn func())` — wires Back button
- `SetOnCompleteTask(fn func())` — wires Complete Task button
- `SetOnAbandonPlan(fn func())` — wires Abandon Plan button

### 2. PlannerViewBindable interface

Introduced `PlannerViewBindable` interface in `app_binder.go` combining `RefreshableView` with the four setter methods. This replaces the `RefreshableView` type for the `plannerView` parameter in `AppBinder`.

### 3. Expanded PlannerCallbacks interface

Added `NextStep(ctx) error`, `PreviousStep()`, and `AbandonPlan(ctx) error` to `PlannerCallbacks` so `AppBinder.Bind()` can delegate to the presenter.

### 4. Wiring in AppBinder.Bind()

`Bind()` now calls all four setters with closures that delegate to the presenter:
- Next → `plannerP.NextStep(ctx)`
- Back → `plannerP.PreviousStep()`
- Complete Task → `plannerP.CompleteCurrentTask(ctx)`
- Abandon Plan → `plannerP.AbandonPlan(ctx)`

Errors are intentionally discarded (same pattern as the existing Done button wiring) — UI callbacks should not panic.

## Files Changed

- `internal/ui/planner_view.go` — added 4 setter methods
- `internal/ui/app_binder.go` — expanded interfaces, wiring in Bind()
- `internal/ui/window.go` — changed PlannerViewRef() return type to PlannerViewBindable
- `internal/ui/app_binder_test.go` — 4 new tests, extracted expectBindCalls() helper
- `internal/ui/planner_view_test.go` — 4 new setter tests
- `tests/ui/bugfix_acceptance_test.go` — 5 acceptance tests (4 button callbacks + no-panic)
- `tests/ui/helpers_test.go` — added configurable step field to stubPlannerTimerVM

## Acceptance Criteria

- [x] "Next" button advances the wizard step via presenter
- [x] "Back" button returns to previous wizard step via presenter
- [x] "Complete Task" button completes the current task via presenter
- [x] "Abandon Plan" button deletes the schedule and returns to idle
- [x] All buttons handle presenter errors gracefully (no panics)

## Test Coverage

| Area | Tests |
|---|---|
| PlannerView setters (unit) | 4 tests — each setter wires OnTapped |
| AppBinder.Bind() wiring (unit) | 4 tests — each button delegates to presenter |
| UI acceptance | 5 tests — 4 callback invocations + no-panic safety |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| UI TESTS | orchestrator | manual | — | 78314ba |
| RED (setters 1) | Test Designer | ~32s | ~30,000 | 5229fb7 |
| GREEN (setters 1) | Implementer | ~26s | ~22,000 | 0c72d1c |
| RED (setters 2) | Test Designer | ~36s | ~34,000 | 2e56830 |
| GREEN (setters 2) | orchestrator | manual | — | 98bf53f |
| RED (binder) | Test Designer | ~124s | ~41,000 | 7af1b7f |
| GREEN (binder) | orchestrator | manual | — | 801a595 |
