# Feature 022-Hotfix-E: Timer Tick Loop + Presenter ↔ View Binding

**Phase:** Phase-2-Feature-022-Hotfix-E
**Status:** Done
**Package:** `internal/ui/`, `internal/ui/presenter/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Wires presenter callbacks to views and creates the timer tick loop that drives the countdown timer widget at 1Hz. This is the final hotfix in the Feature 022 series, completing the planner UI by connecting the already-tested presenters (PlannerPresenter, TimerPresenter) to the already-built views (FocusRail, WizardView, PlannerView).

## Design Decisions

### TimerLoop with Interfaces

A dedicated `TimerLoop` type in `internal/ui/timer_loop.go` encapsulates the 1Hz tick goroutine. It depends on three interfaces (`TickableTimer`, `TimerWidget`, `TaskUpdater`) rather than concrete types, enabling deterministic testing without goroutine timing.

The `TickOnce()` method performs a single tick cycle synchronously — tests call it directly. The `Start(ctx)` method wraps `TickOnce()` in a 1-second ticker goroutine for production use.

### AppBinder for Callback Wiring

A dedicated `AppBinder` type in `internal/ui/app_binder.go` wires presenter callbacks to view updates. It depends on four interfaces (`PlannerCallbacks`, `FocusRailCallbacks`, `RefreshableView`, `ViewNavigator`) for testability.

`Bind()` wires two callbacks:
- `SetOnStepChange` — refreshes wizard view on every step change; additionally navigates to plan view and activates focus rail on `StepActive`, deactivates on `StepIdle`
- `SetOnDone` — calls `CompleteCurrentTask` when the Done button is tapped

`AutoLoad(ctx)` calls `LoadExistingPlan` and updates focus rail active state.

### PlannerPresenter.SetOnStepChange

Added a callback field + `fireStepChange()` helper. Every method that assigns `p.step` now calls `fireStepChange()` afterward. This includes: `StartPlanning`, `NextStep` (all branches), `PreviousStep` (all branches), `SelectSchedule`, `AbandonPlan`, `LoadExistingPlan`.

### FocusRail.Container()

Added a `Container()` method returning a `*fyne.Container` with VBox layout of all rail widgets (timer, task label, plan/back/done/review buttons). This allows `window.go` to use the real FocusRail instead of the placeholder label.

## API

### New Types

| Type | Package | Purpose |
|---|---|---|
| `TimerLoop` | `internal/ui` | 1Hz tick loop driving timer → widget updates |
| `AppBinder` | `internal/ui` | Presenter ↔ view callback wiring |

### New Interfaces

| Interface | Package | Methods |
|---|---|---|
| `TickableTimer` | `internal/ui` | `Tick()`, `ElapsedFraction()`, `IsFlashVisible()`, `CurrentTaskName()` |
| `TimerWidget` | `internal/ui` | `SetProgress(float64)`, `SetFlashVisible(bool)` |
| `TaskUpdater` | `internal/ui` | `SetCurrentTask(string)` |
| `PlannerCallbacks` | `internal/ui` | `SetOnStepChange(...)`, `HasActivePlan()`, `LoadExistingPlan(...)`, `CompleteCurrentTask(...)` |
| `FocusRailCallbacks` | `internal/ui` | `SetActivePlan(bool)`, `SetCurrentTask(string)`, `SetOnDone(func())` |
| `RefreshableView` | `internal/ui` | `Refresh()` |
| `ViewNavigator` | `internal/ui` | `NavigateTo(CenterView)` |

### New Methods on Existing Types

| Method | Type | Purpose |
|---|---|---|
| `SetOnStepChange(func(WizardStep))` | `PlannerPresenter` | Register step-change callback |
| `Container() *fyne.Container` | `FocusRail` | Get VBox container of all rail widgets |

## Error Handling

- `NewTimerLoop` / `NewAppBinder` return errors on nil dependencies
- `AutoLoad` propagates `LoadExistingPlan` errors
- `Bind()` Done callback swallows `CompleteCurrentTask` errors (fire-and-forget from UI)

## Test Coverage

24 new tests across 4 files:

| File | Tests | Coverage |
|---|---|---|
| `planner_presenter_test.go` | 6 | SetOnStepChange fires on all step transitions |
| `focus_rail_test.go` | 1 | Container() returns non-nil |
| `timer_loop_test.go` | 9 | Constructor validation, TickOnce behavior, Stop |
| `app_binder_test.go` | 8 | Constructor validation, Bind wiring, AutoLoad |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 128s | 67,887 | c0ef44b |
| GREEN | Implementer | 137s | 53,383 | db548d8 |
| REFACTOR | Refactorer | 65s | 34,777 | e604f58 |
