# UI Specification — Cue

This document is the authoritative design specification for Cue's Fyne desktop GUI. Claude implements UI features from this spec. ASCII wireframes define layout; design tokens define styling; component specs define behavior.

---

## Overall Layout

The main window uses a tab bar at the bottom to switch between Notifications view (Phase 1) and Day Planner view (Phase 2). The top split (notification queue + activity log) is always visible; the bottom area switches content.

```
┌──────────────────────────────────────────────────────────────────┐
│  Cue  [Settings] [About] [Quit]                        Menu Bar │
├───────────────────────────┬──────────────────────────────────────┤
│                           │                                      │
│   Notification Queue      │         Activity Log                 │
│   (scrollable list)       │         (scrollable list)            │
│                           │                                      │
│   [Src] Sender | Chan |   │   [HH:MM:SS] Source: Message         │
│         Message Preview   │   [HH:MM:SS] Source: Message         │
│   ─────────────────────   │   [HH:MM:SS] Source: Error!    (red) │
│   [Src] Sender | Chan |   │   [HH:MM:SS] Source: Message         │
│         Message Preview   │                                      │
│   ─────────────────────   │                                      │
│   [Src] Sender | Chan |   │                                      │
│         Message Preview   │                                      │
│                           │                                      │
│                           ├──────────────────────────────────────┤
│                           │   [Character Widget] (opt-in)        │
├───────────────────────────┴──────────────────────────────────────┤
│  [Notifications] [Day Planner] [Review Buffered]    ← tab bar   │
├──────────────────────────────────────────────────────────────────┤
│  Tab content area (see pane specs below)                         │
└──────────────────────────────────────────────────────────────────┘

  Split: 50/50 horizontal (HSplit) for top area
  Window default: 1200w × 800h (from config.toml)
  Character widget: below activity log (right pane), visible when gui.character != "none"
  Tab bar: bottom of window, switches between Notifications (default), Day Planner, Review Buffered
```

---

## Design Tokens

### Colors

| Token              | Value                        | Usage                        |
|--------------------|------------------------------|------------------------------|
| `error-text`       | `RGBA(255, 80, 80, 255)`    | Activity log error entries   |
| `normal-text`      | `color.White`                | Activity log normal entries  |
| `background`       | Fyne theme default           | All pane backgrounds         |
| `button`           | Fyne theme default           | All buttons                  |
| `timer-line`       | `#FFCE1B`                    | Countdown timer lines        |
| `timer-line-dim`   | `RGBA(255, 206, 27, 64)`    | Countdown timer elapsed lines|
| `block-focus`      | `RGBA(76, 175, 80, 200)`    | Timeline focus block         |
| `block-short-break`| `RGBA(144, 202, 249, 200)`  | Timeline short break block   |
| `block-long-break` | `RGBA(100, 181, 246, 200)`  | Timeline long break block    |
| `block-meeting`    | `RGBA(255, 183, 77, 200)`   | Timeline meeting block       |
| `block-current`    | `RGBA(255, 255, 255, 40)`   | Current block highlight      |
| `overload-warning` | `RGBA(255, 152, 0, 255)`    | Overload warning text        |
| `category-badge`   | Per-category color           | Task category badges         |

### Typography

| Token              | Value                        | Usage                        |
|--------------------|------------------------------|------------------------------|
| `log-entry`        | Fyne default monospace       | Activity log entries         |
| `label`            | Fyne default                 | All labels, list items       |
| `detail-heading`   | Fyne default bold            | Dialog headings              |

### Spacing & Sizing

