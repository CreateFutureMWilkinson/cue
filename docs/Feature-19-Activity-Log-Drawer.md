# Feature 19 — Activity Log Drawer

## Overview

Implements the activity log as a pull-up drawer at the bottom of the character area (center column). The drawer is hidden by default — only a toggle button is visible. When opened, it slides up to occupy ~40% of the character area, showing a real-time feed of system events. Events continue to accumulate in the buffer while the drawer is hidden; opening it shows the latest state.

## Design Decisions

| Decision | Rationale |
|---|---|
| Hidden by default | UI-SPEC.md requires drawer closed on startup; avoids visual clutter for ADHD-friendly design |
| Toggle button text changes | "Activity Log" when closed, "close ▼" when open — clear affordance for current state |
| VSplit offset 0.6 | Character widget gets top 60%, drawer gets bottom 40% — matches UI-SPEC.md "~40% of character area" |
| `ContainerWithCharacter` method | Encapsulates the VSplit layout so `window.go` doesn't need to know about drawer internals |
| Nil character widget fallback | Uses empty label placeholder when no character widget exists — prevents nil panics |
| Stack container for drawer body | Allows swapping between button-only (closed) and border layout (open) without recreating the container |
| Reuses `newActivityLog` | Existing `widget.List` creation with entry formatting and color rules is shared, not duplicated |

## API

### ActivityLogDrawer

```go
func NewActivityLogDrawer(ap *presenter.ActivityPresenter) *ActivityLogDrawer
func (d *ActivityLogDrawer) IsOpen() bool
func (d *ActivityLogDrawer) ToggleOpen()
func (d *ActivityLogDrawer) Container() fyne.CanvasObject
func (d *ActivityLogDrawer) ContainerWithCharacter(character fyne.CanvasObject) fyne.CanvasObject
```

## Error Handling

No new error paths — the drawer is a pure UI widget with no I/O. The underlying `ActivityPresenter` handles event buffering and FIFO eviction independently.

## Integration Points

- **ActivityPresenter** — provides entries and update callbacks via `newActivityLog()`
- **MainWindow** (`window.go`) — uses `ContainerWithCharacter(characterWidget)` to build the center pane
- **CenterViewRouter** — drawer is only visible when `ViewCharacter` is active (controlled by window layout, not the drawer itself)

## Test Coverage

7 tests in `ActivityLogDrawerSuite`:

| Test | Validates |
|---|---|
| `TestNewActivityLogDrawerNotNil` | Constructor returns non-nil |
| `TestDrawerDefaultHidden` | `IsOpen()` returns false by default |
| `TestDrawerToggleOpen` | `ToggleOpen()` sets `IsOpen()` to true |
| `TestDrawerToggleClose` | Toggling twice returns to closed |
| `TestDrawerContainerNotNil` | `Container()` returns non-nil |
| `TestDrawerContainerWithCharacterWidget` | `ContainerWithCharacter(widget)` returns non-nil |
| `TestDrawerContainerWithNilCharacterWidget` | `ContainerWithCharacter(nil)` returns non-nil |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 278s | 24,820 | 6edb636 |
| GREEN | Implementer | 37s | 21,165 | c69a786 |
| REFACTOR | Refactorer | 298s | 24,314 | d79e3c4 |
