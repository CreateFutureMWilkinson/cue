# Feature 022-Hotfix-C: Todo List View + Task Detail Modal

**Phase:** Phase-2-Feature-022-Hotfix-C
**Status:** Planned
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Implements the right half of the Plan view — the todo list with checkboxes, priority indicators, category badges, due dates, inline task creation, and a task detail modal for editing. This is an independent widget that occupies the right 50% of the Plan view's horizontal split.

## Requirements (from UI-SPEC.md)

### Todo List

Displays all incomplete tasks from the todo repository, plus completed tasks for the current day.

Each task shows:
- **Row 1:** Checkbox + task title
- **Row 2:** Priority (`P:{N}`), category badges (colored), optional due date
- **Row 3:** `[details]` link — opens Task Detail Modal

Completed tasks shown with strikethrough text and reduced opacity.

**Sort order:** Incomplete first, then by priority (ascending P:1 before P:2), then by creation date.

### Inline Task Creation

Input row pinned at bottom:
- Task title entry field
- Priority number entry field
- Add button — writes to todo repository, adds to list

### Task Detail Modal

Modal dialog (500w × 450h) blocking main window input:
- Title (editable entry)
- Priority (integer entry)
- Category (free-text entry)
- Due Date (ISO date `YYYY-MM-DD`, optional)
- Notes (multiline entry, new field)
- Save button — persist to todo repo, close modal, refresh list
- Cancel button — discard changes, close modal
- Close (X) — same as Cancel

## Files

| File | Action |
|---|---|
| `internal/ui/todo_list_view.go` | **New** — Todo list widget with checkboxes, priority, categories |
| `internal/ui/task_detail_modal.go` | **New** — Task detail dialog |
| `internal/ui/todo_list_view_test.go` | **New** — todo list rendering and interaction tests |
| `internal/ui/task_detail_modal_test.go` | **New** — modal field and save/cancel tests |

## Dependencies

- 022-A (center view router wiring) — plan view must be displayable
- 022-B (plan view) — todo list is the right half of the plan view HSplit

## Test Coverage

**Todo List:**
- Renders todos from repository
- Shows checkbox, title, priority, categories, due date per task
- Completed tasks shown with reduced opacity
- Sorted: incomplete first, then priority ascending, then creation date
- Checkbox toggle marks task complete/incomplete in todo repo
- `[details]` link opens task detail modal

**Inline Task Creation:**
- Add button creates new task in todo repo
- New task appears in list after creation
- Empty title prevented

**Task Detail Modal:**
- Pre-fills all fields from existing task data
- Save persists changes to todo repo and closes modal
- Cancel discards changes and closes modal
- Modal size is 500w × 450h
- Notes field is multiline
