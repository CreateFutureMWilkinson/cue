# Feature 022-Hotfix-A: Center View Router Wiring

**Phase:** Phase-2-Feature-022-Hotfix-A
**Status:** Done
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Feature 022 delivered the presenter layer (PlannerPresenter, TimerPresenter) with full test coverage and the state machine logic. The center view router existed with ViewCharacter/ViewPlan/ViewWizard states, but `window.go` never swapped the center pane content when the router navigated. This hotfix wires the router so the center column actually switches content.

This is the **critical foundation** for hotfixes 022-B through 022-E — none of the plan view, wizard, or timer wiring can function without center view switching.

## Design Decisions

### Multi-listener Router Pattern

The `CenterViewRouter` originally supported a single callback via `SetOnViewChange`. FocusRail uses this to toggle Plan/Back button visibility. Adding a second `SetOnViewChange` call in `window.go` would overwrite FocusRail's callback.

**Solution:** Added `AddOnViewChange(fn)` which appends to a separate `listeners` slice. `SetOnViewChange` continues to set a single primary callback (backward compatible). `NavigateTo` fires the primary callback first, then all additional listeners. This allows FocusRail and window.go to coexist without interference.

### View Content Map

A `map[CenterView]fyne.CanvasObject` maps each view state to its content widget. This avoids switch statements and makes it trivial to add new views later (022-B through 022-E just replace the placeholder labels with real content).

### container.NewStack for Content Swapping

The center pane uses a Fyne `Stack` container with a single child. On navigation, the stack's `Objects` slice is replaced and refreshed. This is the lightest-weight approach — no custom layout, no show/hide complexity.

### CenterContent() Derives from Router State

Rather than maintaining a separate `centerContent` field that could drift out of sync, `CenterContent()` reads `viewRouter.CurrentView()` and looks up the corresponding content in the view map. Single source of truth.

## API

### CenterViewRouter (modified)

```go
// AddOnViewChange appends a listener without replacing the primary callback.
func (r *CenterViewRouter) AddOnViewChange(fn func(CenterView))
```

### MainWindow (modified)

```go
// CenterContent returns the canvas object currently displayed in the center column.
func (m *MainWindow) CenterContent() fyne.CanvasObject
```

## Error Handling

- Nil `viewRouter`: `CenterContent()` returns nil; no listener registered.
- Unknown view in map lookup: no-op (content unchanged).

## Files Changed

| File | Action |
|---|---|
| `internal/ui/center_view_router.go` | Added `listeners` field and `AddOnViewChange` method |
| `internal/ui/window.go` | Wired center view router, added `CenterContent()`, `switchCenterView()` |
| `internal/ui/window_layout_test.go` | 4 new tests for center view switching |

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestCenterViewDefaultsToCharacterContent` | Center pane shows character content on startup |
| `TestNavigateToPlanSwapsCenterContent` | NavigateTo(ViewPlan) swaps to plan placeholder |
| `TestNavigateToWizardSwapsCenterContent` | NavigateTo(ViewWizard) swaps to wizard placeholder |
| `TestNavigateBackToCharacterRestoresContent` | Round-trip navigation restores character content |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 46s | 23,628 | f4d05ef |
| GREEN | Implementer | 84s | 27,795 | 522313f |
| REFACTOR | Refactorer | 112s | 27,922 | 6174019 |
