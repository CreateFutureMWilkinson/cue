# UI Specification — Cue

This document is the authoritative design specification for Cue's Fyne desktop GUI. Claude implements UI features from this spec. ASCII wireframes define layout; design tokens define styling; component specs define behavior.

---

## Overall Layout

The main window uses a three-column layout. No tab bar — navigation is handled by buttons in the Focus rail and contextual controls on each pane.

### Default State (Collapsed Notifications)

```
┌──────────────────────────────────────────────────────────────────┐
│  Cue  [Settings] [About] [Quit]                        Menu Bar │
├──────┬───────────────────────────────────────────┬───────────────┤
│      │                                           │  Notifs (4)   │
│  ◯   │                                           │───────────────│
│ 18m  │                                           │ [9] #alerts   │
│      │         Character Area                    │  Added to...  │
│ Write│         (fairy, Phase 3)                  │───────────────│
│ report│                                          │ [8.5] Inbox   │
│      │                                           │  Server down  │
│[Done]│                                           │───────────────│
│      │                                           │ [8] #general  │
│      │                                           │  @user deploy │
│      │                                           │───────────────│
│[Plan]│                                           │ [7.2] #team   │
│      │      [ Activity Log (drawer) ]            │  Review Q1... │
│      │                                           │  [◀ expand]   │
├──────┴───────────────────────────────────────────┴───────────────┤

  Focus rail: 10% width
  Character area: 60% width
  Notifications: 30% width
  Window default: 1200w × 800h (from config.toml)
```

### Expanded Notifications State

Triggered by the expand toggle on the notification panel. Character area is temporarily hidden; notifications take its space.

```
┌──────────────────────────────────────────────────────────────────┐
│  Cue  [Settings] [About] [Quit]                        Menu Bar │
├──────┬───────────────────────────────────────────────────────────┤
│      │  Notifications (4)                          [collapse ▶] │
│  ◯   │─────────────────────────────────────────────────────────│
│ 18m  │  [9.0]  slack  #alerts   bot         2m ago   [Dismiss] │
│      │         You were added to #alerts                        │
│ Write│─────────────────────────────────────────────────────────│
│ report│ [8.5]  email  Inbox     alice@ex    5m ago   [Dismiss] │
│      │         URGENT: Server down, need immediate action       │
│[Done]│─────────────────────────────────────────────────────────│
│      │  [8.0]  slack  #general  JohnDoe    12m ago   [Dismiss] │
│      │         Hey @user, the deploy is failing again           │
│      │─────────────────────────────────────────────────────────│
│[Plan]│  [7.2]  slack  #team     manager    20m ago   [Dismiss] │
│[Review]│       Please review the Q1 budget proposal             │
├──────┴───────────────────────────────────────────────────────────┤

  Focus rail: 10% width (unchanged)
  Expanded notifications: 90% width (replaces character area)
  Review button: visible only in expanded state
```

### Column Definitions

| Column | Width | Contents | Always visible |
|---|---|---|---|
| Focus rail | 10% | Timer ring, task name, Done, Plan | Yes |
| Character area | 60% | Fairy (Phase 3), activity log drawer | When notifications collapsed |
| Notifications | 30% (collapsed) / 90% (expanded) | Compact or full notification cards | Yes |

### Center Area Views

The center 60% column displays different content depending on state:

| View | Trigger | Contents |
|---|---|---|
| Character (default) | App startup / collapse notifications / "Back" from Plan view | Fairy character, activity log drawer |
| Plan | "Plan" button in focus rail | Plan overview + todo list (see Plan View spec) |
| Wizard | "Plan My Day" from plan view | Day planner wizard steps 1–4 (see Day Planner spec) |

Only one center view is active at a time. The focus rail and notification column remain unchanged across all center views. There is no separate "Active Schedule" view — the Plan view tree serves as the active schedule when a plan exists.

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
| `notif-card-high`  | `#ffc9c9`                   | Notification card IS ≥ 9     |
| `notif-card-mid`   | `#ffd8a8`                   | Notification card IS ≥ 8     |
| `notif-card-low`   | `#dbe4ff`                   | Notification card IS < 8     |
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
| `task-detail-width`      | 500      | Task detail modal width                |
| `task-detail-height`     | 450      | Task detail modal height               |
| `activity-max-entries`   | 500      | Activity log circular buffer           |
| `feedback-window-width`  | 600      | Feedback review modal width            |
| `feedback-window-height` | 400      | Feedback review modal height           |
| `settings-window-width`  | 400      | Legacy settings modal width (unused)   |
| `settings-window-height` | 280      | Legacy settings modal height (unused)  |
| `refresh-interval`       | 30s      | Canvas refresh interval                |
| `timer-segments`         | 45       | Countdown timer line count             |
| `timer-flash-hz`         | 1        | Countdown timer flash rate (Hz)        |
| `timer-ring-radius`      | 120      | Countdown timer outer radius (px)      |
| `timer-line-short`       | 12       | Countdown timer short line length (px) |
| `timer-line-medium`      | 24       | Countdown timer 2× line length (px)    |
| `timer-line-long`        | 36       | Countdown timer 3× line length (px)    |
| `timeline-block-height`  | 40       | Schedule timeline row height (px)      |

---

## Notification Panel (Right Column)

