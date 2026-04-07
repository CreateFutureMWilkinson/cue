# Feature 063: PlannerView Missing Widget Tree

**Phase:** Phase-6-Feature-063
**Type:** Bugfix
**Severity:** High
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 022B (Plan View), Feature 022C (Todo List View), Feature 056 (Plan/Wizard Wiring), Feature 052 (Automated UI Testing)

---

## Bug Description

`PlannerView.Container()` returns a flat `container.NewVBox` containing only five raw buttons. The acceptance tests expect a **horizontal `container.Split`** with plan overview (leading) and todo list (trailing). Additionally, `TodoListView.Container()` returns an empty `container.NewVBox()` — no task rows, no Entry field, no "Add" button are ever rendered.

## Expected Behavior

Per UiSpec.md (lines 1079-1121):
- Plan view is split 50/50 horizontally: Plan Overview (left) + Todo List (right)
- Todo list has inline task creation at bottom: title field, priority field, Add button
- Task rows display with checkboxes, titles, priority, category badges

## Actual Behavior

- `PlannerView`: `container.NewVBox(planBtn, nextBtn, backBtn, completeTaskBtn, abandonBtn)` — no Split, no todo section
- `TodoListView`: `container.NewVBox()` — empty, never populated with widgets
- Both views track state in struct fields and expose accessor methods, but never render actual Fyne widgets into their containers

## Root Cause

Features 022B and 022C implemented the data model and accessor methods but stopped short of rendering widgets into containers. Feature 056 wired the views into `MainWindow` but didn't address the missing widget trees.

## Failing Acceptance Tests

| Test | What It Expects |
|---|---|
| `TestPlanViewContainsSplit` | Horizontal `container.Split` with Leading + Trailing |
| `TestPlanViewContainsTodoSection` | Horizontal split exists (same widget, different assertion) |
| `TestTodoListHasInlineCreation` | `*widget.Button` with Text=="Add" in tree |
| `TestTodoListHasEntryFields` | At least one `*widget.Entry` in tree |

## Fix

### TodoListView — render widgets into container

1. Add `titleEntry *widget.Entry` and `addBtn *widget.Button` ("Add") as struct fields, initialized in constructor.
2. Add `buildContainer()` method called from `NewTodoListView` and `Refresh`:
   - Clear `v.container.Objects`
   - For each item in `v.items`: add a `widget.NewCheck(title, onChanged)` to the container
   - Append inline creation row: `container.NewHBox(v.titleEntry, v.addBtn)` at the bottom
3. "Add" button calls `v.AddItem(v.titleEntry.Text, 0)`, clears the entry, and calls `buildContainer()`.

### PlannerView — horizontal Split layout

1. Add `todoList *TodoListView` field (or accept pre-built `*TodoListView` / `TodoListViewModel` in constructor).
2. Replace flat VBox with:
   - Leading pane: `container.NewVBox(planBtn, nextBtn, backBtn, completeTaskBtn, abandonBtn)` (existing buttons)
   - Trailing pane: `v.todoList.Container()` (or empty container if nil)
   - Split: `container.NewHSplit(leading, trailing)` — produces `*container.Split` with `Horizontal = true`
3. Wrap split in `container.NewStack(split)` since `Container()` returns `*fyne.Container` and `*container.Split` is not `*fyne.Container`.
4. Nil-safe: if no TodoListViewModel provided, trailing pane is an empty `container.NewVBox()`.

### Constructor signature change

`NewPlannerView` gains a `TodoListViewModel` parameter (or pre-built `*TodoListView`). All call sites updated. Existing accessor methods (`PlanButton()`, `AbandonButton()`, etc.) remain unchanged — buttons are relocated into the split's leading pane.

## Files to Modify

| File | Change |
|---|---|
| `internal/ui/todo_list_view.go` | Add `titleEntry`, `addBtn` fields; `buildContainer()` renders task rows + inline creation |
| `internal/ui/todo_list_view_test.go` | Unit tests for Entry, Add button, and task row widgets in container |
| `internal/ui/planner_view.go` | Replace VBox with HSplit(leading, trailing) wrapped in Stack |
| `internal/ui/planner_view_test.go` | Update for new layout; add TodoListViewModel mock |
| `tests/ui/helpers_test.go` | Update `stubPlannerTimerVM` or add todo VM to `newMainWindowWithPlanner` |

## TDD Behaviors (Micro-Loops)

1. **TodoListView renders inline creation widgets** — Entry field + "Add" button in container
2. **TodoListView renders task rows** — Check widget per item above creation row
3. **PlannerView uses horizontal Split** — HSplit with buttons (leading) + todo list (trailing)

## Risk Areas

- **Constructor signature change** ripples to unit tests, acceptance test helpers, and `window.go`
- **`*container.Split` vs `*fyne.Container`** — must wrap with `container.NewStack()`
- **Existing unit tests** call accessor methods on buttons — these still work since buttons are struct fields, just repositioned in the tree

## TDD Agent Stats

_To be filled during implementation._
