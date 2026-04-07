# Feature 077 — Plan My Day Button Is a Noop

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | High |
| Status | Done |
| Depends on | 056, 071 |
| UI Tests | Yes |

## Problem

Clicking "Plan > Plan My Day" in the focus rail does nothing visible. The button handler calls `v.router.NavigateTo(ViewWizard)`, which triggers `switchCenterView()` in `MainWindow`, but the wizard view content either doesn't render or the navigation doesn't produce a visible change.

## Root Cause

Three interacting defects:

1. **Missing StartPlanning call**: The "Plan My Day" button navigated to ViewWizard but never called `PlannerPresenter.StartPlanning()`, so the wizard step stayed at `StepIdle`. `WizardView.renderContainer()` had no case for `StepIdle`, rendering an empty container.

2. **No idle state in WizardView**: Even as a safety net, `renderContainer()` had no `StepIdle` case — any display of the wizard before `StartPlanning` showed a blank view.

3. **Thread-unsafe view switching**: `switchCenterView()` manipulated Fyne canvas objects without wrapping in `fyne.Do()`, risking silent failures when called from non-UI goroutines.

## Changes Made

1. **PlannerView**: Added `SetOnPlanMyDay(fn func())` callback setter. Button handler invokes this callback before navigating.

2. **AppBinder.Bind()**: Wires `SetOnPlanMyDay` to call `plannerP.StartPlanning(ctx)` then `viewRouter.NavigateTo(ViewWizard)`. This transitions the wizard from `StepIdle` to `StepTaskSelect` before the view swap.

3. **WizardView**: Added `renderIdle()` method with a guiding prompt label, and a `StepIdle` case in `renderContainer()`. Acts as safety net if the view is displayed before `StartPlanning` completes.

4. **MainWindow.switchCenterView()**: Wrapped canvas object manipulation in `fyne.Do()` for thread safety (shared fix with Feature 083).

5. **Interface changes**: Added `StartPlanning` to `PlannerCallbacks`, `SetOnPlanMyDay` to `PlannerViewBindable`.

## Acceptance Criteria

- Clicking "Plan My Day" navigates to the wizard view with step 1 visible
- If no plan data exists, the wizard shows step 1 (task selection) as the starting state
- Navigation is visually distinct (center column content changes)
- Wizard at StepIdle shows a meaningful prompt (not empty)

## Test Coverage

- UI acceptance tests: `TestBug077Suite` (3 tests) in `tests/ui/bugfix_acceptance_test.go`
- Unit tests: `TestPlanButtonInvokesOnPlanMyDayCallback` in `planner_view_test.go`
- Unit tests: `TestBindWiresPlanMyDayToStartPlanningAndNavigate` in `app_binder_test.go`
- Unit tests: `TestIdleStateRendersContent`, `TestIdleStateShowsPromptLabel` in `wizard_view_test.go`

## TDD Agent Stats

| Phase | Role | Commit |
|---|---|---|
| UI Tests | Test Designer | 4028616 |
| RED (callback) | Test Designer | 8f3ff39 |
| GREEN (callback) | Implementer | abb1757 |
| RED (binder wiring) | Test Designer | 88c9d5a |
| GREEN (binder wiring) | Implementer | d7360cb |
| RED (idle state) | Test Designer | 016be64 |
| GREEN (idle state) | Implementer | 460e498 |
| GREEN (thread safety) | Implementer | 18621bf |