### Purpose

Displays NOTIFIED messages sorted newest-first. Has two states: collapsed (compact cards in 30% column) and expanded (full cards replacing character area at 90%).

### Collapsed State (30% width, default)

```
┌───────────────┐
│ Notifs (4)    │
│───────────────│
│ [9] #alerts   │
│  Added to...  │
│  bot  2m ago  │
│───────────────│
│ [8.5] Inbox   │
│  Server down  │
│  alice  5m    │
│───────────────│
│ [8] #general  │
│  @user deploy │
│  JohnDoe 12m  │
│───────────────│
│ [7.2] #team   │
│  Review Q1... │
│  manager 20m  │
│               │
│  [◀ expand]   │
└───────────────┘
```

#### Compact Card Format

Each card shows:
- **Row 1:** Importance score badge (color-coded) + channel name
- **Row 2:** Message preview (truncated to fit)
- **Row 3:** Sender + relative time

Card background opacity fades with lower importance score (IS 9 = 40%, IS 7 = 20%).

#### Card Color by Importance

| Importance | Card Background | Badge Color |
|---|---|---|
| IS ≥ 9 | `#ffc9c9` (light red) | `#ef4444` (red) |
| IS ≥ 8 | `#ffd8a8` (light orange) | `#f59e0b` (amber) |
| IS < 8 | `#dbe4ff` (light blue) | `#4a9eed` (blue) |

### Expanded State (90% width)

Triggered by expand toggle. Character area is hidden. Review button appears in focus rail.

```
┌─────────────────────────────────────────────────────────────┐
│  Notifications (4)                               [collapse ▶]│
│─────────────────────────────────────────────────────────────│
│  [9.0]  slack  #alerts   bot         2m ago       [Dismiss] │
│         You were added to #alerts                            │
│─────────────────────────────────────────────────────────────│
│  [8.5]  email  Inbox     alice@ex    5m ago       [Dismiss] │
│         URGENT: Server down, need immediate action           │
│─────────────────────────────────────────────────────────────│
│  [8.0]  slack  #general  JohnDoe    12m ago       [Dismiss] │
│         Hey @user, the deploy is failing again               │
│─────────────────────────────────────────────────────────────│
│  [7.2]  slack  #team     manager    20m ago       [Dismiss] │
│         Please review the Q1 budget proposal                 │
└─────────────────────────────────────────────────────────────┘
```

#### Expanded Card Format

```
[{IS badge}] {Source}  {Channel}  {Sender}  {Relative time}  [Dismiss]
             {Full message preview, word-wrapped}
```

### Interactions

| Action | State | Behavior |
|---|---|---|
| Click expand toggle | Collapsed | Expand to 90%, hide character, show Review in focus rail |
| Click collapse toggle | Expanded | Collapse to 30%, restore character, hide Review |
| Click compact card | Collapsed | Open detail dialog modal (see below) |
| Click expanded card | Expanded | Open detail dialog modal (same dialog) |
| Click Dismiss | Expanded | Mark message as Resolved, remove from list |
| Click Review | Expanded | Open feedback review for buffered messages |
| List refresh | Either | Triggered by 30s canvas tick or manual action |

### Detail Dialog (Modal, from card click in either state)

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

## Activity Log (Drawer in Character Area)

### Purpose

Real-time feed of system events. Hidden by default — accessible via a pull-up drawer button at the bottom of the character area. Only visible when the character view is active (not during expanded notifications, Plan view, or Wizard). Events continue to accumulate in the buffer while the drawer is hidden; opening it shows the latest state.

### Layout

```
┌───────────────────────────────────────────┐
│                                           │
│         Character Area (fairy)            │
│                                           │
├───────────────────────────────────────────┤
│  Activity Log                    [close ▼]│
│                                           │
│  [14:32:05] Slack: Fetched 12 messages    │
│  [14:32:06] Router: 8 NOTIFIED, 3 BUF.. │
│  [14:32:06] Ollama: inference took 250ms  │
│  [14:32:15] Email: connection error...    │  ← red
│  [14:32:20] Email: reconnected            │
│                                           │
│      [ Activity Log (drawer) ]            │
└───────────────────────────────────────────┘
```

When closed, only the drawer toggle button is visible at the bottom of the character area. When open, the log slides up and shares space with the character.

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
- Drawer occupies bottom ~40% of character area when open

---

## Focus Rail (Left Column, 10%)

### Purpose

Persistent left column showing the countdown timer, current task, and navigation. Always visible regardless of center area state.

### Layout

```
┌──────┐
│      │
│  ◯   │  ← Countdown timer ring (see below)
│ 18m  │
│      │
│ Write│  ← Current task name
│ report│
│      │
│[Done]│  ← Completes current task
│      │
│      │
│      │
│[Back]│  ← Returns to character view (only in Plan/Wizard)
│[Plan]│  ← Opens Plan view in center area
│[Review]│ ← Only visible when notifications expanded
└──────┘
```

### Widgets

| Widget | Type | Notes |
|---|---|---|
| Timer ring | Custom `fyne.Widget` | Miniature countdown timer ring (see spec below) |
| Task name | `widget.Label` | Current highest-priority task, word-wrapped |
| Done | `widget.Button` | Marks task done, rolls in next incomplete task |
| Back | `widget.Button` | Returns center area to character view |
| Plan | `widget.Button` | Switches center area to Plan view |
| Review | `widget.Button` | Opens feedback review; only visible in expanded notification state |

