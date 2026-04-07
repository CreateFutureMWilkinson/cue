# Feature 072 — Wizard Step 3 Up/Down Reorder Buttons Are Noops

**Phase:** Phase-6-Feature-072
**Type:** Bugfix
**Severity:** Medium
**Depends on:** 071

## Problem

In the day planner wizard step 3 (priority ordering), the "Up" and "Down" buttons for reordering tasks do nothing. They are rendered but their callbacks are empty `func() {}`.

## Root Cause

`wizard_view.go:390-391`:
```go
widget.NewButton("Up", func() {}),
widget.NewButton("Down", func() {}),
```

The `WizardViewModel` interface defines `ReorderTask(from, to int)` and `PlannerPresenter` implements it, but the buttons never call it.

## Fix

1. Track the currently selected task index (or use the button's position in the list).
2. Wire "Up" button to call `v.vm.ReorderTask(index, index-1)` (swap with previous).
3. Wire "Down" button to call `v.vm.ReorderTask(index, index+1)` (swap with next).
4. After reorder, refresh the view to reflect the new order.
5. Disable "Up" on the first item and "Down" on the last item.

## Files to Change

- `internal/ui/wizard_view.go` — wire Up/Down callbacks to `ReorderTask`

## Acceptance Criteria

- [ ] "Up" button moves a task one position higher in the priority list
- [ ] "Down" button moves a task one position lower in the priority list
- [ ] "Up" is disabled/hidden for the first task
- [ ] "Down" is disabled/hidden for the last task
- [ ] Priority list visually updates after reorder
- [ ] Reorder persists to the presenter's task order
