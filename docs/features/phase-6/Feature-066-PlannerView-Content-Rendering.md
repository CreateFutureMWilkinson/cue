# Feature 066 — PlannerView No-Plan Content Not Rendered

**Phase:** Phase-6-Feature-066
**Type:** Bugfix
**Severity:** High
**Depends on:** Feature 063, Feature 071

## Problem

The Plan view gives no way to access the wizard. When the user navigates to the Plan view with no active plan, they should see a humorous placeholder message and a "Plan My Day" button. Instead, the `PlannerView` container only renders the HSplit of navigation buttons (left) and todo list (right) — the placeholder text and schedule tree are computed in `buildContent()` and stored in fields but never placed into the widget tree.

## Root Cause

`planner_view.go:90-108` built the container as:
```go
leading := container.NewVBox(planBtn, nextBtn, backBtn, completeTaskBtn, abandonBtn)
split := container.NewHSplit(leading, trailing)
v.container = container.NewStack(split)
```

The `placeholderText` and `scheduleTree` fields were populated but never rendered into the container.

## Fix

Added a `centerContent *fyne.Container` field to `PlannerView` that is populated by `updateCenterContent()`:

1. **No-plan state (StepIdle, no active plan):** Renders a centered `widget.Label` with the random placeholder message.
2. **Active plan state:** Renders a schedule cycle summary label from the `ScheduleTree.Cycles()` data.
3. **Refresh()** calls `buildContent()` which calls `updateCenterContent()` to swap the center content dynamically.
4. The leading pane uses `container.NewBorder(buttons, nil, nil, nil, centerContent)` so the buttons are at the top and the center content fills the remaining space.

## Files Changed

- `internal/ui/planner_view.go` — added `centerContent` field, `updateCenterContent()` method, restructured container layout
- `internal/ui/planner_view_test.go` — added `TestNoPlanPlaceholderLabelInWidgetTree` verifying the Label exists in the widget tree

## Acceptance Criteria

- [x] No-plan state shows random placeholder message text in center area
- [x] No-plan state shows "Plan My Day" button
- [x] Tapping "Plan My Day" navigates to ViewWizard
- [x] Active plan state renders schedule tree in center area
- [x] Refresh() updates center content dynamically

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestNoPlanPlaceholderLabelInWidgetTree` | Placeholder text rendered as `widget.Label` in container widget tree |
| `TestNoPlanShowsPlaceholderMessage` | PlaceholderText() returns valid message |
| `TestNoPlanShowsPlanButton` | Plan button visible in idle state |
| `TestPlanButtonNavigatesToWizard` | Plan button navigates to ViewWizard |
| `TestActivePlanShowsScheduleTree` | ScheduleTree() non-nil with active plan |
| `TestRefreshUpdatesContent` | Content transitions from idle to active |
| Bug066 acceptance: `TestPlaceholderTextRenderedInWidgetTree` | Label in widget tree (acceptance) |
| Bug066 acceptance: `TestPlanMyDayButtonNavigatesToWizard` | Button navigation (acceptance) |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~34s | ~30,000 | 024ae2e |
| GREEN | Implementer | manual | — | 81b417e |
| REFACTOR | Refactorer | manual | — | 35b9b13 |