| Token                    | Value    | Usage                                  |
|--------------------------|----------|----------------------------------------|
| `truncate-source`        | 15 chars | Notification row source column         |
| `truncate-sender`        | 15 chars | Notification row sender column         |
| `truncate-channel`       | 15 chars | Notification row channel column        |
| `truncate-preview`       | 80 chars | Notification row message preview       |
| `activity-max-entries`   | 500      | Activity log circular buffer           |
| `feedback-window-width`  | 600      | Feedback review modal width            |
| `feedback-window-height` | 400      | Feedback review modal height           |
| `settings-window-width`  | 400      | Settings modal width                   |
| `settings-window-height` | 280      | Settings modal height                  |
| `refresh-interval`       | 30s      | Canvas refresh interval                |
| `timer-segments`         | 45       | Countdown timer line count             |
| `timer-flash-hz`         | 1        | Countdown timer flash rate (Hz)        |
| `timer-ring-radius`      | 120      | Countdown timer outer radius (px)      |
| `timer-line-short`       | 12       | Countdown timer short line length (px) |
| `timer-line-medium`      | 24       | Countdown timer 2× line length (px)    |
| `timer-line-long`        | 36       | Countdown timer 3× line length (px)    |
| `timeline-block-height`  | 40       | Schedule timeline row height (px)      |

---

## Pane 1: Notification Queue (Top-Left)

### Purpose

Displays NOTIFIED messages sorted newest-first. Each row is a summary; clicking expands to a detail dialog.

### Layout

```
┌─────────────────────────────────────────┐
│  Notification Queue                      │
│                                          │
│  [slack___] JohnDoe_______ | #general__ |│
│            Hey @user, the deploy is...   │
│  ─────────────────────────────────────── │
│  [email__] alice@exam_____ | Inbox_____ |│
│            URGENT: Server down, need...  │
│  ─────────────────────────────────────── │
│  [slack___] bot____________ | #alerts__ |│
│            You were added to #alerts     │
│                                          │
└─────────────────────────────────────────┘
```

### Widget

`widget.List` with `widget.Label` item template.

### Row Format

```
[{Source, 15ch}] {Sender, 15ch} | {Channel, 15ch} | {Preview, 80ch}
```

All fields independently truncated to their max width.

### Interactions

| Action          | Behavior                                                |
|-----------------|---------------------------------------------------------|
| Click row       | Open detail dialog (see below)                          |
| List refresh    | Triggered by 30s canvas tick or manual action           |

### Detail Dialog

```
┌──────────────────────────────────────┐
│  Message Detail                       │
│                                       │
│  Importance Score: 8.5                │
│  Confidence Score: 0.92               │
│  Created: 2026-03-28 14:32:05         │
│                                       │
│  ┌────────────────────────────────┐   │
│  │ Full message content here,     │   │
│  │ word-wrapped, no truncation.   │   │
│  └────────────────────────────────┘   │
│                                       │
│              [ Resolve ]              │
└──────────────────────────────────────┘
```

| Action          | Behavior                                                |
|-----------------|---------------------------------------------------------|
| Click Resolve   | Mark message as Resolved, remove from notification list |

---

## Pane 2: Activity Log (Top-Right)

### Purpose

Real-time feed of system events. Errors render in red; everything else in white.

### Layout

```
┌─────────────────────────────────────────┐
│  Activity Log                            │
│                                          │
│  [14:32:05] Slack: Fetched 12 messages   │
│  [14:32:06] Router: 8 NOTIFIED, 3 BUF..│
│  [14:32:06] Ollama: inference took 250ms │
│  [14:32:15] Email: connection error...   │  ← red
│  [14:32:20] Email: reconnected           │
│                                          │
└─────────────────────────────────────────┘
```

### Widget

`widget.List` with `canvas.Text` items.

### Entry Format

```
[HH:MM:SS] {Source}: {Message}
```

### Color Rules

| Condition       | Text Color                   |
|-----------------|------------------------------|
| `IsError=true`  | `RGBA(255, 80, 80, 255)`    |
| `IsError=false` | `color.White`                |

### Constraints

- Maximum 500 entries (circular buffer, oldest evicted)
- Updates arrive via channel from orchestrator → presenter
- Callback-driven refresh (`SetOnUpdate`)

---

## Pane 3: Feedback Review (Modal)

### Purpose

Review BUFFERED messages one at a time. Rate them 0–10 with optional notes. Triggered by "Review Buffered" button.

### Layout

