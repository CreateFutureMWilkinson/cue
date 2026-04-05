# Feature 16 — Three-Column Layout + Center View Router

## Overview

Restructures the Fyne GUI from a two-pane HSplit into a three-column layout matching the UI-SPEC.md wireframe, and introduces a `CenterViewRouter` state machine to control which view occupies the center column.

## Design Decisions

| Decision | Rationale |
|---|---|
| Nested HSplit for columns | Fyne has no native percentage-based grid; nested HSplit with offsets (0.1 outer, 0.667 inner) approximates 10/60/30 split |
| CenterViewRouter as simple state machine | Minimal surface area — one current view, one callback. No mutex needed (UI is single-threaded in Fyne) |
| Focus rail is placeholder | Feature 17 populates it with timer ring and navigation buttons |
| Removed "Review Buffered" button | Moves to notification panel expand state in Feature 18 |
| newFyneApp factory variable | Allows tests to inject `test.NewApp()` via `export_test.go` init(), avoiding headless display issues |
| Nil-safe presenter guards | NewMainWindow handles nil presenters gracefully so the API contract test can pass without full wiring |

## API

### CenterViewRouter

```go
type CenterView int

const (
    ViewCharacter CenterView = iota  // Default
    ViewPlan                          // Day planner (Feature 22)
    ViewWizard                        // Day planner wizard (Feature 22)
)

func NewCenterViewRouter() *CenterViewRouter
func (r *CenterViewRouter) CurrentView() CenterView
func (r *CenterViewRouter) NavigateTo(view CenterView)
func (r *CenterViewRouter) SetOnViewChange(fn func(CenterView))
```

### NewMainWindow (updated signature)

```go
func NewMainWindow(
    cfg config.GUIConfig,
    np *presenter.NotificationPresenter,
    ap *presenter.ActivityPresenter,
    fp *presenter.FeedbackPresenter,
    appP *presenter.AppPresenter,
    sp *presenter.SettingsPresenter,
    characterWidget fyne.CanvasObject,
    viewRouter *CenterViewRouter,       // NEW
) *MainWindow
```

## Error Handling

- Nil presenters produce empty placeholder widgets instead of panics
- Menu items are dynamically built based on available presenters
- CenterViewRouter tolerates nil callback (no-op on NavigateTo)

## Integration Points

| Feature | Dependency |
|---|---|
| Feature 17 (Focus Rail) | Populates the left column placeholder; reads CenterViewRouter for Back/Plan button visibility |
| Feature 18 (Notification Redesign) | Occupies right column; expand toggle hides center area |
| Feature 19 (Activity Log Drawer) | Activity log becomes a drawer within the center column |
| Feature 22 (Planner UI) | Registers ViewPlan and ViewWizard with CenterViewRouter |

## Test Coverage

| Suite | Tests | Coverage |
|---|---|---|
| CenterViewRouterSuite | 6 | Default view, navigation, callbacks, multi-nav, no-panic without callback |
| ThreeColumnLayoutSuite | 1 | API contract (NewMainWindow accepts CenterViewRouter) |

## Files Changed

| File | Change |
|---|---|
| `internal/ui/center_view_router.go` | New — state machine |
| `internal/ui/center_view_router_test.go` | New — 6 tests |
| `internal/ui/window_layout_test.go` | New — 1 test |
| `internal/ui/export_test.go` | New — test helper for headless Fyne |
| `internal/ui/window.go` | Modified — three-column layout, new parameter |
| `cmd/cue/main.go` | Modified — pass CenterViewRouter to NewMainWindow |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 89s | 22,943 | e3c364b |
| GREEN | Implementer | 476s | 46,629 | 49308c9 |
| REFACTOR | Refactorer | 87s | 41,773 | 32d6cc7 |
