# Feature 022-Hotfix-B: Plan View (Schedule Tree + No-Plan State)

**Phase:** Phase-2-Feature-022-Hotfix-B
**Status:** Done
**Package:** `internal/ui/`
**Parent:** Feature 022 (Planner UI)

---

## Overview

Implements the left half of the Plan view — the schedule tree (active plan state) and the no-plan placeholder state. The previous `planner_view.go` contained only 5 buttons in a VBox with empty callbacks; this hotfix adds content rendering for both plan states.

## Design Decisions

### PlannerView Content State

The `PlannerView` now maintains two content fields:
- `placeholderText string` — randomly selected from 7 motivational/passive-aggressive messages on view creation (or Refresh)
- `scheduleTree *ScheduleTree` — built from `ActiveScheduleState.Blocks` when an active plan exists

Content is rebuilt via `buildContent()` on construction and on each `Refresh()` call, reading from the `PlannerViewModel` interface.

### ScheduleTree as Separate Type

The schedule tree is a standalone `ScheduleTree` type (not embedded in PlannerView) to enable independent testing of cycle grouping, elapsed pruning, and bar scaling logic. It takes `[]TimeBlockPreview` and `time.Time` (now) and produces `[]ScheduleCycle`.

### Cycle Grouping

Blocks are grouped into cycles by splitting at long break boundaries. A long break is the **last** block of its cycle — the next focus block after a long break starts a new cycle. This matches how the planner engine generates schedules (N focus blocks + short breaks, then a long break).

### Elapsed Block Pruning

Blocks with `End <= now` are pruned. Fully elapsed cycles (all blocks pruned) are removed from the result but their original numbering is preserved (e.g., if cycle 1 is fully elapsed, cycle 2 still shows as "Cycle 2/3").

### Bar Width Scaling

The longest remaining (non-elapsed) block across all cycles gets `BarWidth = 1.0`. All other blocks are proportional: `duration / maxDuration`.

### Router Integration

`NewPlannerView` now accepts a `*CenterViewRouter` as its third parameter. The "Plan My Day" button callback calls `router.NavigateTo(ViewWizard)` to switch the center column to the wizard view.

## API

### PlannerView (modified)

```go
func NewPlannerView(plannerModel PlannerViewModel, timerModel TimerViewModel, router *CenterViewRouter) *PlannerView
func (v *PlannerView) PlaceholderText() string    // "" when active plan
func (v *PlannerView) ScheduleTree() *ScheduleTree // nil when no plan
```

### ScheduleTree (new)

```go
func NewScheduleTree(blocks []presenter.TimeBlockPreview, now time.Time) *ScheduleTree
func (t *ScheduleTree) Cycles() []ScheduleCycle

type ScheduleCycle struct {
    Number int              // Original cycle number (1-indexed)
    Total  int              // Total cycles before pruning
    Blocks []ScheduleBlockRow
}

type ScheduleBlockRow struct {
    StartTime    string      // "HH:MM"
    Title        string      // "Focus", "Meeting: {name}", "Short Break", "Long Break"
    DurationText string      // "{N}m"
    BarWidth     float32     // 0.0–1.0 proportional to longest block
    Color        color.Color // Green/light blue/blue/amber by type
}
```

## Block Colors

| Type | Color | RGBA |
|---|---|---|
| Focus | Green | (76, 175, 80, 255) |
| Short Break | Light Blue | (129, 212, 250, 255) |
| Long Break | Blue | (66, 165, 245, 255) |
| Meeting | Amber | (255, 193, 7, 255) |

## Error Handling

- Nil router: "Plan My Day" button does nothing (no-op guard)
- Nil ActiveSchedule or empty blocks: ScheduleTree remains nil
- All blocks elapsed: Cycles() returns empty slice

## Test Coverage

| Area | Tests |
|---|---|
| No-plan state | 5 (placeholder message, plan button, abandon hidden, tree nil, navigation) |
| Active plan state | 4 (tree non-nil, plan button hidden, abandon visible, placeholder empty) |
| Refresh transition | 1 (idle → active content update) |
| Cycle grouping | 3 (multi-cycle, single cycle, numbering) |
| Block rendering | 4 (start time, focus title, meeting title, break titles) |
| Elapsed pruning | 3 (block pruning, cycle pruning, partial cycle) |
| Bar scaling | 2 (proportional width, duration text) |
| Block colors | 1 (all 4 types) |
| **Total** | **23 new tests** |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 210s | 63,915 | 522861a |
| GREEN | Implementer | 158s | 51,781 | 1d0fd7f |
| REFACTOR | Refactorer | 85s | 27,760 | a93a5da |