```
┌──────────────────────────────────────────────────────┐
│  Feedback Review                              600×400 │
│                                                       │
│  3 of 47 buffered messages reviewed                   │
│                                                       │
│  Source: slack  Sender: JohnDoe  Channel: #general    │
│  IS: 7.2  CS: 0.65                                    │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ Full message content, word-wrapped.              │  │
│  │ Can be long, so the entire modal scrolls.        │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│  Rate (0-10):                                         │
│  [0] [1] [2] [3] [4] [5] [6] [7] [8] [9] [10]       │
│                                                       │
│  ┌─────────────────────────────────────────────────┐  │
│  │ Optional notes...                                │  │
│  └─────────────────────────────────────────────────┘  │
│                                                       │
│              [ Skip ]    [ Delete ]                    │
└──────────────────────────────────────────────────────┘
```

### Widgets

| Widget              | Type                    | Notes                       |
|---------------------|-------------------------|-----------------------------|
| Counter             | `widget.Label`          | `"X of Y buffered messages reviewed"` (1-indexed) |
| Detail info         | `widget.Label`          | Source, Sender, Channel, IS, CS |
| Content             | `widget.Label`          | Word-wrapped                |
| Rating buttons      | 11× `widget.Button`    | In `container.NewHBox()`    |
| Notes               | `widget.MultiLineEntry` | Placeholder: `"Optional notes..."` |
| Skip                | `widget.Button`         |                             |
| Delete              | `widget.Button`         |                             |
| Scroll wrapper      | `container.NewVScroll`  | Wraps entire content        |

### Interactions

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Click rating (0–10) | Save rating + notes to buffer service, advance index  |
| Click Skip          | Advance index without saving                          |
| Click Delete        | Remove message from buffer, advance index             |
| All reviewed        | Close modal or show "all reviewed" state              |

---

## Pane 4: Day Planner (Tab)

### Purpose

Wizard-driven day planning with Pomodoro scheduling and a Countdown-style timer. The pane transitions through wizard steps, then displays an active schedule with burndown timer.

### Idle State (No Active Plan)

