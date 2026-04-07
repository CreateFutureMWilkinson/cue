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

## Fix

1. Add setter methods or constructor parameters to `PlannerView` for button callbacks:
   - `SetOnNext(fn func())` — calls `plannerPresenter.NextStep()`
   - `SetOnBack(fn func())` — calls `plannerPresenter.PreviousStep()`
   - `SetOnCompleteTask(fn func())` — calls `plannerPresenter.CompleteCurrentTask()`
   - `SetOnAbandonPlan(fn func())` — calls `plannerPresenter.AbandonPlan()`
2. Wire these in `AppBinder.Bind()` or in `main.go` after creating the PlannerView.
3. Button callbacks should delegate to the presenter and handle errors gracefully (log, don't panic).

## Files to Change

- `internal/ui/planner_view.go` — add callback setters, wire button OnTapped
- `internal/ui/app_binder.go` — wire PlannerView button callbacks in `Bind()`

## Acceptance Criteria

- [ ] "Next" button advances the wizard step via presenter
- [ ] "Back" button returns to previous wizard step via presenter
- [ ] "Complete Task" button completes the current task via presenter
- [ ] "Abandon Plan" button deletes the schedule and returns to idle
- [ ] All buttons handle presenter errors gracefully (no panics)
