# Feature 070A: Activity Log Button Layout

**Phase:** Phase-6-Feature-070A
**Type:** Bugfix (Hotfix)
**Severity:** Medium
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 070 (Activity Log Overlay), Feature 019 (Activity Log Drawer)

---

## Problem

The "Activity Log" toggle button fills the entire center panel when the drawer is closed. It should be a small button anchored at the bottom of the character area — the same height as the focus rail's Plan/Back/Done buttons. Additionally, the character widget should treat this button area as bottom padding, shifting the character rendering area upward by the button's height so the character and button don't overlap.

## Root Cause

`activity_log_drawer.go:35`:
```go
d.drawerBox = container.NewStack(d.toggleBtn)
```

And `activity_log_drawer.go:84-85`:
```go
d.stackContainer = container.NewStack(character, d.drawerBox)
```

`container.NewStack` causes every child to fill the entire available space. The toggle button stretches to fill the full center panel area — both width and height. When the drawer is closed, this means the entire center column is one giant clickable button.

## Proposed Fix

### Closed State: Button anchored at bottom, character padded above

When the drawer is closed, use `container.NewBorder` instead of `container.NewStack` to anchor the toggle button at the bottom, with the character filling the remaining space above:

```go
// Closed state: character fills center, button anchored at bottom
d.stackContainer = container.NewBorder(nil, d.toggleBtn, nil, nil, character)
```

This gives the layout:
```
┌─────────────────────────┐
│                         │
│    Character widget     │  ← fills remaining space (character shifted up)
│    (fairy in jar)       │
│                         │
├─────────────────────────┤
│    [Activity Log]       │  ← button at natural height (same as Plan/Back)
└─────────────────────────┘
```

The button renders at its natural `MinSize` height (matching `widget.Button` default, same as focus rail buttons). The character widget gets the remaining space above — no overlap, no stretching.

### Open State: Overlay with button + log list

When the drawer opens, switch to the existing Stack overlay approach but with the drawer properly constrained. The overlay should still cover the character area with the semi-transparent background, with the log list and close button on top:

```go
// Open state: character underneath, overlay on top
overlay := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 77})
drawerContent := container.NewBorder(d.toggleBtn, nil, nil, nil, d.logList)
d.stackContainer.Objects = []fyne.CanvasObject{
    container.NewBorder(nil, d.toggleBtn, nil, nil, character), // character still padded
    overlay,
    container.NewBorder(d.toggleBtn, nil, nil, nil, d.logList), // log overlay
}
```

Alternatively, the simpler approach: keep the Border layout as the base and overlay only when open:

```go
func (d *ActivityLogDrawer) ToggleOpen() {
    d.open = !d.open
    if d.open {
        d.toggleBtn.SetText("close ▼")
        overlay := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 77})
        logView := container.NewBorder(d.toggleBtn, nil, nil, nil, d.logList)
        d.stackContainer.Objects = []fyne.CanvasObject{d.character, overlay, logView}
    } else {
        d.toggleBtn.SetText("Activity Log")
        d.stackContainer.Objects = []fyne.CanvasObject{
            container.NewBorder(nil, d.toggleBtn, nil, nil, d.character),
        }
    }
    d.stackContainer.Refresh()
}
```

**Key change:** In the closed state, the `stackContainer` holds a single `Border` layout (not a `Stack`), so nothing stretches. In the open state, the overlay behaviour remains as Feature 070 implemented it.

### ContainerWithCharacter update

```go
func (d *ActivityLogDrawer) ContainerWithCharacter(character fyne.CanvasObject) fyne.CanvasObject {
    if character == nil {
        character = widget.NewLabel("")
    }
    d.character = character
    if d.open {
        overlay := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 77})
        logView := container.NewBorder(d.toggleBtn, nil, nil, nil, d.logList)
        d.stackContainer = container.NewStack(d.character, overlay, logView)
    } else {
        // Button at bottom, character fills remaining space above
        d.stackContainer = container.NewStack(
            container.NewBorder(nil, d.toggleBtn, nil, nil, d.character),
        )
    }
    return d.stackContainer
}
```

## Visual Comparison

### Before (Bug)
```
┌─────────────────────────┐
│                         │
│  ┌───────────────────┐  │
│  │                   │  │
│  │  [Activity Log]   │  │  ← button fills ENTIRE center panel
│  │  (covers fairy)   │  │
│  │                   │  │
│  └───────────────────┘  │
│                         │
└─────────────────────────┘
```

### After (Fix)
```
┌─────────────────────────┐
│                         │
│    Character widget     │
│    (fairy in jar)       │  ← character gets full space minus button height
│                         │
├─────────────────────────┤
│    [Activity Log]       │  ← button at natural height (~36px)
└─────────────────────────┘
```

## Test Strategy

### Behaviours

1. **Closed button height** — when closed, the toggle button does NOT fill the entire center panel; its height matches natural `widget.Button` MinSize.
2. **Character not overlapped** — when closed, the character widget's available height is reduced by the button height (character is padded up, not hidden behind button).
3. **Open overlay unchanged** — when opened, overlay still covers character area with semi-transparent background (existing Feature 070 behaviour preserved).
4. **No Stack stretch when closed** — the toggle button is not a direct child of a Stack container when the drawer is closed.

### TDD Micro-Loops

| # | Behaviour | Scope |
|---|---|---|
| 1 | Closed state uses Border layout (button at bottom, character above) | `internal/ui/` |
| 2 | Button height matches natural MinSize, not panel height | `internal/ui/` |
| 3 | Open state overlay behaviour preserved (semi-transparent bg) | `internal/ui/` |
| 4 | Character widget available space excludes button area when closed | `internal/ui/` |

## Files to Change

| File | Change |
|---|---|
| `internal/ui/activity_log_drawer.go` | Replace Stack with Border for closed state; update ToggleOpen and ContainerWithCharacter |
| `internal/ui/activity_log_drawer_test.go` | Update/add tests for button layout in closed state |
| `tests/ui/activity_log_acceptance_test.go` | Update acceptance tests for new layout behaviour |

## Acceptance Criteria

- [ ] When closed, "Activity Log" button anchored at bottom of center panel at natural button height
- [ ] When closed, character widget fills remaining space above the button (not behind it)
- [ ] When opened, overlay behaviour unchanged from Feature 070 (semi-transparent bg, log list, close button)
- [ ] Button height matches focus rail buttons (standard `widget.Button` MinSize)
- [ ] Existing overlay tests remain green (semi-transparent background, no Split)
- [ ] Character animations render correctly in the reduced space