```
┌──────────────────────────────────────────────────────────────────┐
│  Day Planner                                                     │
│                                                                  │
│                                                                  │
│                                                                  │
│                     [ Plan My Day ]                              │
│                                                                  │
│                                                                  │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Step 1: Task Selection

```
┌──────────────────────────────────────────────────────────────────┐
│  Day Planner — Select Tasks                         Step 1 of 4  │
│                                                                  │
│  Planning: Wednesday 2026-03-28                                  │
│                                                                  │
│  ☑ Write quarterly report          P:1  [work]  Due: Mar 30     │
│  ☑ Review PR #423                  P:2  [code]                   │
│  ☐ Clean up test fixtures          P:3  [code] [cleanup]         │
│  ☑ Reply to client email           P:4  [work]                   │
│  ☐ Update documentation            P:5  [docs]                   │
│                                                                  │
│  ┌──────────────────────────────────────────────────┐            │
│  │ Add new task: [________________________] P:[_]   │            │
│  │                                    [ Add ]       │            │
│  └──────────────────────────────────────────────────┘            │
│                                                                  │
│                                           [ Cancel ] [ Next → ]  │
└──────────────────────────────────────────────────────────────────┘
```

| Widget              | Type                    | Notes                           |
|---------------------|-------------------------|---------------------------------|
| Step indicator      | `widget.Label`          | `"Step N of 4"`                 |
| Date label          | `widget.Label`          | Target planning date            |
| Task list           | `widget.List` + checks  | Checkboxes for selection        |
| Category badges     | `canvas.Rectangle`      | Colored by category             |
| New task input      | `widget.Entry`          | Title field                     |
| Priority input      | `widget.Entry`          | Integer input                   |
| Add button          | `widget.Button`         | Writes to todo repo             |
| Cancel / Next       | `widget.Button`         | Navigation                      |

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Toggle checkbox     | Select/deselect task for planning                     |
| Add task            | Create in todo repo, add to list as selected          |
| Cancel              | Return to idle state                                  |
| Next                | Proceed to estimates (requires ≥1 task selected)      |

### Step 2: Time Estimates

```
┌──────────────────────────────────────────────────────────────────┐
│  Day Planner — Estimate Time                        Step 2 of 4  │
│                                                                  │
│  Available: 12 Pomodoros (5h focus in 8h day, 3 meetings)        │
│                                                                  │
│  Task                           Est. Pomos    Override           │
│  ─────────────────────────────────────────────────────────       │
│  Write quarterly report         ▪▪▪▪ (4)      [__]              │
│  Review PR #423                 ▪▪ (2)        [__]              │
│  Reply to client email          ▪ (1)         [__]              │
│                                                                  │
│  Total: 7 of 12 Pomodoros                                        │
│                                                                  │
│                                     [ ← Back ] [ Next → ]       │
└──────────────────────────────────────────────────────────────────┘
```

Overloaded state:

```
│  Total: 15 of 12 Pomodoros                                       │
│  ⚠ Overloaded — 3 Pomodoros won't fit today                     │
```

| Widget              | Type                    | Notes                           |
|---------------------|-------------------------|---------------------------------|
| Available label     | `widget.Label`          | Calculated from calendar gaps   |
| Estimate rows       | `widget.List`           | Task + visual dots + number     |
| Override input      | `widget.Entry`          | Optional integer override       |
| Total summary       | `widget.Label`          | Updates live                    |
| Overload warning    | `widget.Label`          | Orange text, visible when > available |
| Back / Next         | `widget.Button`         | Navigation                      |

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Enter override      | Replace Ollama estimate for that task                 |
| Clear override      | Revert to Ollama estimate                             |
| Back                | Return to task selection                              |
| Next                | Proceed to priority ordering                          |

### Step 3: Priority Ordering

```
┌──────────────────────────────────────────────────────────────────┐
│  Day Planner — Set Priority                         Step 3 of 4  │
│                                                                  │
│  Drag to reorder or use arrows:                                  │
│                                                                  │
│  1. Write quarterly report       (4 pomos)   [↑] [↓]            │
│  2. Review PR #423               (2 pomos)   [↑] [↓]            │
│  3. Reply to client email        (1 pomo)    [↑] [↓]            │
│                                                                  │
│  Tasks are scheduled in this order. Higher = scheduled first.    │
│                                                                  │
│                                     [ ← Back ] [ Next → ]       │
└──────────────────────────────────────────────────────────────────┘
```

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Click ↑ / ↓        | Move task up/down in priority order                   |
| Drag row            | Reorder task (if Fyne drag-list supported)            |
| Back                | Return to estimates                                   |
| Next                | Proceed to schedule choice                            |

### Step 4: Schedule Choice

```
┌──────────────────────────────────────────────────────────────────┐
│  Day Planner — Choose Schedule                      Step 4 of 4  │
│                                                                  │
│  ┌─────────────────────────┐  ┌─────────────────────────┐       │
│  │ A: Focus-Maximized      │  │ B: Recovery-Balanced     │       │
│  │                         │  │                          │       │
│  │ 10 focus blocks         │  │ 8 focus blocks           │       │
│  │ 3 short breaks          │  │ 6 short breaks           │       │
│  │ 1 long break            │  │ 2 long breaks            │       │
│  │ 4h 10m total focus      │  │ 3h 20m total focus       │       │
│  │                         │  │                          │       │
│  │ ██░██░████▓▓██░██░██    │  │ ██░██░██▓▓░░██░██░░██   │       │
│  │                         │  │                          │       │
│  │ Best for high-energy    │  │ Best for tough days      │       │
│  │ days with momentum      │  │ or recovery needed       │       │
│  │                         │  │                          │       │
│  │      [ Select A ]       │  │      [ Select B ]        │       │
│  └─────────────────────────┘  └─────────────────────────┘       │
│                                                                  │
│                                     [ ← Back ]                   │
└──────────────────────────────────────────────────────────────────┘

  Legend: ██ = focus  ░░ = break  ▓▓ = meeting