### Control Visibility

| Control | Visible when |
|---|---|
| Timer ring | Active plan exists |
| Task name | Active plan exists |
| Done | Active plan exists |
| Back | Center area showing Plan view or Wizard (not character) |
| Plan | Center area showing character (not Plan/Wizard) |
| Review | Notifications expanded |

When no active plan and character is showing, the focus rail shows only the Plan button. Back and Plan are mutually exclusive — one navigates into Plan view, the other returns to character.

### Focus Rail Timer Ring

A miniature version of the Countdown Timer Widget, sized to fit the 10% rail width. Same mechanics as the full-size timer:

- 45 lines at 8° intervals, radiating inward from outer edge
- Cardinal lines (0°, 90°, 180°, 270°): 3× short length
- Diagonal lines (45°, 135°, 225°, 315°): 2× short length
- All other lines: 1× short length
- Future segments: solid yellow `#FFCE1B`
- Current segment: flashing at 1 Hz (500ms on/off)
- Elapsed segments: dimmed `RGBA(255, 206, 27, 64)` or hidden
- Segments deplete clockwise, 12 o'clock is last
- Timer resets at the start of each new block

The ring radius scales to fit the rail width (approximately 40–50px radius at 1200w window). Design token `timer-ring-radius` (120px) applies to the full-size spec; the focus rail version scales proportionally.

---

## Plan View (Center Area)

### Purpose

Intermediate view between the default character view and the full day planner wizard. Occupies the center 60% column, split 50/50 horizontally into a Plan Overview (left) and Todo List (right). Shown when the user clicks "Plan" in the focus rail.

### Layout (Active Plan)

Elapsed blocks (end time in the past) are removed from the tree. The first visible entry is always the current block. Bars auto-scale so the longest remaining block fills the available panel width.

```
┌─────────────────────────────┬─────────────────────────────┐
│  Today's Plan               │  Todo List                  │
│                             │                             │
│  Cycle 1/3                  │  ☐ Write quarterly report   │
│  ├─ Focus                   │     P:1  [work]  Due: Mar 30│
│  │  09:30 ██████████ 25m ██ │  ☐ Review PR #423           │
│  ├─ Short break             │     P:2  [code]             │
│  │  09:55 ██ 5m             │  ☑ Reply to client email    │
│  ├─ Focus                   │     P:4  [work]             │
│  │  10:00 ██████████ 25m ██ │  ☐ Update documentation     │
│  └─ Long break              │     P:5  [docs]             │
│     10:25 ██████ 15m ██     │  ☐ Clean up test fixtures   │
│                             │     P:3  [code] [cleanup]   │
│  Cycle 2/3                  │                             │
│  ├─ Focus                   │  ┌─────────────────────┐   │
│  │  10:40 ██████████ 25m ██ │  │ New task: [_______]  │   │
│  ├─ Short break             │  │ P:[_]  [ Add ]       │   │
│  │  11:05 ██ 5m             │  └─────────────────────┘   │
│  ├─ Meeting: Standup        │                             │
│  │  11:10 ████████████ 30m █│                             │
│  └─ Short break             │                             │
│     11:40 ██ 5m             │                             │
│                             │                             │
│  Cycle 3/3                  │                             │
│  ├─ ...                     │                             │
│                             │                             │
│         [ Abandon Plan ]    │                             │
└─────────────────────────────┴─────────────────────────────┘

  Duration text overlays the bar (centered within it).
  Top entry is always the current block (elapsed blocks removed).
  Bar widths auto-scale: longest remaining block = full panel width.
```

### Layout (No Plan)

```
┌─────────────────────────────┬─────────────────────────────┐
│  Today's Plan               │  Todo List                  │
│                             │                             │
│                             │  ☐ Write quarterly report   │
│                             │     P:1  [work]  Due: Mar 30│
│                             │  ☐ Review PR #423           │
│     "Who even knows"        │     P:2  [code]             │
│                             │  ☐ Reply to client email    │
│            or               │     P:4  [work]             │
│                             │  ☐ Update documentation     │
│  "Its your time you're      │     P:5  [docs]             │
│        wasting"             │                             │
│                             │  ┌─────────────────────┐   │
│            or               │  │ New task: [_______]  │   │
│                             │  │ P:[_]  [ Add ]       │   │
│  "A goal without a plan     │  └─────────────────────┘   │
│   is just a wish"           │                             │
│                             │                             │
│                             │                             │
│        [ Plan My Day ]      │                             │
└─────────────────────────────┴─────────────────────────────┘
```

### Plan Overview (Left Half)

Displays the current day's schedule as a tree view, organized by Pomodoro cycle.

#### Tree Structure

```
Cycle {N}/{Total}
├─ {Block type}                    ← Focus/Short break/Long break (no task name)
│  {HH:MM} ████ {duration} ████   ← Start time, bar with duration overlaid
├─ Meeting: {Event title}          ← Meetings keep their name in the title
│  {HH:MM} ██████ {duration} ████
...
└─ {Block type}
   {HH:MM} ██ {duration}
```

