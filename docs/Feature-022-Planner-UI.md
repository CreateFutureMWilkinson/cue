# Feature 022: Planner UI

**Phase:** Phase-2-Feature-022
**Status:** Done
**Packages:** `internal/ui/presenter/`, `internal/ui/`

---

## Overview

New Fyne pane with a wizard-style workflow for day planning and an active schedule view with a Countdown-style circular timer. The wizard guides the user through task selection, time estimation review, priority ordering, and schedule choice. Once a schedule is selected, the pane switches to an execution view showing the timeline, current block, and burndown timer. Users trigger planning manually via a "Plan My Day" button and can abandon the plan at any time.

## Design Decisions

- **Wizard flow within a single pane** — not a modal dialog. The pane content transitions through wizard steps, keeping the notification queue and activity log visible alongside. Each step replaces the previous in the planner pane area.
- **"Plan My Day" as entry point** — planning is opt-in. The pane shows the button (and any existing schedule) by default. No auto-planning on startup.
- **"Abandon Plan" clears the active schedule** — removes from display and deletes from SQLite. Returns to the idle state with the "Plan My Day" button.
- **Countdown-style timer is a custom Fyne canvas widget** — 45 lines arranged in a ring, drawn via `canvas.Line` objects. Not a standard Fyne widget; requires custom rendering.
- **Presenter/view separation maintained** — `PlannerPresenter` handles wizard state machine and schedule logic. `TimerPresenter` handles countdown state and block transitions. View layer is thin Fyne wiring.
- **Task input during wizard writes back to todo repo** — new tasks entered in the planning wizard are persisted, not ephemeral.
- **Schedule display as vertical timeline** — each block shown as a colored bar with time range and label. Current block highlighted.

## Wizard Steps

### Step 1: Task Selection

- Display all incomplete todos from the todo repository, ordered by priority.
- Each todo shown with title, priority, categories (colored badges), and due date if set.
- User can select/deselect tasks to include in the plan.
- User can add new tasks inline (title + priority minimum) — written to todo repo immediately.
- "Next" button proceeds to estimation.

### Step 2: Time Estimates

- For each selected task, display Ollama-generated Pomodoro estimate.
- User can override any estimate via a numeric input (integer Pomodoros).
- Summary line shows total estimated Pomodoros vs available focus blocks for the day.
- Visual indicator if tasks exceed available time ("Overloaded — N Pomodoros won't fit").
- "Next" proceeds to priority review; "Back" returns to task selection.

### Step 3: Priority Ordering

- Tasks displayed in current priority order.
- User can drag-to-reorder or use up/down buttons to adjust.
- Changes update the todo repo priority field.
- "Next" proceeds to schedule choice; "Back" returns to estimates.

### Step 4: Schedule Choice

- Two schedule cards displayed side-by-side:
  - **Option A: Focus-Maximized** — description of tradeoffs, total focus time, break count.
  - **Option B: Recovery-Balanced** — description of tradeoffs, total focus time, break count.
- Each card shows a mini-timeline preview of the day.
- No computed ranking or recommendation — user chooses.
- Selecting a card persists the schedule and transitions to the active view.

### Step 5: Active Schedule View

- Vertical timeline showing all blocks for the day.
- Current block highlighted with distinct background.
- Countdown timer widget displayed prominently.
- Current task name shown below the timer.
- "Complete Task" button — marks the current task as done, rolls highest-priority incomplete task into next focus block.
- "Abandon Plan" button — clears schedule, returns to idle state.

## Countdown Timer Widget

### Geometry