```

| Widget              | Type                    | Notes                           |
|---------------------|-------------------------|---------------------------------|
| Schedule cards      | `container.NewHBox`     | Two side-by-side cards          |
| Stats labels        | `widget.Label`          | Block counts, total focus time  |
| Mini-timeline       | `canvas.Rectangle` row  | Colored blocks in a row         |
| Tradeoff text       | `widget.Label`          | Plain description, no score     |
| Select buttons      | `widget.Button`         | One per card                    |

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Select A / B        | Persist schedule to SQLite, transition to active view |
| Back                | Return to priority ordering                           |

### Step 5: Active Schedule View

```
┌──────────────────────────────────────────────────────────────────┐
│  Day Planner — Active                                            │
│                                                                  │
│  ┌─────────────────┐  ┌──────────────────────────────────┐      │
│  │                  │  │  Schedule                         │      │
│  │    ╱╲            │  │                                   │      │
│  │   ╱  ╲           │  │  09:00 ██████ Focus: Write report │      │
│  │  │    │          │  │  09:25 ░░░░░░ Short break         │      │
│  │  │    │          │  │  09:30 ██████ Focus: Write report │  ←   │
│  │  │    │          │  │  09:55 ░░░░░░ Short break         │      │
│  │  │    │          │  │  10:00 ▓▓▓▓▓▓ Meeting: Standup   │      │
│  │   ╲  ╱           │  │  10:15 ░░░░░░ Short break         │      │
│  │    ╲╱            │  │  10:20 ██████ Focus: Write report │      │
│  │                  │  │  10:45 ░░░░░░ Short break         │      │
│  │  Write report    │  │  10:50 ██████ Focus: Review PR    │      │
│  │                  │  │  11:15 ░░░░░░░░░░ Long break     │      │
│  │ [Complete Task]  │  │  11:35 ██████ Focus: Review PR    │      │
│  │ [Abandon Plan]   │  │  ...                              │      │
│  └─────────────────┘  └──────────────────────────────────┘      │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

  ← arrow indicates current block (highlighted row)
  Timer ring: see Countdown Timer Widget spec below
