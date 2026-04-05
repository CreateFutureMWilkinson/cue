# Feature 017: Day Planner (Scheduling Engine)

**Phase:** Phase-2-Feature-017
**Status:** Planned
**Packages:** `internal/service/planner/`

---

## Overview

Core scheduling engine that generates flexible Pomodoro-style day plans. Takes a prioritized todo list and calendar events as input, produces two candidate schedules (Focus-Maximized and Recovery-Balanced) for user selection. Handles meeting merging, break debt calculation, post-meeting recovery windows, and Ollama-powered task duration estimates. Schedule is persisted to SQLite for survival across restarts. Break debt is a planning-time concept only — not tracked during execution.

## Design Decisions

- **Two schedule options, no computed ranking** — Focus-Maximized minimizes breaks and maximizes contiguous focus time. Recovery-Balanced respects all break intervals even at the cost of fewer Pomodoros. User chooses based on how they feel, not a score.
- **Break debt is planning-time only** — calculated during schedule generation to inform break placement. Once the user selects a schedule, debt is baked into the plan and never recalculated.
- **Meetings are fixed focus windows** — calendar events are immovable. The scheduler wraps Pomodoros around them. No focus/break blocks overlap meetings.
- **Meeting merging** — gaps < 5 minutes between consecutive meetings are absorbed into a single extended meeting block. Prevents micro-breaks that provide no real recovery.
- **Post-meeting breaks scale with meeting length** — ≤30min meeting targets a 5min break; >30min meeting targets a ~20min recovery window. These are best-effort — if no time remains (e.g., near end of day), the break is skipped.
- **Lunch-adjacent break recovery** — accumulated break debt is preferentially repaid by extending a break within the configurable lunch window (default 12:00–14:00).
- **Ollama estimates are advisory** — inference suggests Pomodoro count per task. User can override any estimate. Purpose is visibility into overloaded days, not precision scheduling.
- **Planning horizon auto-switches** — if current time is past the configurable cutoff (default 16:00), the planner targets the next working day instead of today.
- **Schedule persisted to SQLite** — survives app restarts. Only the selected schedule is persisted, not both candidates. Loaded on startup if a schedule exists for the current day.

## Data Model

### TimeBlock

```go
type BlockType int

const (
    BlockFocus      BlockType = iota  // Pomodoro focus window
    BlockShortBreak                    // Short break (default 5min)
    BlockLongBreak                     // Long break (default 20min)
    BlockMeeting                       // Calendar event (fixed)
)

type TimeBlock struct {
    Start    time.Time
    End      time.Time
    Type     BlockType
    TaskID   *uuid.UUID  // nil for breaks and meetings
    TaskName string      // display label; meeting title for BlockMeeting
}
```

### DaySchedule

```go
type DaySchedule struct {
    ID        uuid.UUID
    Date      time.Time
    Strategy  string       // "focus-maximized" | "recovery-balanced"
    Blocks    []TimeBlock
    CreatedAt time.Time
}
```

### TaskEstimate

```go
type TaskEstimate struct {
    TodoID          uuid.UUID
    Title           string
    EstimatedPomos  int     // Ollama-suggested
    UserOverride    *int    // nil = accept Ollama estimate
}
```

## Config

```toml
[planner]
workday_start = "09:00"
workday_end = "17:00"
planning_cutoff = "16:00"
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4
meeting_merge_gap_minutes = 5
lunch_window_start = "12:00"
lunch_window_end = "14:00"
```

## API

### Planner Service

```go
func NewPlanner(cfg PlannerConfig, estimator TaskEstimator, clock Clock) (*Planner, error)
```

### Schedule Generation

```go
// GenerateSchedules produces two candidate schedules from tasks and calendar events.
func (p *Planner) GenerateSchedules(
    ctx context.Context,
    tasks []TaskEstimate,
    events []CalendarEvent,
    targetDate time.Time,
) (focusMaximized *DaySchedule, recoveryBalanced *DaySchedule, error)
```

### Task Estimation

```go
type TaskEstimator interface {
    EstimatePomodoros(ctx context.Context, title string, description string) (int, error)
}
```

### Schedule Persistence

