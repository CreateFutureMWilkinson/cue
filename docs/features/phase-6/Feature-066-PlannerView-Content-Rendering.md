# Feature 066 — PlannerView No-Plan Content Not Rendered

**Phase:** Phase-6-Feature-066
**Type:** Bugfix
**Severity:** High
**Depends on:** Feature 063, Feature 071

## Problem

The Plan view gives no way to access the wizard. When the user navigates to the Plan view with no active plan, they should see a humorous placeholder message and a "Plan My Day" button. Instead, the `PlannerView` container only renders the HSplit of navigation buttons (left) and todo list (right) — the placeholder text and schedule tree are computed in `buildContent()` and stored in fields but never placed into the widget tree.

## Root Cause

`planner_view.go:90-108` builds the container as:
```go
leading := container.NewVBox(planBtn, nextBtn, backBtn, completeTaskBtn, abandonBtn)
split := container.NewHSplit(leading, trailing)
v.container = container.NewStack(split)
```

The `placeholderText` and `scheduleTree` fields are populated but never rendered into the container. The "Plan My Day" button is in the `leading` VBox but there is no center content area showing the placeholder or schedule tree.

## Fix

1. Restructure `PlannerView` container to include a center content area between the navigation buttons and the todo list.
2. When `StepIdle` with no active plan: render placeholder text (centered) and the "Plan My Day" button below it.
3. When an active plan exists: render the schedule tree in the center content area.
4. When in wizard steps: hide PlannerView content (the wizard view handles rendering).
5. `Refresh()` must update the center content area, not just button visibility.

## Files to Change

- `internal/ui/planner_view.go` — restructure container to include center content
- `internal/ui/planner_view_test.go` — verify placeholder/schedule tree renders in container (if UI tests exist)

## Acceptance Criteria

- [ ] No-plan state shows random placeholder message text in center area
- [ ] No-plan state shows "Plan My Day" button
- [ ] Tapping "Plan My Day" navigates to ViewWizard
- [ ] Active plan state renders schedule tree in center area
- [ ] Schedule tree shows Pomodoro cycles with colored bars
