# Feature 056: Plan and Wizard View Wiring

**Phase:** Phase-6-Feature-056
**Type:** Bugfix
**Severity:** High
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 022B (Plan View), Feature 022D (Wizard Steps), Feature 016 (Three-Column Layout)

---

## Bug Description

The Plan and Wizard views in the center view router are placeholder `widget.NewLabel` instances instead of the real `PlannerView` and `WizardView` widgets. Both implementations exist in their respective files but are never instantiated in `MainWindow`.

## Expected Behavior

Navigating to the Plan view should show `PlannerView` (schedule tree + todo list). Navigating to the Wizard view should show `WizardView` (4-step day planner wizard). Both are specified in UiSpec.md and implemented in `planner_view.go` / `wizard_view.go`.

## Actual Behavior

`window.go:89-90` sets `ViewPlan: widget.NewLabel("Plan")` and `ViewWizard: widget.NewLabel("Wizard")`. The real widgets are dead code.

## Root Cause

Views were built in Phase 2 (Features 022B, 022D) but never wired into the `MainWindow` view map. Placeholder labels remain from initial scaffolding.

## Proposed Fix

Instantiate `PlannerView` and `WizardView` in `MainWindow`, passing required presenters (planner presenter, todo presenter, wizard presenter). Replace the placeholder labels in the view map.

## Test Strategy

- RED: Structural test asserting `ViewPlan` contains `ScheduleTree` and `TodoListView`, not a label
- RED: Structural test asserting `ViewWizard` contains wizard step content, not a label
- GREEN: Wire both views into `window.go`
- REFACTOR: Remove dead placeholder code
