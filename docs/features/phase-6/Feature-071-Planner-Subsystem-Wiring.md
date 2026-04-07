# Feature 071 — Planner Subsystem Not Wired in main.go

**Phase:** Phase-6-Feature-071
**Type:** Bugfix
**Severity:** Critical
**Depends on:** —

## Problem

The entire Phase 2 day planner subsystem — `PlannerPresenter`, `TimerPresenter`, `TimerLoop`, `AppBinder`, `TodoRepository`, `ScheduleRepository`, `CalendarProvider`, and planner engine — is fully implemented and tested but **never instantiated or wired in `cmd/cue/main.go`**.

At `main.go:344`, the `NewMainWindow` call passes `nil, nil, nil` for `plannerVM`, `timerVM`, and `wizardVM`:

```go
mainWindow := ui.NewMainWindow(..., nil, nil, nil)
```

This means:
- The Plan view shows a placeholder `widget.NewLabel("Plan")` instead of `PlannerView`
- The Wizard view shows a placeholder `widget.NewLabel("Wizard")` instead of `WizardView`
- The countdown timer never ticks
- The `AppBinder` is never created, so no step-change callbacks fire
- Todo list is inaccessible
- Schedule persistence never loads existing plans on startup

## Root Cause

`main.go` creates all Phase 1 dependencies (orchestrator, router, notifications, feedback, alerts, character) but never creates the Phase 2 dependencies:
- No `sqlite.NewSQLiteTodoRepository()` call
- No `sqlite.NewSQLiteScheduleRepository()` call  
- No `sqlite.NewSQLiteCategoryRepository()` call
- No `calendar.NewProvider()` or equivalent
- No `planner.NewGenerator()` or `planner.NewEstimator()` calls
- No `presenter.NewPlannerPresenter()` call
- No `presenter.NewTimerPresenter()` call
- No `ui.NewTimerLoop()` call
- No `ui.NewAppBinder()` call
- No `planner.NewRealClock()` call

## Fix

1. Create `TodoRepository`, `CategoryRepository`, `ScheduleRepository` from the existing SQLite DB connection.
2. Create `CalendarProvider` (may use a noop/stub if ICS config not present).
3. Create `planner.Generator`, `planner.Estimator`, and `planner.RealClock`.
4. Create `PlannerPresenter` with all dependencies.
5. Create `TimerPresenter` with clock and timer alerter.
6. Pass `plannerPresenter`, `timerPresenter`, and `plannerPresenter` (as wizard VM) to `NewMainWindow` instead of `nil`.
7. After window creation, create `AppBinder` with the planner presenter, focus rail, wizard view, planner view, and view router. Call `Bind()` and `AutoLoad()`.
8. Create and start `TimerLoop`.

## Files to Change

- `cmd/cue/main.go` — wire all Phase 2 dependencies

## Acceptance Criteria

- [ ] Plan view shows `PlannerView` (not a placeholder label)
- [ ] Wizard view shows `WizardView` (not a placeholder label)
- [ ] "Plan My Day" button is visible and navigates to wizard
- [ ] Wizard steps 1-4 are functional
- [ ] Active plan loads on startup
- [ ] Countdown timer ticks at 1 Hz
- [ ] Done button completes current task
- [ ] Abandon Plan deletes schedule and returns to no-plan state