```

| Widget              | Type                    | Notes                           |
|---------------------|-------------------------|---------------------------------|
| Timer ring          | Custom `fyne.Widget`    | See Countdown Timer spec below  |
| Task name           | `widget.Label`          | Current highest-priority task   |
| Complete Task       | `widget.Button`         | Marks task done, rolls next in  |
| Abandon Plan        | `widget.Button`         | Clears schedule, returns idle   |
| Timeline            | `widget.List`           | Scrollable, current highlighted |
| Block rows          | `canvas.Rectangle` + label | Colored by block type        |

| Action              | Behavior                                              |
|---------------------|-------------------------------------------------------|
| Complete Task       | Mark task done in todo repo, next incomplete task fills remaining focus blocks |
| Abandon Plan        | Delete schedule from SQLite, return to idle state     |
| Timeline scroll     | Scrollable; auto-scrolls to keep current block visible |

---

## Countdown Timer Widget

### Purpose

Custom Fyne canvas widget displaying a circular burndown timer inspired by the Countdown TV show clock. 45 lines radiate inward from the outer edge of a ring, depleting clockwise as time elapses.

### Geometry

```
                    0° (12 o'clock)
                    │  LAST segment
                    │
            315°    │    45°
              ╲     │     ╱
               ╲    │    ╱
                ╲   │   ╱
     270° ──────── ◯ ──────── 90°
                ╱   │   ╲
               ╱    │    ╲
              ╱     │     ╲
            225°    │    135°
                    │
                   180°

  45 lines at 8° intervals (360° / 45 = 8°)
  Start: 8° clockwise from 12 o'clock (first segment)
  End: 0° / 12 o'clock (last segment)
  Direction: clockwise
```

### Line Length Tiers

Lines radiate **inward** from the outer edge of the ring toward the center.

| Angle from vertical | Positions (clock face) | Length   | Count |
|---------------------|------------------------|----------|-------|
| 0°, 90°, 180°, 270° | 12, 3, 6, 9 o'clock  | 3× short | 4     |
| 45°, 135°, 225°, 315° | 1:30, 4:30, 7:30, 10:30 | 2× short | 4 |
| All other angles    | Remaining 37 positions | 1× short | 37    |

Default short length: 12px. Adjustable via design token `timer-line-short`.

### Color and Animation

| State              | Appearance                                         |
|--------------------|----------------------------------------------------|
| Future segments    | Solid yellow `#FFCE1B`                             |
| Current segment    | Yellow `#FFCE1B`, flashing at 1 Hz (500ms on/off) |
| Elapsed segments   | Dimmed `RGBA(255, 206, 27, 64)` or hidden         |

### Timing

- Each segment represents 1/45th of the current block duration (~2.222%).
- The first active segment is at 8° clockwise from 12 o'clock.
- Segments deplete clockwise: 8° → 16° → ... → 352° → 0°.
- The 0° (12 o'clock) line is the **last** segment to deplete.
- Timer resets at the start of each new block.

### Rendering

```go
// Pseudocode for line positioning
for i := 0; i < 45; i++ {
    angle := float64(i+1) * 8.0  // segment 0 = 8°, segment 44 = 360° (= 0°)
    angleRad := angle * math.Pi / 180.0

    // Determine line length tier
    normalizedAngle := math.Mod(angle, 360.0)
    switch {
    case isCardinal(normalizedAngle):    // 0°, 90°, 180°, 270°
        length = 3 * shortLength
    case isDiagonal(normalizedAngle):    // 45°, 135°, 225°, 315°
        length = 2 * shortLength
    default:
        length = shortLength
    }

    // Line from outer edge inward
    outerX := centerX + radius * math.Sin(angleRad)
    outerY := centerY - radius * math.Cos(angleRad)
    innerX := centerX + (radius - length) * math.Sin(angleRad)
    innerY := centerY - (radius - length) * math.Cos(angleRad)
}
```

---

## Settings Modal (Menu → Settings)

### Layout

```
┌──────────────────────────────────┐
│  Audio Settings           400×280 │
│                                   │
│  Notification Volume: 75%         │
│  ┌───────────────────────────┐   │
│  │ ████████████░░░░░░░░░░░░░ │   │  ← slider 0–100
│  └───────────────────────────┘   │
│                                   │
│  Timer Volume: 75%                │
│  ┌───────────────────────────┐   │
│  │ ████████████░░░░░░░░░░░░░ │   │  ← slider 0–100
│  └───────────────────────────┘   │
│                                   │
└──────────────────────────────────┘
```

### Widgets

| Widget               | Type            | Notes                                     |
|----------------------|-----------------|-------------------------------------------|
| Title                | `widget.Label`  | `"Audio Settings"`                        |
| Notification label   | `widget.Label`  | Updates live: `"Notification Volume: {N}%"` |
| Notification slider  | `widget.Slider` | Min=0, Max=100, Step=1                    |
| Timer label          | `widget.Label`  | Updates live: `"Timer Volume: {N}%"`      |
| Timer slider         | `widget.Slider` | Min=0, Max=100, Step=1                    |

### Interactions

| Action                    | Behavior                                              |
|---------------------------|-------------------------------------------------------|
| Drag notification slider  | Update label live, call `AlertService.SetVolume()`    |
| Drag timer slider         | Update label live, call `TimerAlertService.SetVolume()` |

---

## Menu Bar

```
Cue
 ├── Settings    → Open settings modal
 ├── About       → Show version dialog
 └── Quit        → Graceful shutdown
```

---

## Data Flow

```
Orchestrator ──event──→ Bridge goroutine ──→ ActivityPresenter ──→ Activity Log
                                                                     (callback)

Repository ──query──→ NotificationPresenter ──→ Notification Queue
                                                     (30s refresh)

BufferService ──load──→ FeedbackPresenter ──→ Feedback Review Modal
                                                   (on-demand)

AlertService ──volume──→ SettingsPresenter ──→ Settings Modal

TodoRepo ─────query───→ PlannerPresenter ──→ Day Planner Pane (wizard)
CalendarProvider ─fetch─╯     │                    │
Planner ──schedules───────────╯                    │
ScheduleRepo ──persist────────────────────────────╯

TimerPresenter ──tick──→ Countdown Timer Widget
    │                         (1Hz refresh)
    ├──block-end──→ TimerAlertService ──→ sound / missed alert
    └──task-name──→ Task Label
```

---

## Acceptance Criteria (for testing)

### Notification Queue
- [ ] Displays only messages with status NOTIFIED
- [ ] Rows sorted newest-first
- [ ] Source, Sender, Channel truncated to 15 chars independently
- [ ] Preview truncated to 80 chars
- [ ] Clicking a row opens detail dialog with IS, CS, timestamp, full content
- [ ] Resolve button marks message as Resolved and removes from list

### Activity Log
- [ ] Entries formatted as `[HH:MM:SS] Source: Message`
- [ ] Error entries render in red (`RGBA(255, 80, 80, 255)`)
- [ ] Normal entries render in white
- [ ] Maximum 500 entries with FIFO eviction
- [ ] Updates arrive in real-time via channel

### Feedback Review
- [ ] Counter shows `"X of Y buffered messages reviewed"` (1-indexed)
- [ ] Shows Source, Sender, Channel, IS, CS for current message
- [ ] Full message content displayed word-wrapped
- [ ] 11 rating buttons (0–10) in a horizontal row
- [ ] Notes field accepts multiline text
- [ ] Rating click saves and advances; Skip advances without saving; Delete removes and advances
- [ ] Modal scrolls vertically for long content

### Settings
- [ ] Notification volume slider range 0–100 with step 1
- [ ] Notification volume label updates live during drag
- [ ] Notification volume clamped to [0, 100]
- [ ] Timer volume slider range 0–100 with step 1
- [ ] Timer volume label updates live during drag
- [ ] Timer volume clamped to [0, 100]
- [ ] Both sliders operate independently

### Day Planner — Wizard
- [ ] Idle state shows "Plan My Day" button when no active plan
- [ ] Existing plan auto-loaded from SQLite on startup
- [ ] Step 1: displays incomplete todos with checkboxes, categories as colored badges
- [ ] Step 1: new tasks can be added inline and are written to todo repo
- [ ] Step 1: "Next" requires at least one task selected
- [ ] Step 2: Ollama-generated Pomodoro estimates displayed per task
- [ ] Step 2: user can override any estimate via numeric input
- [ ] Step 2: total vs available Pomodoros shown; overload warning when exceeded
- [ ] Step 3: tasks displayed in priority order with up/down reorder controls
- [ ] Step 3: reordering updates todo repo priority field
- [ ] Step 4: two schedule cards side-by-side with stats and mini-timeline
- [ ] Step 4: no computed ranking — tradeoff descriptions only
- [ ] Step 4: selecting a schedule persists to SQLite and transitions to active view
- [ ] All steps: "Back" returns to previous step; "Cancel" returns to idle
- [ ] Calendar fetch failure degrades gracefully (planning proceeds without meetings)

### Day Planner — Active Schedule
- [ ] Vertical timeline shows all blocks with time, colored bar, and label
- [ ] Current block highlighted with distinct background
- [ ] Auto-scrolls to keep current block visible
- [ ] Focus blocks show assigned task name
- [ ] Meeting blocks show calendar event title

### Day Planner — Countdown Timer
- [ ] 45 lines arranged in a ring at 8° intervals
- [ ] Lines radiate inward from the outer edge
- [ ] Cardinal lines (0°, 90°, 180°, 270°) are 3× short length
- [ ] Diagonal lines (45°, 135°, 225°, 315°) are 2× short length
- [ ] All other lines are 1× short length
- [ ] All lines colored yellow `#FFCE1B`
- [ ] Current segment flashes at 1 Hz (500ms on / 500ms off)
- [ ] Segments deplete clockwise starting at 8° from vertical
- [ ] 12 o'clock (0°) is the last segment
- [ ] Elapsed segments dimmed or hidden
- [ ] Timer resets at start of each new block
- [ ] Current task name displayed below the timer ring
- [ ] "Complete Task" marks task done and rolls in next highest-priority incomplete task
- [ ] "Abandon Plan" clears schedule from SQLite and returns to idle

### Day Planner — Audio
- [ ] Timer-end sound plays at end of focus/break blocks
- [ ] Timer-end sound is distinct from notification alert (different beep tonality)
- [ ] During meetings: no timer-end sound played
- [ ] Missed timer alerts during meetings routed to notification queue
- [ ] No replay of missed sounds after meetings end