- Focus blocks do NOT show the assigned task name — the task is tracked in the focus rail. Only meetings display a name in the tree title.
- Duration text is rendered centered **on top of** the bar, not beside it.
- Elapsed blocks (end time < now) are pruned from the tree. The first visible block is always the current one.
- If an entire cycle has elapsed, that cycle heading is also removed.

#### Block Types and Bars

Bars are horizontal `canvas.Rectangle` elements with widths proportional to duration relative to the longest block in the plan.

| Block Type | Bar Color | Example Duration |
|---|---|---|
| Focus | `block-focus` `RGBA(76, 175, 80, 200)` | 25m |
| Short break | `block-short-break` `RGBA(144, 202, 249, 200)` | 5m |
| Long break | `block-long-break` `RGBA(100, 181, 246, 200)` | 15–30m |
| Meeting | `block-meeting` `RGBA(255, 183, 77, 200)` | Variable |

Bar width formula: `(block_duration / max_remaining_block_duration) * available_width`

The scale is defined by the longest **remaining** (non-elapsed) block. As blocks elapse and are pruned, the bars rescale so the new longest block fills the full width. Duration text is rendered centered on the bar using contrasting color for readability.

The first block (current) has no special highlight — its position at the top of the tree is sufficient indication.

#### No-Plan Placeholder Text

When no plan exists for the current day, display one of the following randomly selected messages centered in the plan panel, styled in `normal-text` color with a larger font:

- "Who even knows"
- "It's your time you're wasting"
- "A goal without a plan is just a wish"
- "Winging it, are we?"
- "The plan is there is no plan"
- "Chaos is also a strategy, I suppose"
- "Bold of you to go planless"

Selected once on view load (not rotating).

#### Bottom Controls

| Plan State | Control | Behavior |
|---|---|---|
| No plan | **Plan My Day** | Launches the day planner wizard in center area |
| Active plan | **Abandon Plan** | Deletes current plan from SQLite, returns to no-plan state with placeholder |

### Todo List (Right Half)

Displays all incomplete tasks from the todo repository, plus completed tasks for the current day. Provides inline task creation and per-task detail editing via a modal dialog.

#### Layout

```
┌─────────────────────────────┐
│  Todo List                  │
│                             │
│  ☐ Write quarterly report   │
│     P:1  [work]  Due: Mar 30│
│     [details]               │
│  ───────────────────────────│
│  ☐ Review PR #423           │
│     P:2  [code]             │
│     [details]               │
│  ───────────────────────────│
│  ☑ Reply to client email    │
│     P:4  [work]             │
│     [details]               │
│  ───────────────────────────│
│                             │
│  ┌─────────────────────┐   │
│  │ New task: [_______]  │   │
│  │ P:[_]  [ Add ]       │   │
│  └─────────────────────┘   │
└─────────────────────────────┘
```

Scrollable list of tasks, each showing:
- **Row 1:** Checkbox + task title
- **Row 2:** Priority (`P:{N}`), category badges (colored), optional due date
- **Row 3:** `[details]` link — opens the Task Detail Modal

Completed tasks shown with strikethrough text and reduced opacity.

#### Task Detail Modal

A modal dialog (blocks main window interaction until closed). Opens when user clicks `[details]` on any task.

```
┌──────────────────────────────────────────┐
│  Task Detail                      [Close]│
│                                          │
│  Title                                   │
│  ┌──────────────────────────────────┐   │
│  │ Write quarterly report            │   │
│  └──────────────────────────────────┘   │
│                                          │
│  Priority        Category                │
│  ┌──────┐       ┌──────────────────┐   │
│  │ 1    │       │ work             │   │
│  └──────┘       └──────────────────┘   │
│                                          │
│  Due Date                                │
│  ┌──────────────────────────────────┐   │
│  │ 2026-03-30                        │   │
│  └──────────────────────────────────┘   │
│                                          │
│  Notes                                   │
│  ┌──────────────────────────────────┐   │
│  │ Need to include Q4 actuals and    │   │
│  │ projections for Q2. Check with    │   │
│  │ finance for latest numbers.       │   │
│  └──────────────────────────────────┘   │
│                                          │
│           [ Save ]    [ Cancel ]         │
└──────────────────────────────────────────┘
```

#### Task Detail Modal Widgets

| Widget | Type | Notes |
|---|---|---|
| Title | `widget.Entry` | Editable, pre-filled |
| Priority | `widget.Entry` | Integer input |
| Category | `widget.Entry` | Free-text, maps to badge color |
| Due Date | `widget.Entry` | ISO date format `YYYY-MM-DD`, optional |
| Notes | `widget.MultiLineEntry` | Free-text, optional |
| Save | `widget.Button` | Persist changes to todo repo, close modal |
| Cancel | `widget.Button` | Discard changes, close modal |
| Close (X) | `widget.Button` | Same as Cancel |

#### Task Detail Modal Behavior

| Property | Value |
|---|---|
| Window type | Modal — main window does not accept input while open |
| Size | 500w × 450h |
| Pre-fill | All fields populated from existing task data |
| New fields | Notes field is new (not shown in list view) |

#### Inline Task Creation

Input row pinned at bottom of the todo list:
- Task title entry field
- Priority number entry field
- Add button — writes to todo repository, adds to list

#### Interactions

