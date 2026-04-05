# Feature 022-Hotfix-B: Plan View (Schedule Tree + No-Plan State)

**Phase:** Phase-2-Feature-022-Hotfix-B
**Status:** Planned
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Implements the left half of the Plan view — the schedule tree (active plan state) and the no-plan placeholder state. The current `planner_view.go` contains only 5 buttons in a VBox with empty callbacks; this hotfix replaces it with the actual Plan Overview layout per UI-SPEC.md.

## Requirements (from UI-SPEC.md)

### No-Plan State

When no active schedule exists, the plan overview (left half) shows:
- One of 7 motivational/passive-aggressive placeholder messages (randomly selected once on view load)
- A "Plan My Day" button (centered, prominent) that navigates to ViewWizard

### Active Schedule State

When a schedule exists, the plan overview shows a schedule tree:
- Grouped by cycle: "Cycle 1/4", "Cycle 2/4", etc.
- Each block as a row: start time (HH:MM), colored bar proportional to duration, duration text overlaid on bar
- Block types: Focus (green), Short break (light blue), Long break (blue), Meeting (amber)
- Focus blocks titled "Focus" only (no task name — task shown in focus rail)
- Meeting blocks titled "Meeting: {event name}"
- Elapsed blocks (end time in past) pruned; first visible block is always current
- Fully elapsed cycles also pruned
- Bar width auto-scales: longest remaining block = full panel width
- "Abandon Plan" button at bottom

### Placeholder Messages

- "Who even knows"
- "It's your time you're wasting"
- "A goal without a plan is just a wish"
- "Winging it, are we?"
- "The plan is there is no plan"
- "Chaos is also a strategy, I suppose"
- "Bold of you to go planless"

## Files

| File | Action |
|---|---|
| `internal/ui/planner_view.go` | Rewrite — Plan Overview with no-plan state and schedule tree |
| `internal/ui/schedule_tree.go` | **New** — Schedule tree widget (cycle groups, color-coded bars) |
| `internal/ui/planner_view_test.go` | Rewrite — tests for plan overview states |
| `internal/ui/schedule_tree_test.go` | **New** — schedule tree rendering tests |

## Dependencies

- 022-A (center view router wiring) — plan view must be displayable in center column

## Test Coverage

**No-Plan State:**
- Shows one of 7 placeholder messages
- Shows "Plan My Day" button
- "Plan My Day" triggers navigation to ViewWizard

**Active Schedule State:**
- Shows schedule tree grouped by cycles
- Blocks display correct colors by type
- Elapsed blocks are pruned
- Fully elapsed cycles are pruned
- Bar widths proportional to duration
- Duration text overlaid on bars
- "Abandon Plan" triggers presenter method
- "Abandon Plan" returns to no-plan state

**Schedule Tree Widget:**
- Renders cycle headers with correct numbering
- Renders block rows with start time, bar, duration
- Color-codes bars by block type
- Focus blocks show "Focus" title (no task name)
- Meeting blocks show "Meeting: {event name}"