```go
type ScheduleRepository interface {
    Save(ctx context.Context, schedule *DaySchedule) error
    LoadByDate(ctx context.Context, date time.Time) (*DaySchedule, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### Planning Horizon

```go
// TargetDate returns today if before cutoff, or the next working day if after.
func (p *Planner) TargetDate(now time.Time) time.Time
```

### Clock Interface (for testability)

```go
type Clock interface {
    Now() time.Time
}
```

## Scheduling Algorithm

### Focus-Maximized Strategy

1. Place all meetings as fixed blocks.
2. Merge meetings with gaps < `meeting_merge_gap_minutes`.
3. Calculate minimum required breaks (1 short break per 4 Pomodoros as long break).
4. Fill all remaining time with focus blocks, using shortest possible breaks.
5. Skip post-meeting recovery windows when they would reduce focus time.
6. Place the single required long break at the lunch-adjacent window if possible.

### Recovery-Balanced Strategy

1. Place all meetings as fixed blocks.
2. Merge meetings with gaps < `meeting_merge_gap_minutes`.
3. Calculate break debt: after meetings ≤30min → 5min break; after meetings >30min → 20min recovery.
4. Place post-meeting breaks immediately after each meeting block.
5. Fill remaining time with standard Pomodoro cycles (focus + short break, long break every N cycles).
6. Accumulate any unplaceable break debt and repay via extended lunch-adjacent break.
7. If no time remains for a break, skip it (best-effort, not forced).

### Task Assignment

Both strategies assign tasks to focus blocks in priority order:
1. Sort tasks by priority (lower number = higher priority).
2. For each task, allocate `estimatedPomos` consecutive focus blocks where possible.
3. If a task spans a meeting gap, split across the gap (resume after meeting + break).
4. If total task Pomodoros exceed available focus blocks, fill what fits and leave remaining tasks unscheduled (visible to user as "won't fit today").

## Error Handling

| Scenario | Behavior |
|---|---|
| No tasks provided | Return two empty schedules (breaks/meetings only) |
| No calendar events | Schedule is pure Pomodoro cycles for the full workday |
| Ollama estimation fails | Fallback: 1 Pomodoro per task, log warning |
| All time consumed by meetings | Return schedules with zero focus blocks, user sees "no time for tasks" |
| Invalid config (e.g., workday_end before workday_start) | Validation error at config load |
| Target date in the past | Return error (cannot plan past days) |
| Schedule persistence fails | Log error, schedule remains in memory for current session |

## Integration Points

- **Todo List (Feature 015):** Reads incomplete todos as task candidates. New tasks entered during planning are written back to the todo repository.
- **Calendar Adapter (Feature 016):** Fetches `[]CalendarEvent` for the target date to place as fixed blocks.
- **Planner UI (Feature 018):** Presents the two schedule candidates, handles user selection, displays active schedule.
- **Planner Audio (Feature 019):** Timer-end alerts triggered at block boundaries.
- **Config (Feature 001):** New `[planner]` section with validation rules for all timing parameters.
- **Ollama Client (Feature 004):** `TaskEstimator` implementation wraps the existing Ollama client for duration inference.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `planner` | `PlannerSuite` | Constructor validation, target date logic (before/after cutoff), config defaults |
| `planner` | `ScheduleGenerationSuite` | Pure Pomodoro day (no meetings), meetings-only day, mixed day, meeting merging (<5min gap), post-meeting breaks (≤30min, >30min), break debt and lunch recovery, focus-maximized vs recovery-balanced differences, task assignment by priority, task overflow ("won't fit"), all-day event handling |
| `planner` | `TaskEstimationSuite` | Ollama estimate success, Ollama failure fallback (1 pomo), user override applied |
| `sqlite` | `ScheduleRepositorySuite` | Save, load by date, delete, not found, overwrite existing |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Planner Core | RED | Test Designer | — | — | — |
| Planner Core | GREEN | Implementer | — | — | — |
| Planner Core | REFACTOR | Refactorer | — | — | — |
| Schedule Gen | RED | Test Designer | — | — | — |
| Schedule Gen | GREEN | Implementer | — | — | — |
| Schedule Gen | REFACTOR | Refactorer | — | — | — |
| Task Estimation | RED | Test Designer | — | — | — |
| Task Estimation | GREEN | Implementer | — | — | — |
| Task Estimation | REFACTOR | Refactorer | — | — | — |
| Schedule Repo | RED | Test Designer | — | — | — |
| Schedule Repo | GREEN | Implementer | — | — | — |
| Schedule Repo | REFACTOR | Refactorer | — | — | — |