| Action | Behavior |
|---|---|
| Toggle checkbox | Mark task complete/incomplete in todo repo |
| Click `[details]` | Open Task Detail Modal for that task |
| Save in modal | Persist all field changes to todo repo, close modal, refresh list |
| Cancel in modal | Discard changes, close modal |
| Add task | Create in todo repo with title + priority, appear in list |
| Scroll | Vertical scroll for long lists |

#### Sort Order

Tasks sorted by: incomplete first, then by priority (ascending P:1 before P:2), then by creation date.

---

## Feedback Review (Modal)

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

## Day Planner (Center Area — Wizard)

### Purpose

Wizard-driven day planning with Pomodoro scheduling and a Countdown-style timer. Replaces the character area when launched from the Plan view. Transitions through wizard steps, then displays an active schedule with burndown timer.

The wizard has no idle state of its own — the Plan View's no-plan state handles the "Plan My Day" entry point. The wizard only contains Steps 1–4.

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
| Cancel              | Return to Plan view                                   |
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
| Select A / B        | Persist schedule to SQLite, return to Plan view (which now shows the schedule tree) |
| Back                | Return to priority ordering                           |

There is no separate Active Schedule view. When the wizard completes (Step 4), the user returns to the Plan view, which now displays the schedule as a tree of cycles and blocks. The timer ring in the focus rail tracks the current block. "Done" in the focus rail completes the current task. "Abandon Plan" in the Plan view deletes the schedule.

---

## Countdown Timer Widget

### Purpose

Custom Fyne canvas widget displaying a circular burndown timer inspired by the Countdown TV show clock. 45 lines radiate inward from the outer edge of a ring, depleting clockwise as time elapses. The full-size spec below defines the canonical geometry; the focus rail uses a scaled-down version (see Focus Rail Timer Ring).

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

## Settings View (Center Column)

