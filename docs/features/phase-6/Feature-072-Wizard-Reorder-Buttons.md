# Feature 072 — Wizard Step 3 Up/Down Reorder Buttons

**Phase:** Phase-6-Feature-072
**Type:** Bugfix
**Severity:** Medium
**Depends on:** 071

## Problem

In the day planner wizard step 3 (priority ordering), the "Up" and "Down" buttons for reordering tasks were noops. They rendered but their callbacks were empty `func() {}`.

## Root Cause

`wizard_view.go` rendered a single global pair of Up/Down buttons with empty callbacks. The `WizardViewModel` interface defined `ReorderTask(from, to int)` and `PlannerPresenter` implemented it, but the buttons never called it.

## Solution

Replaced global noop Up/Down buttons with **per-row** buttons in `renderStep3()`:

1. Each priority item now gets its own Up and Down button
2. Up button calls `v.vm.ReorderTask(idx, idx-1)` then `v.Refresh()`
3. Down button calls `v.vm.ReorderTask(idx, idx+1)` then `v.Refresh()`
4. First item's Up button is disabled (can't move higher)
5. Last item's Down button is disabled (can't move lower)
6. `Refresh()` re-reads the view model and re-renders, so the list updates visually

## Files Changed

- `internal/ui/wizard_view.go` — `renderStep3()` rewritten with per-row buttons
- `internal/ui/wizard_view_test.go` — 5 new unit tests + `findNthButton` helper
- `tests/ui/bugfix_acceptance_test.go` — Fixed Bug 072 acceptance tests to provide estimates and find enabled buttons

## Acceptance Criteria

- [x] "Up" button moves a task one position higher in the priority list
- [x] "Down" button moves a task one position lower in the priority list
- [x] "Up" is disabled for the first task
- [x] "Down" is disabled for the last task
- [x] Priority list visually updates after reorder
- [x] Reorder persists to the presenter's task order

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestStep3UpButtonCallsReorderTask` | Tapping 2nd item's Up calls `ReorderTask(1, 0)` |
| `TestStep3DownButtonCallsReorderTask` | Tapping 1st item's Down calls `ReorderTask(0, 1)` |
| `TestStep3FirstItemUpDisabled` | First item's Up button is disabled |
| `TestStep3LastItemDownDisabled` | Last item's Down button is disabled |
| `TestStep3UpButtonRefreshesView` | After tap, priority list order updates |
| `TestUpButtonCallsReorderTask` (UI acceptance) | Up button calls ReorderTask on VM |
| `TestDownButtonCallsReorderTask` (UI acceptance) | Down button calls ReorderTask on VM |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~102s | ~43,000 | c377959 |
| GREEN | Implementer | ~26s | ~23,000 | db13fc5 |
| FIX (acceptance) | orchestrator | manual | — | 4f5f14b |
