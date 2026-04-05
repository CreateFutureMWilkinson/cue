# Feature 019: Activity Log Drawer

**Phase:** Phase-1-Feature-019
**Status:** Planned
**Packages:** `internal/ui/`, `internal/ui/presenter/`

---

## Overview

Convert the activity log from a standalone right-side pane to a pull-up drawer within the character area (center column). The drawer is hidden by default with only a toggle button visible at the bottom of the character area. When opened, it slides up and occupies the bottom ~40% of the character area, sharing space with the character widget above. The drawer is only accessible when the character view is active — not during expanded notifications, Plan view, or Wizard. Events continue to accumulate in the buffer while the drawer is hidden.

## Design Decisions

- **Drawer pattern, not a separate pane** — the activity log shares vertical space with the character widget instead of consuming a dedicated column. This frees horizontal space for the three-column layout.
- **Hidden by default** — the activity log is secondary information. Users who want to see system events pull up the drawer; it doesn't compete for attention with notifications.
- **Only visible in character view** — the drawer toggle and content are part of the character area. When the center area shows Plan or Wizard, the drawer is not accessible. Events still accumulate in the circular buffer and are visible when the user returns to character view.
- **Bottom ~40% when open** — the character widget occupies the top portion and the activity log the bottom. The split is approximate; Fyne's layout handles the specifics.
- **Existing ActivityPresenter reused** — no changes to the presenter's event handling, circular buffer, or entry format. Only the view layer changes from a standalone pane to a drawer widget.
- **Entry format unchanged** — `[HH:MM:SS] Source: Message` with red for errors, white for normal.

## API

### ActivityDrawer (View Component)

```go
type ActivityDrawer struct {
    widget.BaseWidget
    presenter ActivityPresenter
    isOpen    bool
    onToggle  func(bool)
}

func NewActivityDrawer(presenter ActivityPresenter) *ActivityDrawer
func (d *ActivityDrawer) IsOpen() bool
func (d *ActivityDrawer) Toggle()
func (d *ActivityDrawer) SetOnToggle(fn func(bool))
```

### Character Area Container

```go
// CharacterAreaContainer manages the vertical split between
// the character widget and the activity log drawer.
type CharacterAreaContainer struct {
    characterWidget fyne.CanvasObject
    activityDrawer  *ActivityDrawer
}

func NewCharacterAreaContainer(
    characterWidget fyne.CanvasObject,
    activityDrawer *ActivityDrawer,
) *CharacterAreaContainer
```

## Layout

### Drawer Closed (Default)

```
┌───────────────────────────────────────────┐
│                                           │
│         Character Area (fairy)            │
│                                           │
│                                           │
│                                           │
│                                           │
│                                           │
│                                           │
│      [ ▲ Activity Log ]                  │  ← toggle button
└───────────────────────────────────────────┘
```

### Drawer Open

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
└───────────────────────────────────────────┘
```

## Color Rules

| Condition | Text Color |
|---|---|
| `IsError=true` | `RGBA(255, 80, 80, 255)` |
| `IsError=false` | `color.White` |

## Constraints

- Maximum 500 entries (circular buffer, oldest evicted) — unchanged from current implementation
- Updates arrive via channel from orchestrator — unchanged
- Callback-driven refresh (`SetOnUpdate`) — unchanged
- Drawer occupies bottom ~40% of character area when open
- Only accessible when character view is active

## Error Handling

| Scenario | Behavior |
|---|---|
| Drawer toggled while not in character view | Toggle ignored, drawer remains hidden |
| Character widget nil | Drawer can still open; character portion shows empty space |
| Rapid toggle | Debounced by Fyne's layout refresh cycle |

## Integration Points

- **Three-Column Layout (Feature 016):** Activity drawer lives inside the center area column, within the character view.
- **CenterViewRouter (Feature 016):** Drawer auto-closes and hides when center view switches away from character.
- **Existing ActivityPresenter (Feature 011):** Reused without modification. Only the view layer changes.
- **Character System (Feature 014):** Character widget shares vertical space with the drawer.
- **UI-SPEC.md:** Authoritative reference for drawer behavior, entry format, and color rules.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `ui` | `ActivityDrawerSuite` | Default closed, toggle opens, toggle closes, toggle callback fires, entries display with timestamps, error entries red, normal entries white, max 500 entries, drawer hidden when not character view |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| ActivityDrawer | RED | Test Designer | — | — | — |
| ActivityDrawer | GREEN | Implementer | — | — | — |
| ActivityDrawer | REFACTOR | Refactorer | — | — | — |