The Settings view occupies the center column (60%) when navigated to via the Settings button in the focus rail. It uses a tabbed layout (`container.AppTabs`) with a Done button at the bottom that returns to the character view.

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [Slack] [Email] [Calendar] [Audio] [Ollama]        Tab bar │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  (Active tab content — see per-tab specs below)             │
│                                                             │
│                                                             │
│                                                             │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                    [Done]   │
└─────────────────────────────────────────────────────────────┘
```

### Tab Order

| Position | Tab        | Content                                      |
|----------|------------|----------------------------------------------|
| 1        | Slack      | Slack account management (list + add form)    |
| 2        | Email      | Email account management (list + add form)    |
| 3        | Calendar   | Calendar account management (list + add form) |
| 4        | Audio      | Notification and timer volume sliders         |
| 5        | Ollama     | Ollama connection settings (read-only)        |

### Slack Tab

Displays a list of configured Slack accounts with a button to add new ones.

**Account List View:**

```
┌─────────────────────────────────────────────────────┐
│  Slack Accounts                                     │
├─────────────────────────────────────────────────────┤
│  Slack: T12345 (@alice)                   [Delete]  │
│  Slack: T67890 (@bob)                     [Delete]  │
│                                                     │
│                              [Add Account]          │
└─────────────────────────────────────────────────────┘
```

**Empty State:** `"No Slack accounts configured. Tap Add Account to get started."`

**Add Account Form** (replaces list when tapped):

| Field             | Widget              | Notes                                         |
|-------------------|---------------------|-----------------------------------------------|
| Friendly Name     | `widget.Entry`      | Placeholder: `"Friendly Name"`                |
| Web URL           | `widget.Entry`      | Placeholder: `"Web URL"`                      |
| Token             | `widget.Entry`      | Placeholder: `"User OAuth Token (xoxp-...)"`, password masked |
| Workspace ID      | `widget.Entry`      | Placeholder: `"Workspace ID"`                 |
| Username          | `widget.Entry`      | Placeholder: `"Your Slack Username (@handle)"`|
| Poll Interval     | `widget.Entry`      | Placeholder: `"Poll Interval (seconds)"`, default: 60 |
| Error             | `widget.Label`      | Hidden by default, shown on validation failure|
| Save / Cancel     | `widget.Button` ×2  | Save validates + persists; Cancel returns to list |

**Token Instructions** (`widget.Accordion`, collapsed by default, placed between form header and first entry):

A single accordion item titled **"How to get a token"** with step-by-step guidance:

1. Go to https://api.slack.com/apps
2. Create a new app (or select existing) for your workspace
3. Add the following **User Token Scopes** under OAuth & Permissions:
   - `channels:history` — read messages in public channels
   - `channels:read` — list public channels
   - `groups:history` — read messages in private channels
   - `groups:read` — list private channels
   - `im:history` — read direct messages
   - `im:read` — list direct message channels
   - `mpim:history` — read group direct messages
   - `mpim:read` — list group DM channels
   - `users:read` — resolve user display names
4. Install the app to your workspace
5. Copy the User OAuth Token (starts with `xoxp-`)

**Validation:** Token, Workspace ID, Username, and Poll Interval are required. Poll Interval must be a number. If a `SlackValidator` is configured, credentials are validated before save. Inline error shown on failure.

### Email Tab

Displays a list of configured email accounts with a button to add new ones. Same list/form switching pattern as Slack.

**Account List View:**

```
┌─────────────────────────────────────────────────────┐
│  Email Accounts                                     │
├─────────────────────────────────────────────────────┤
│  Email: user@gmail.com (imap.gmail.com:993) [Delete]│
│                                                     │
│                              [Add Account]          │
└─────────────────────────────────────────────────────┘
```

**Empty State:** `"No Email accounts configured. Tap Add Account to get started."`

**Add Account Form:**

| Field             | Widget              | Notes                                         |
|-------------------|---------------------|-----------------------------------------------|
| Friendly Name     | `widget.Entry`      | Placeholder: `"Friendly Name"`                |
| Web URL           | `widget.Entry`      | Placeholder: `"Web URL"`                      |
| IMAP Host         | `widget.Entry`      | Placeholder: `"IMAP Host"`                    |
| IMAP Port         | `widget.Entry`      | Placeholder: `"IMAP Port"`                    |
| Username          | `widget.Entry`      | Placeholder: `"Username"`                     |
| Password          | `widget.Entry`      | Placeholder: `"Password"`, password masked    |
| Encryption        | `widget.Select`     | Options: `"SSL/TLS (Recommended)"`, `"STARTTLS"`, `"None"`. Default: SSL/TLS |
| Poll Interval     | `widget.Entry`      | Placeholder: `"Poll Interval (seconds)"`, default: 600 |
| Error             | `widget.Label`      | Hidden by default, shown on validation failure|
| Save / Cancel     | `widget.Button` ×2  | Save validates + persists; Cancel returns to list |

**Validation:** IMAP Host, Port, Username, Password, and Poll Interval are required. Port and Poll Interval must be numbers. If an `EmailValidator` is configured, credentials are validated before save. Inline error shown on failure.

### Calendar Tab

Displays a list of configured calendar accounts with a button to add new ones. Same list/form switching pattern as Slack and Email.

**Account List View:**

```
┌─────────────────────────────────────────────────────┐
│  Calendar Accounts                                  │
├─────────────────────────────────────────────────────┤
│  Calendar: Work Calendar                  [Delete]  │
│                                                     │
│                              [Add Account]          │
└─────────────────────────────────────────────────────┘
```

**Empty State:** `"No Calendar accounts configured. Tap Add Account to get started."`

**Add Account Form:**

| Field             | Widget              | Notes                                         |
|-------------------|---------------------|-----------------------------------------------|
| Account Name      | `widget.Entry`      | Placeholder: `"Account Name"`                 |
| ICS URL           | `widget.Entry`      | Placeholder: `"ICS Calendar URL"`             |
| Poll Interval     | `widget.Entry`      | Placeholder: `"Poll Interval (seconds)"`, default: 600 |
| Error             | `widget.Label`      | Hidden by default, shown on validation failure|
| Save / Cancel     | `widget.Button` ×2  | Save validates + persists; Cancel returns to list |

**Validation:** Name, URL, and Poll Interval are required. Poll Interval must be a number. If a `CalendarValidator` is configured, the URL is validated before save. Inline error shown on failure.

### Audio Tab

```
┌─────────────────────────────────────────────────────┐
│  Audio Settings                                     │
│                                                     │
│  Notification Volume: 75%                           │
│  ┌───────────────────────────────────────────────┐  │
│  │ ████████████░░░░░░░░░░░░░                     │  │  ← slider 0–100
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  Timer Volume: 75%                                  │
│  ┌───────────────────────────────────────────────┐  │
│  │ ████████████░░░░░░░░░░░░░                     │  │  ← slider 0–100
│  └───────────────────────────────────────────────┘  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

| Widget               | Type            | Notes                                     |
|----------------------|-----------------|-------------------------------------------|
| Title                | `widget.Label`  | `"Audio Settings"`                        |
| Notification label   | `widget.Label`  | Updates live: `"Notification Volume: {N}%"` |
| Notification slider  | `widget.Slider` | Min=0, Max=100, Step=1                    |
| Timer label          | `widget.Label`  | Updates live: `"Timer Volume: {N}%"`      |
| Timer slider         | `widget.Slider` | Min=0, Max=100, Step=1                    |

### Ollama Tab

Displays the current Ollama configuration as read-only labels.

```
┌─────────────────────────────────────────────────────┐
│  Ollama Settings                                    │
│                                                     │
│  Host: localhost                                    │
│  Port: 11434                                        │
│  Inference Model: neural-chat                       │
│  Embedding Model: nomic-embed-text                  │
│  Timeout: 10s                                       │
│                                                     │
└─────────────────────────────────────────────────────┘
```

| Widget           | Type            | Notes                              |
|------------------|-----------------|------------------------------------|
| Title            | `widget.Label`  | `"Ollama Settings"`                |
| Host             | `widget.Label`  | From `config.OllamaConfig.Host`    |
| Port             | `widget.Label`  | From `config.OllamaConfig.Port`    |
| Inference Model  | `widget.Label`  | From `config.OllamaConfig.InferenceModel` |
| Embedding Model  | `widget.Label`  | From `config.OllamaConfig.EmbeddingModel` |
| Timeout          | `widget.Label`  | From `config.OllamaConfig.TimeoutSeconds` |

### Settings — Interactions

