# Feature 022-Hotfix-A: Planner UI Views + Center View Wiring

**Phase:** Phase-2-Feature-022-Hotfix-A
**Status:** Planned
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Feature 022 delivered the presenter layer (PlannerPresenter, TimerPresenter) with full test coverage and the state machine logic, but the **Fyne view layer is almost entirely missing**. The center view router exists with ViewCharacter/ViewPlan/ViewWizard states, but `window.go` never swaps the center pane content when the router navigates. The `planner_view.go` file contains only a minimal button container with no actual UI rendering.

This hotfix implements:
1. Center view content switching in `window.go`
2. Plan Overview view (schedule tree + todo list)
3. Day Planner Wizard steps 1-4 as Fyne UI
4. Active schedule display with timer integration

## Issues to Fix

### 1. Center View Router Not Wired in Window (CRITICAL)

**Location:** `window.go:63-71`

```go
// Currently shows character widget + activity log drawer; later will route between
// ViewCharacter, ViewPlan, and ViewWizard.
var centerPane fyne.CanvasObject
if ap != nil {
    drawer := NewActivityLogDrawer(ap)
    centerPane = drawer.ContainerWithCharacter(characterWidget)
} else if characterWidget != nil {
    centerPane = characterWidget
} else {
    centerPane = widget.NewLabel("")
}
```

The comment says "later will route" — but it was never done. Clicking the Plan button in the focus rail calls `router.NavigateTo(ViewPlan)` which updates the router state, but the center pane never changes.

**Fix:** Use a `container.NewStack` (or similar) for the center pane. Register a `viewRouter.SetOnViewChange` callback that swaps the visible content:

```go
characterContent := drawer.ContainerWithCharacter(characterWidget)
planContent := planView.Container()
wizardContent := wizardView.Container()

centerStack := container.NewStack(characterContent)

viewRouter.SetOnViewChange(func(view CenterView) {
    centerStack.Objects = []fyne.CanvasObject{viewForState(view)}
    centerStack.Refresh()
})
```

### 2. Plan View Not Implemented

**Location:** `planner_view.go` — contains only 5 buttons in a VBox with empty/minimal callbacks.

**Required (from UI-SPEC.md):**

#### No-Plan State
When no active schedule exists, the Plan view shows:
- A "Plan My Day" button (centered, prominent)
- One of 7 motivational placeholder messages (randomly selected)
- Todo list on the right half (if todos exist)

#### Active Schedule State
When a schedule exists:
- **Left half:** Schedule tree view
  - Grouped by cycle: "Cycle 1/4", "Cycle 2/4", etc.
  - Each block as a row: time range, task name, block type icon
  - Color-coded timeline bars: green (focus), light blue (short break), blue (long break), grey (meeting)
  - Current block highlighted
- **Right half:** Todo list
  - Checkboxes for completion
  - Priority indicators
  - Category badges
  - Due dates
  - "Add Task" inline input
  - Click opens task detail modal

#### Action Buttons
- "Abandon Plan" — deletes schedule, returns to no-plan state
- "Plan My Day" — enters wizard (ViewWizard)

### 3. Day Planner Wizard Steps Not Implemented

The PlannerPresenter has all step transitions tested, but no Fyne UI exists.

**Step 1: Task Selection**
- Checkbox list of todos from repository
- Inline "Add Task" field
- Category display per task
- "Next" button (enabled when ≥1 task selected)
- "Cancel" returns to Plan view

**Step 2: Pomodoro Estimates**
- Table: task name | estimated pomodoros | override input
- Ollama estimation status indicator
- Fallback indicator (1 pomo) on estimation failure
- Overload warning if total > available slots
- "Next" / "Back" buttons

**Step 3: Priority Ordering**
- Draggable task list (or up/down buttons)
- Task name + estimated pomos displayed
- "Next" / "Back" buttons

**Step 4: Schedule Choice**
- Two schedule cards side by side:
  - "Focus Maximized" — description + mini timeline preview
  - "Recovery Balanced" — description + mini timeline preview
- Mini timeline: horizontal bar with color-coded blocks
- "Select" button on each card
- "Back" button

### 4. Timer Ring Integration with Active Schedule

The TimerPresenter provides `ElapsedFraction()`, `IsFlashVisible()`, `ActiveSegment()`, `CurrentTaskName()`, and `IsRunning()`. These need to be wired to the CountdownTimer widget in the focus rail:

- UI tick loop (1Hz or 30Hz) calls `timerPresenter.Tick()`
- After tick: `timer.SetProgress(timerPresenter.ElapsedFraction())`
- After tick: `timer.SetFlashVisible(timerPresenter.IsFlashVisible())`
- After tick: `focusRail.SetCurrentTask(timerPresenter.CurrentTaskName())`

This tick loop doesn't exist yet.

### 5. PlannerPresenter ↔ View Binding

The PlannerPresenter has callbacks (`SetOnStepChange`, `SetOnScheduleReady`, etc.) but nothing in the UI subscribes to them. The view must:
- Listen for step changes to swap wizard content
- Listen for schedule-ready to display the schedule tree
- Listen for block-advance to update the timer and current task

## Files

| File | Action |
|---|---|
| `internal/ui/window.go` | Modify — wire center view router to swap center pane content |
| `internal/ui/planner_view.go` | Rewrite — implement Plan Overview (no-plan state, schedule tree, todo list) |
| `internal/ui/wizard_view.go` | **New** — Day Planner Wizard steps 1-4 |
| `internal/ui/schedule_tree.go` | **New** — Schedule tree widget (cycle groups, color-coded blocks) |
| `internal/ui/todo_list_view.go` | **New** — Todo list with checkboxes, priority, categories |
| `internal/ui/task_detail_modal.go` | **New** — Task detail dialog |
| `internal/ui/planner_view_test.go` | Modify — tests for actual view structure |
| `internal/ui/wizard_view_test.go` | **New** — wizard step tests |
| `cmd/cue/main.go` | Modify — wire timer tick loop, connect PlannerPresenter callbacks to views |

## Dependencies

- Feature-017-Hotfix-A (Timer Renderer): Timer ring must actually draw for the active schedule display to be meaningful
- Existing PlannerPresenter and TimerPresenter are fully tested — this hotfix is purely the Fyne view layer

## Test Coverage

**Center View Switching:**
- NavigateTo(ViewPlan) swaps center content to plan view
- NavigateTo(ViewCharacter) swaps back to character
- NavigateTo(ViewWizard) shows wizard content

**Plan View:**
- No-plan state shows "Plan My Day" button and placeholder message
- Active schedule state shows schedule tree and todo list
- "Abandon Plan" triggers presenter method
- "Plan My Day" navigates to ViewWizard

**Wizard Steps:**
- Step 1 shows task checkboxes from presenter
- Step 2 shows estimate table with override inputs
- Step 3 shows orderable task list
- Step 4 shows two schedule choice cards
- Navigation: Next/Back/Cancel buttons trigger correct presenter methods
- Cancel returns to ViewPlan

**Timer Integration:**
- Tick loop updates timer widget progress
- Tick loop updates flash visibility
- Tick loop updates task label
- Block complete callback fires

**Todo List:**
- Renders todos from repository
- Checkbox toggles completion
- Click opens detail modal
- Add task input creates new todo
