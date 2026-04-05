# Feature 022-Hotfix-A: Center View Router Wiring

**Phase:** Phase-2-Feature-022-Hotfix-A
**Status:** Planned
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Feature 022 delivered the presenter layer (PlannerPresenter, TimerPresenter) with full test coverage and the state machine logic. The center view router exists with ViewCharacter/ViewPlan/ViewWizard states, but `window.go` never swaps the center pane content when the router navigates. This hotfix wires the router so the center column actually switches content.

This is the **critical foundation** for hotfixes 022-B through 022-E — none of the plan view, wizard, or timer wiring can function without center view switching.

## Issue

### Center View Router Not Wired in Window

**Location:** `window.go:62-72`

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

Plan and wizard content will be placeholder containers until 022-B/C/D implement the real views. The key deliverable is that `NavigateTo()` actually swaps what's visible.

## Files

| File | Action |
|---|---|
| `internal/ui/window.go` | Modify — wire center view router to swap center pane content |
| `internal/ui/window_layout_test.go` | Modify — tests for center view switching |

## Dependencies

- None — this is the foundation hotfix

## Test Coverage

- NavigateTo(ViewPlan) swaps center content to plan view placeholder
- NavigateTo(ViewCharacter) swaps back to character content
- NavigateTo(ViewWizard) shows wizard content placeholder
- Default state shows character content on startup
- FocusRail Plan/Back buttons trigger correct router navigation (existing tests)