| Action                    | Behavior                                              |
|---------------------------|-------------------------------------------------------|
| Switch tab                | Display selected tab content; previous tab state preserved |
| Drag notification slider  | Update label live, call `AlertService.SetVolume()`    |
| Drag timer slider         | Update label live, call `TimerAlertService.SetVolume()` |
| Tap Add Account           | Replace account list with add form                    |
| Tap Save (account form)   | Validate fields, persist account, return to list      |
| Tap Cancel (account form) | Discard form, return to account list                  |
| Tap Delete (account row)  | Remove account and its watcher                        |
| Tap Done                  | Navigate back to character view                       |

---

## Menu Bar

```
Cue
 ├── Settings    → Navigate to settings view (center column)
 ├── About       → Show version dialog
 └── Quit        → Graceful shutdown
```

---

## Data Flow

```
Orchestrator ──event──→ Bridge goroutine ──→ ActivityPresenter ──→ Activity Log Drawer
                                                                     (callback)

Repository ──query──→ NotificationPresenter ──→ Notification Panel (right column)
                                                     (30s refresh)

BufferService ──load──→ FeedbackPresenter ──→ Feedback Review Modal
                                                   (on-demand, from expanded view)

AlertService ──volume──→ SettingsPresenter ──→ Settings View (center column, Audio tab)
ServiceConfigRepo ─────→ ServiceSettingsPresenter ──→ Settings View (Slack/Email/Calendar tabs)

TodoRepo ─────query───→ PlannerPresenter ──→ Plan View / Wizard (center area)
CalendarProvider ─fetch─╯     │                    │
Planner ──schedules───────────╯                    │
ScheduleRepo ──persist────────────────────────────╯

TimerPresenter ──tick──→ Countdown Timer Widget (focus rail)
    │                         (1Hz refresh)
    ├──block-end──→ TimerAlertService ──→ sound / missed alert
    └──task-name──→ Task Label (focus rail)

CenterViewRouter ──state──→ Character | Plan View | Wizard
    (manages which view occupies the center 60% column)
```

---

## Acceptance Criteria (for testing)

### Three-Column Layout
- [ ] Focus rail occupies 10% width, always visible
- [ ] Character area occupies 60% width (center column)
- [ ] Notification panel occupies 30% width (right column)
- [ ] No tab bar — navigation via focus rail buttons and contextual controls

### Focus Rail
- [ ] Timer ring visible when active plan exists (miniature countdown timer)
- [ ] Timer ring has same mechanics as full spec: 45 lines, 8° intervals, color/flash behavior
- [ ] Timer ring scales to fit rail width (~40–50px radius)
- [ ] Current task name displayed below timer
- [ ] Done button marks task complete and rolls next task in
- [ ] Back button visible when center area is Plan view or Wizard (returns to character)
- [ ] Plan button visible when center area is character (switches to Plan view)
- [ ] Back and Plan are mutually exclusive
- [ ] Review button only visible when notifications are expanded

### Notification Panel (Collapsed)
- [ ] Displays only messages with status NOTIFIED
- [ ] Cards sorted newest-first
- [ ] Each card shows: IS badge (color-coded), channel, message preview, sender, relative time
- [ ] Card background opacity fades with lower importance score
- [ ] Clicking a card opens detail dialog modal with IS, CS, timestamp, full content
- [ ] Expand toggle at bottom expands to 90% width
- [ ] Resolve button in detail dialog marks message as Resolved

### Notification Panel (Expanded)
- [ ] Takes over character area (90% width), character hidden
- [ ] Full-width cards with source, channel, sender, relative time, message preview
- [ ] Clicking a card opens detail dialog modal (same as collapsed)
- [ ] Per-card Dismiss button marks as Resolved and removes from list
- [ ] Collapse toggle returns to 30% width and restores character
- [ ] Review button appears in focus rail

### Activity Log (Drawer)
- [ ] Hidden by default, toggle button at bottom of character area
- [ ] Entries formatted as `[HH:MM:SS] Source: Message`
- [ ] Error entries render in red (`RGBA(255, 80, 80, 255)`)
- [ ] Normal entries render in white
- [ ] Maximum 500 entries with FIFO eviction
- [ ] Updates arrive in real-time via channel
- [ ] Only accessible when character area is visible (not during expanded notifications)
- [ ] Drawer occupies bottom ~40% of character area when open

### Feedback Review
- [ ] Triggered from Review button in focus rail (expanded notifications state)
- [ ] Counter shows `"X of Y buffered messages reviewed"` (1-indexed)
- [ ] Shows Source, Sender, Channel, IS, CS for current message
- [ ] Full message content displayed word-wrapped
- [ ] 11 rating buttons (0–10) in a horizontal row
- [ ] Notes field accepts multiline text
- [ ] Rating click saves and advances; Skip advances without saving; Delete removes and advances
- [ ] Modal scrolls vertically for long content

### Settings View
- [ ] Rendered in center column (60%) via CenterViewRouter
- [ ] Contains 5 tabs in order: Slack, Email, Calendar, Audio, Ollama
- [ ] Defaults to first tab (Slack) on open
- [ ] Done button at bottom navigates back to character view