- 45 lines arranged in a circle, radiating inward from the outer edge of a ring.
- Lines are evenly spaced at 8° intervals (360° / 45 = 8°).
- Starting position: 8° clockwise from 12 o'clock (vertical up).
- Final position: 0° (12 o'clock / vertical up) — the last time segment.
- Progression: clockwise.

### Line Lengths (Three Tiers)

| Position | Angle from vertical | Count | Length |
|---|---|---|---|
| Cardinal (N, E, S, W) | 0°, 90°, 180°, 270° | 4 | 3× short |
| Diagonal (NE, SE, SW, NW) | 45°, 135°, 225°, 315° | 4 | 2× short |
| All other lines | remaining positions | 37 | 1× short (base) |

### Color and Animation

- All lines: yellow `#FFCE1B`.
- The **current segment line** flashes at 1 Hz (visible/invisible toggle, 500ms on / 500ms off).
- Each line represents ~2.2222% of the total block duration (100% / 45).
- When a segment's time elapses, the next clockwise line becomes the active (flashing) indicator.
- Inactive (elapsed) lines are hidden or dimmed to show burndown progress.

### Behavior

- Timer resets at the start of each new block (focus, break, or meeting).
- During meetings: timer runs but no end-of-block sound (Feature 023).
- On block completion: timer stops, alert fires (unless meeting), next block begins.

## API

### PlannerPresenter

```go
type PlannerPresenter struct {
    // wizard state, schedule data, todo/calendar/planner deps
}

func NewPlannerPresenter(
    todos TodoQuerier,
    categories CategoryQuerier,
    calendar CalendarProvider,
    planner ScheduleGenerator,
    estimator TaskEstimator,
    scheduleRepo ScheduleRepository,
    clock Clock,
) (*PlannerPresenter, error)

// Wizard navigation
func (p *PlannerPresenter) StartPlanning(ctx context.Context) error
func (p *PlannerPresenter) CurrentStep() WizardStep
func (p *PlannerPresenter) NextStep(ctx context.Context) error
func (p *PlannerPresenter) PreviousStep()

// Step 1: Task selection
func (p *PlannerPresenter) AvailableTasks() []TodoRow
func (p *PlannerPresenter) SelectTask(id uuid.UUID, selected bool)
func (p *PlannerPresenter) AddTask(ctx context.Context, title string, priority int) error

// Step 2: Estimates
func (p *PlannerPresenter) Estimates() []TaskEstimateRow
func (p *PlannerPresenter) OverrideEstimate(todoID uuid.UUID, pomos int)
func (p *PlannerPresenter) EstimateSummary() EstimateSummary

// Step 3: Priority
func (p *PlannerPresenter) ReorderTask(fromIndex, toIndex int)

// Step 4: Schedule choice
func (p *PlannerPresenter) FocusSchedule() *SchedulePreview
func (p *PlannerPresenter) RecoverySchedule() *SchedulePreview
func (p *PlannerPresenter) SelectSchedule(ctx context.Context, strategy string) error

// Step 5: Active schedule
func (p *PlannerPresenter) ActiveSchedule() *ActiveScheduleState
func (p *PlannerPresenter) CompleteCurrentTask(ctx context.Context) error
func (p *PlannerPresenter) AbandonPlan(ctx context.Context) error

// Lifecycle
func (p *PlannerPresenter) HasActivePlan() bool
func (p *PlannerPresenter) LoadExistingPlan(ctx context.Context) error
```

### TimerPresenter

```go
type TimerPresenter struct {
    // countdown state, block tracking
}

func NewTimerPresenter(clock Clock, alerter TimerAlerter) (*TimerPresenter, error)

func (p *TimerPresenter) Start(block TimeBlock)
func (p *TimerPresenter) Stop()
func (p *TimerPresenter) ActiveSegment() int          // 0–44, which line is flashing
func (p *TimerPresenter) ElapsedFraction() float64     // 0.0–1.0
func (p *TimerPresenter) IsFlashVisible() bool         // current flash state (1Hz)
func (p *TimerPresenter) CurrentTaskName() string
func (p *TimerPresenter) BlockType() BlockType
func (p *TimerPresenter) IsRunning() bool
func (p *TimerPresenter) SetOnTick(fn func())          // UI refresh callback
func (p *TimerPresenter) SetOnBlockComplete(fn func()) // block-end callback
```

### WizardStep Enum

```go
type WizardStep int

const (
    StepIdle        WizardStep = iota  // "Plan My Day" button shown
    StepTaskSelect                      // Step 1
    StepEstimates                       // Step 2
    StepPriority                        // Step 3
    StepSchedule                        // Step 4
    StepActive                          // Step 5: execution view
)
```

## UI Placement

The planner pane replaces or sits alongside the existing layout. Suggested placement as a tab or switchable view:

```
┌───────────────────────────┬──────────────────────────────────────┐
│                           │         Activity Log                 │
│   Notification Queue      │         (scrollable list)            │
│                           │                                      │
│                           ├──────────────────────────────────────┤
│                           │   [Character Widget]                 │
├───────────────────────────┴──────────────────────────────────────┤
│  [Notifications] [Day Planner] [Review Buffered]  ← tab bar     │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Day Planner Pane (wizard steps or active schedule view)        │
│                                                                  │
│   ┌─────────────────┐  ┌──────────────────────────────────┐     │
│   │   Countdown      │  │  Timeline                        │     │
│   │   Timer Ring     │  │  ┌──────────────────────────┐    │     │
│   │                  │  │  │ 09:00 ██ Focus: Task A    │    │     │
│   │   ◯ (45 lines)  │  │  │ 09:25 ░░ Short break     │    │     │
│   │                  │  │  │ 09:30 ██ Focus: Task A    │    │     │
│   │  Current Task:   │  │  │ 10:00 ▓▓ Meeting: Standup│    │     │
│   │  "Write report"  │  │  │ 10:15 ░░ Short break     │    │     │
│   │                  │  │  │ ...                        │    │     │
│   │  [Complete Task] │  │  └──────────────────────────┘    │     │
│   │  [Abandon Plan]  │  │                                  │     │
│   └─────────────────┘  └──────────────────────────────────┘     │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Calendar fetch fails during wizard | Proceed without calendar events, log warning, show "No calendar data" in wizard |
| Ollama estimation fails | Fallback to 1 Pomodoro per task, show warning in estimates step |
| Todo repo query fails | Show error in pane, wizard cannot proceed |
| Schedule save fails | Log error, schedule remains in memory for current session |
| No tasks selected | "Next" disabled, show hint to select at least one task |
| All time consumed by meetings | Show "No time for tasks" in schedule preview |
| Existing plan on startup | Auto-load and display in active view (Step 5) |

## Integration Points

- **Todo List (Feature 015):** TodoQuerier and CategoryQuerier for task selection; todo repo writes for new tasks and priority updates.
- **Calendar Adapter (Feature 020):** CalendarProvider for fetching meeting blocks.
- **Day Planner (Feature 021):** ScheduleGenerator for producing schedule candidates; ScheduleRepository for persistence.
- **Planner Audio (Feature 023):** TimerAlerter for block-end sounds; meeting suppression communicated via BlockType.
- **Three-Column Layout (Feature 016):** Integrates into the center area column via the center view router.
- **Character (Feature 014):** Character state could reflect planner activity (working during focus, idle during breaks) — optional enhancement.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `presenter` | `PlannerPresenterSuite` | Constructor validation, wizard step navigation (forward/back), task selection, add new task, estimate override, estimate summary (overloaded detection), priority reorder, schedule selection, active schedule state, complete task, abandon plan, load existing plan, calendar failure fallback, estimation failure fallback |
| `presenter` | `TimerPresenterSuite` | Start/stop, segment progression (0–44), elapsed fraction, flash visibility toggle (1Hz), block complete callback, tick callback, reset on new block, meeting block (no alert flag) |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Planner Presenter | RED | Test Designer | 430s | 43,627 | bcdbf5d |
| Planner Presenter | GREEN | Implementer | 118s | 51,842 | 9da51be |
| Planner Presenter | REFACTOR | Refactorer | 175s | 58,853 | 9b21936 |
| Timer Presenter | RED | Test Designer | 95s | 41,532 | f77f2a3 |
| Timer Presenter | GREEN | Implementer | 10,499s | 37,359 | 3f81206 |
| Timer Presenter | REFACTOR | Refactorer | 80s | 33,241 | 2433b69 |
| Planner View | RED | Test Designer | 80s | 38,986 | bcdace1 |
| Planner View | GREEN | Implementer | 51s | 39,040 | 5a377b3 |
| Planner View | REFACTOR | Refactorer | 91s | 34,554 | 86fcf4a |
