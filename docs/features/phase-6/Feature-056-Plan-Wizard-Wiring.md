# Feature 056: Plan and Wizard View Wiring

**Phase:** Phase-6-Feature-056
**Type:** Bugfix
**Severity:** High
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 022B (Plan View), Feature 022D (Wizard Steps), Feature 016 (Three-Column Layout), Feature 055 (Focus Rail Wiring)

---

## Bug Description

The Plan and Wizard views in the center view router are placeholder `widget.NewLabel` instances instead of the real `PlannerView` and `WizardView` widgets. Both implementations exist in their respective files but are never instantiated in `MainWindow`.

## Expected Behavior

Navigating to the Plan view should show `PlannerView` (schedule tree + todo list). Navigating to the Wizard view should show `WizardView` (4-step day planner wizard). Both are specified in UiSpec.md and implemented in `planner_view.go` / `wizard_view.go`.

## Actual Behavior

`window.go` sets `ViewPlan: widget.NewLabel("Plan")` and `ViewWizard: widget.NewLabel("Wizard")`. The real widgets are dead code.

## Root Cause

Views were built in Phase 2 (Features 022B, 022D) but never wired into the `MainWindow` view map. Placeholder labels remain from initial scaffolding.

## Fix

Extended `NewMainWindow` signature with three new parameters: `plannerVM PlannerViewModel`, `timerVM TimerViewModel`, `wizardVM WizardViewModel`. When the view model dependencies are non-nil, real view widgets are instantiated; otherwise falls back to placeholder labels (nil-safe for tests).

- `PlannerView` wired when both `plannerVM` and `timerVM` are non-nil
- `WizardView` wired when `wizardVM` is non-nil
- Pattern follows existing FocusRail (Feature 055) and SettingsView wiring

## API Changes

`NewMainWindow` gains three parameters after `viewRouter`:
```go
func NewMainWindow(
    ...,
    viewRouter *CenterViewRouter,
    plannerVM PlannerViewModel,   // NEW
    timerVM TimerViewModel,       // NEW
    wizardVM WizardViewModel,     // NEW
) *MainWindow
```

All existing call sites updated to pass `nil, nil, nil` (no presenter integration yet).

## Test Coverage

| Test | Purpose |
|---|---|
| `TestViewPlanShowsPlannerViewWhenVMsProvided` | Asserts ViewPlan content is not a label when VMs provided |
| `TestViewWizardShowsWizardViewWhenVMProvided` | Asserts ViewWizard content is not a label when VM provided |
| `TestPlanViewContentIsPlaceholderLabel` | Existing — confirms nil-VM fallback still works |
| `TestWizardViewContentIsPlaceholderLabel` | Existing — confirms nil-VM fallback still works |

## TDD Agent Stats

| TDD Phase | Agent | Commit |
|---|---|---|
| RED (behavior 1) | Test Designer | 6b2d455 |
| GREEN (behavior 1) | Implementer | 0cf4cf0 |
| RED (behavior 2) | Test Designer | 904f0f3 |
| GREEN (behavior 2) | Implementer | dd3828c |