### Settings — Slack Tab
- [ ] Shows list of configured Slack accounts with Delete button per row
- [ ] Empty state message when no accounts configured
- [ ] Add Account button shows inline form (replaces list)
- [ ] Form requires Token, Workspace ID, Username, Poll Interval
- [ ] Poll Interval must be a number; inline error on invalid input
- [ ] Save persists account and starts watcher; Cancel returns to list

### Settings — Email Tab
- [ ] Shows list of configured email accounts with Delete button per row
- [ ] Empty state message when no accounts configured
- [ ] Add Account button shows inline form (replaces list)
- [ ] Form requires IMAP Host, Port, Username, Password, Poll Interval
- [ ] Port and Poll Interval must be numbers; inline error on invalid input
- [ ] Encryption dropdown defaults to SSL/TLS (Recommended)
- [ ] Save persists account and starts watcher; Cancel returns to list

### Settings — Calendar Tab
- [ ] Shows list of configured calendar accounts with Delete button per row
- [ ] Empty state message when no accounts configured
- [ ] Add Account button shows inline form (replaces list)
- [ ] Form requires Name, ICS URL, Poll Interval
- [ ] Poll Interval must be a number; inline error on invalid input
- [ ] Save persists account; Cancel returns to list

### Settings — Audio Tab
- [ ] Notification volume slider range 0–100 with step 1
- [ ] Notification volume label updates live during drag
- [ ] Notification volume clamped to [0, 100]
- [ ] Timer volume slider range 0–100 with step 1
- [ ] Timer volume label updates live during drag
- [ ] Timer volume clamped to [0, 100]
- [ ] Both sliders operate independently

### Settings — Ollama Tab
- [ ] Displays Host, Port, Inference Model, Embedding Model, Timeout as read-only labels
- [ ] Values sourced from config.OllamaConfig

### Plan View — Layout
- [ ] Displayed in center area (60%) when Plan button clicked in focus rail
- [ ] Split 50/50 horizontally: Plan Overview (left) + Todo List (right)

### Plan View — Plan Overview (Active Plan)
- [ ] Tree view organized by Pomodoro cycle (Cycle 1/N, Cycle 2/N, etc.)
- [ ] Elapsed blocks (end time in past) pruned — first entry is current block
- [ ] Fully elapsed cycles also pruned
- [ ] Each cycle shows nested blocks: Focus, Short break, Long break, Meeting
- [ ] Blocks displayed with colored bars, width proportional to duration
- [ ] Bar scale: longest remaining block = full panel width, others proportional
- [ ] Duration text overlays the bar (centered on it)
- [ ] Each bar row prefixed with start time (HH:MM)
- [ ] Focus bars green, short break light blue, long break blue, meeting amber
- [ ] No current block highlight — position at top is sufficient indication
- [ ] Focus blocks titled "Focus" only (no task name — task shown in focus rail)
- [ ] Meeting blocks titled "Meeting: {event name}"
- [ ] "Abandon Plan" button at bottom — deletes plan, shows placeholder

### Plan View — No Plan State
- [ ] Random humorous/passive-aggressive placeholder text displayed centered
- [ ] Text selected once on view load (not rotating)
- [ ] "Plan My Day" button at bottom — launches wizard

### Plan View — Todo List
- [ ] Displays incomplete tasks + current day's completed tasks
- [ ] Each task shows: checkbox, title, priority, category badges, optional due date
- [ ] Each task has a `[details]` link below metadata
- [ ] Completed tasks shown with strikethrough and reduced opacity
- [ ] Sorted: incomplete first, then by priority (ascending), then creation date
- [ ] Inline task creation at bottom: title field, priority field, Add button
- [ ] Toggle checkbox marks task complete/incomplete in todo repo
- [ ] Add button writes new task to todo repo

### Plan View — Task Detail Modal
- [ ] Opens when `[details]` clicked on any task
- [ ] Modal blocks main window input until closed
- [ ] Pre-fills all fields from existing task data
- [ ] Editable fields: Title, Priority, Category, Due Date, Notes
- [ ] Save persists changes to todo repo and closes modal
- [ ] Cancel discards changes and closes modal
- [ ] Close (X) button behaves same as Cancel
- [ ] Modal size: 500w × 450h

### Day Planner — Wizard (Steps 1–4)
- [ ] Launched from "Plan My Day" in Plan view, replaces center area
- [ ] No idle state — Plan view handles "Plan My Day" entry point
- [ ] Existing plan auto-loaded from SQLite on startup (goes straight to Plan view)
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
- [ ] Step 4: selecting a schedule persists to SQLite and returns to Plan view
- [ ] All steps: "Back" returns to previous step; "Cancel" returns to Plan view
- [ ] Calendar fetch failure degrades gracefully (planning proceeds without meetings)

### Countdown Timer (Focus Rail)
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
- [ ] Ring scales to fit focus rail width (~40–50px radius)
- [ ] Current task name displayed below the timer ring in focus rail
- [ ] "Done" in focus rail marks task done and rolls in next highest-priority incomplete task
- [ ] "Abandon Plan" in Plan view clears schedule from SQLite and returns to no-plan state

### Day Planner — Audio
- [ ] Timer-end sound plays at end of focus/break blocks
- [ ] Timer-end sound is distinct from notification alert (different beep tonality)
- [ ] During meetings: no timer-end sound played
- [ ] Missed timer alerts during meetings routed to notification queue
- [ ] No replay of missed sounds after meetings end
