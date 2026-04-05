# Feature 022-Hotfix-D: Day Planner Wizard Steps 1-4

**Phase:** Phase-2-Feature-022-Hotfix-D
**Status:** Planned
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

The PlannerPresenter has all wizard step transitions tested, but no Fyne UI exists for steps 1-4. This hotfix implements the wizard view that renders in the center column when `ViewWizard` is active, with step-specific content driven by the PlannerPresenter state machine.

## Requirements (from UI-SPEC.md)

### Step 1: Task Selection

- Step indicator: "Step 1 of 4"
- Date label: target planning date
- Checkbox list of todos from repository with category badges
- Inline "Add Task" field (title + priority + Add button)
- "Next" button (enabled when ≥1 task selected)
- "Cancel" returns to Plan view (ViewPlan)

### Step 2: Pomodoro Estimates

- Available Pomodoros label (calculated from calendar gaps)
- Table: task name | visual dots + estimated pomos | override input
- Total summary: "X of Y Pomodoros" (updates live)
- Overload warning (orange text) when total > available
- "Back" / "Next" buttons

### Step 3: Priority Ordering

- Instruction text: "Drag to reorder or use arrows"
- Numbered task list with estimated pomos and up/down buttons
- "Tasks are scheduled in this order" hint
- "Back" / "Next" buttons

### Step 4: Schedule Choice

- Two schedule cards side by side in HBox:
  - Strategy name ("A: Focus-Maximized" / "B: Recovery-Balanced")
  - Stats: focus block count, break count, total focus time
  - Mini-timeline: horizontal bar with color-coded blocks
  - Tradeoff text (plain description)
  - "Select" button per card
- "Back" button

### Navigation

- All steps: "Back" returns to previous step
- Step 1: "Cancel" returns to ViewPlan
- Step 4: "Select" persists schedule to SQLite, returns to ViewPlan (which now shows schedule tree)

## Files

| File | Action |
|---|---|
| `internal/ui/wizard_view.go` | **New** — Day Planner Wizard steps 1-4 |
| `internal/ui/wizard_view_test.go` | **New** — wizard step rendering and navigation tests |

## Dependencies

- 022-A (center view router wiring) — wizard must display in center column via ViewWizard
- PlannerPresenter (Feature 022) — wizard delegates all state transitions to presenter

## Test Coverage

**Step 1 — Task Selection:**
- Shows task checkboxes from presenter's AvailableTasks()
- Categories displayed as colored badges
- Inline Add Task creates todo via presenter
- "Next" disabled when no tasks selected
- "Next" advances to step 2 via presenter.NextStep()
- "Cancel" navigates to ViewPlan

**Step 2 — Estimates:**
- Shows estimate table from presenter's Estimates()
- Override input calls presenter.OverrideEstimate()
- Summary updates from presenter.EstimateSummary()
- Overload warning visible when Overloaded=true
- "Back" calls presenter.PreviousStep()
- "Next" advances to step 3

**Step 3 — Priority:**
- Shows numbered task list with up/down buttons
- Up/down calls presenter.ReorderTask()
- "Back" / "Next" navigate correctly

**Step 4 — Schedule Choice:**
- Shows two cards from presenter's FocusSchedule() and RecoverySchedule()
- Each card displays stats and mini-timeline
- "Select A" calls presenter.SelectSchedule("focus-maximized")
- "Select B" calls presenter.SelectSchedule("recovery-balanced")
- After selection, navigates to ViewPlan
- "Back" returns to step 3
