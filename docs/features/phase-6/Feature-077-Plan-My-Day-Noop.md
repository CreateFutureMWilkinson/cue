# Feature 077 — Plan My Day Button Is a Noop

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | High |
| Status | Planned |
| Depends on | 056, 071 |
| UI Tests | Yes |

## Problem

Clicking "Plan > Plan My Day" in the focus rail does nothing visible. The button handler calls `v.router.NavigateTo(ViewWizard)`, which triggers `switchCenterView()` in `MainWindow`, but the wizard view content either doesn't render or the navigation doesn't produce a visible change.

## Root Cause Analysis

Investigation shows the wiring chain exists:

1. `PlannerView.planBtn` calls `router.NavigateTo(ViewWizard)` (planner_view.go:118-122)
2. `CenterViewRouter` fires `OnViewChange` callbacks
3. `MainWindow.switchCenterView()` swaps `centerStack.Objects`

Potential failure points:

- **WizardVM nil**: If `plannerPresenter` passed as `wizardVM` is nil at construction time, `NewMainWindow` falls back to a placeholder label (`widget.NewLabel("Wizard")`) — the view "switches" but shows a blank label indistinguishable from the current view
- **WizardView never initializes steps**: Even if the WizardView is created, its step content may not render because the planner subsystem hasn't generated a plan yet and there's no empty-state prompt
- **switchCenterView runs off Fyne thread**: See Feature 083 — the center stack swap may silently fail due to threading violations

## Required Changes

1. Ensure WizardView is created with a valid view model (not nil fallback)
2. WizardView should show a meaningful empty state when no plan exists (e.g., "Tap to start planning your day" with step 1 of the wizard)
3. Wrap `switchCenterView` in `fyne.Do()` (shared fix with Feature 083)
4. Verify the full chain: button tap -> router navigate -> view swap -> wizard step 1 renders

## Acceptance Criteria

- Clicking "Plan My Day" navigates to the wizard view with step 1 visible
- If no plan data exists, the wizard shows step 1 (task selection) as the starting state
- Navigation is visually distinct (center column content changes)

## UI Test Coverage

- UI acceptance test: tap Plan button, tap "Plan My Day", verify wizard step 1 content appears
