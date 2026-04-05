# UI Specification — Cue

This document is the authoritative design specification for Cue's Fyne desktop GUI. Claude implements UI features from this spec. ASCII wireframes define layout; design tokens define styling; component specs define behavior.

---

## Overall Layout

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
├───────────────────────────┴──────────────────────────────────────┤
│                    [ Review Buffered ]                            │
└──────────────────────────────────────────────────────────────────┘

  Split: 50/50 horizontal (HSplit)
  Window default: 1200w × 800h (from config.toml)
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
| `settings-window-height` | 200      | Settings modal height                  |
| `refresh-interval`       | 30s      | Canvas refresh interval                |

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

## Settings Modal (Menu → Settings)

### Layout

```
┌────────────────────────────┐
│  Audio Settings     400×200 │
│                             │
│  Volume: 75%                │
│  ┌─────────────────────┐   │
│  │ ████████████░░░░░░░ │   │  ← slider 0–100
│  └─────────────────────┘   │
│                             │
└────────────────────────────┘
```

### Widgets

| Widget        | Type            | Notes                            |
|---------------|-----------------|----------------------------------|
| Title         | `widget.Label`  | `"Audio Settings"`               |
| Volume label  | `widget.Label`  | Updates live: `"Volume: {N}%"`   |
| Volume slider | `widget.Slider` | Min=0, Max=100, Step=1           |

### Interactions

| Action          | Behavior                                    |
|-----------------|---------------------------------------------|
| Drag slider     | Update volume label live, call `SetVolume()` |

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
- [ ] Volume slider range 0–100 with step 1
- [ ] Volume label updates live during drag
- [ ] Volume clamped to [0, 100]
