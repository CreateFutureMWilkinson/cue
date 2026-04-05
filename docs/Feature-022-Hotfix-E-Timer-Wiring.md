# Feature 022-Hotfix-E: Timer Tick Loop + Presenter ↔ View Binding

**Phase:** Phase-2-Feature-022-Hotfix-E
**Status:** Planned
**Package:** `internal/ui/`, `cmd/cue/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

The TimerPresenter and PlannerPresenter have callbacks (`SetOnTick`, `SetOnBlockComplete`, `SetOnStepChange`, etc.) but nothing in the UI subscribes to them, and no tick loop exists to drive the timer. This hotfix wires the presenters to the views and creates the timer tick loop in `main.go`.

## Issues to Fix

### 1. Timer Tick Loop Missing

The TimerPresenter provides `ElapsedFraction()`, `IsFlashVisible()`, `ActiveSegment()`, `CurrentTaskName()`, and `IsRunning()`. These need to be wired to the CountdownTimer widget in the focus rail:

- UI tick loop (1Hz) calls `timerPresenter.Tick()`
- After tick: `timer.SetProgress(timerPresenter.ElapsedFraction())`
- After tick: `timer.SetFlashVisible(timerPresenter.IsFlashVisible())`
- After tick: `focusRail.SetCurrentTask(timerPresenter.CurrentTaskName())`

This tick loop doesn't exist yet. It should be a goroutine in `main.go` that runs while the app is open.

### 2. PlannerPresenter ↔ View Binding

The PlannerPresenter manages wizard state and active schedule state, but the views don't subscribe to its changes. The wiring must:
- Listen for step changes to swap wizard content (wizard_view refreshes on step change)
- Listen for schedule-ready to display the schedule tree in plan view
- Listen for block-advance to update the timer and current task
- Wire "Done" button in focus rail to `plannerPresenter.CompleteCurrentTask()`
- Wire "Abandon Plan" to `plannerPresenter.AbandonPlan()` and update focus rail state
- Call `focusRail.SetActivePlan(true/false)` when plan state changes

### 3. Existing Plan Auto-Load

On startup, `plannerPresenter.LoadExistingPlan()` should be called. If a plan exists for today, the UI should reflect the active state (timer running, focus rail showing task, etc.).

## Files

| File | Action |
|---|---|
| `cmd/cue/main.go` | Modify — create timer tick loop goroutine, wire presenter callbacks to views, auto-load existing plan |
| `internal/ui/window.go` | Modify — accept PlannerPresenter and TimerPresenter for binding (or wire externally) |

## Dependencies

- 022-A (center view router wiring) — view switching must work
- 022-B (plan view) — schedule tree must exist to display active plan
- 022-D (wizard steps) — wizard view must exist for step changes to refresh
- TimerPresenter + PlannerPresenter (Feature 022) — already fully tested

## Test Coverage

**Timer Tick Loop:**
- Tick loop updates timer widget progress via ElapsedFraction()
- Tick loop updates flash visibility via IsFlashVisible()
- Tick loop updates task label via CurrentTaskName()
- Block complete callback fires and advances to next block

**Presenter ↔ View Binding:**
- Step change refreshes wizard view content
- Schedule selection transitions from wizard to plan view with schedule tree
- "Done" button calls CompleteCurrentTask()
- "Abandon Plan" calls AbandonPlan() and hides timer/task in focus rail
- Focus rail shows timer + task label when plan is active
- Focus rail hides timer + task label when no plan

**Auto-Load:**
- Existing plan loaded on startup
- Focus rail reflects active plan state on startup
- No plan: focus rail in default state
