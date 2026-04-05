# Feature 017 — Focus Rail + Countdown Timer

## Overview

Implements the persistent left column (focus rail) of the three-column layout and a custom countdown timer ring widget. The focus rail provides navigation buttons that respond to the CenterViewRouter state, a task label and Done button that appear when a plan is active, and a Review button visible only when notifications are expanded. The countdown timer is a custom Fyne widget with 45 line segments at 8-degree intervals, supporting progress tracking with distinct visual states for future, current, and elapsed segments.

## Design Decisions

| Decision | Rationale |
|---|---|
| 45 segments at 8-degree intervals | Matches UI-SPEC.md countdown timer spec; 45 * 8 = 360 degrees gives full ring coverage |
| Cardinal lines 3x, diagonal 2x, regular 1x | Visual hierarchy — cardinals (0/90/180/270) are most prominent at 36px, diagonals at 24px, regular at 12px |
| Timer hidden by default | UI-SPEC requires timer hidden until an active plan exists; `SetActivePlan(true)` reveals it |
| FocusRail listens to CenterViewRouter | Plan button visible in Character view, Back button visible in Plan/Wizard — driven by `SetOnViewChange` callback |
| Review button decoupled from view state | Review visibility is controlled separately via `SetNotificationsExpanded`, not by CenterViewRouter |
| Callback-based Done/Review | `SetOnDone` and `SetOnReview` allow parent code to wire behavior without the rail knowing about external systems |
| Progress clamped to [0.0, 1.0] | Prevents invalid rendering state from out-of-range progress values |
| Minimal renderer (placeholder) | Timer rendering logic is a shell; full rendering with canvas lines deferred to Feature 022 (Planner UI) when the timer is actively used |
| Yellow #FFCE1B for future segments | Matches UI-SPEC.md design token for timer color |
| Elapsed segments use alpha=64 | Dimmed appearance clearly distinguishes past from future without a second color |

## API

### CountdownTimer

```go
type SegmentState int  // SegmentFuture, SegmentCurrent, SegmentElapsed

type SegmentInfo struct {
    AngleDeg float64
    Length   float64
    State    SegmentState
    Color    color.NRGBA
}

func NewCountdownTimer() *CountdownTimer
func (t *CountdownTimer) Segments() []SegmentInfo
func (t *CountdownTimer) SetProgress(p float64)
func (t *CountdownTimer) Reset()
func (t *CountdownTimer) MinSize() fyne.Size
func (t *CountdownTimer) CreateRenderer() fyne.WidgetRenderer
```

### FocusRail

```go
func NewFocusRail(router *CenterViewRouter) *FocusRail
func (r *FocusRail) PlanButton() *widget.Button
func (r *FocusRail) BackButton() *widget.Button
func (r *FocusRail) DoneButton() *widget.Button
func (r *FocusRail) ReviewButton() *widget.Button
func (r *FocusRail) TaskLabel() *widget.Label
func (r *FocusRail) Timer() *CountdownTimer
func (r *FocusRail) SetActivePlan(active bool)
func (r *FocusRail) SetCurrentTask(task string)
func (r *FocusRail) SetNotificationsExpanded(expanded bool)
func (r *FocusRail) SetOnDone(fn func())
func (r *FocusRail) SetOnReview(fn func())
```

## Error Handling

- Progress values outside [0.0, 1.0] are clamped silently — no error, no panic
- Done and Review button taps with nil callbacks are no-ops
- CenterViewRouter nil callback is tolerated (inherited from Feature 016)

## Integration Points

| Feature | Dependency |
|---|---|
| Feature 016 (Three-Column Layout) | FocusRail occupies the left column placeholder; reads CenterViewRouter for navigation |
| Feature 018 (Notification Redesign) | `SetNotificationsExpanded` controls Review button visibility based on notification panel state |
| Feature 022 (Planner UI) | `SetActivePlan` / `SetCurrentTask` driven by planner; timer progress updated by Pomodoro engine |
| Feature 023 (Planner Audio) | Timer completion triggers audio alert through Done callback chain |

## Test Coverage

| Suite | Tests | Coverage |
|---|---|---|
| CountdownTimerSuite | 14 | Segment count, interval spacing, cardinal/diagonal/regular lengths, future/elapsed colors, progress states (0%, 50%, 100%), clamping, reset, MinSize, renderer, SegmentInfo fields |
| FocusRailSuite | 19 | Initial visibility, Plan/Back toggling on view change, Done/Review callbacks, task label, timer access, SetActivePlan show/hide, SetNotificationsExpanded, nil callback safety, multi-navigation |

## Files Changed

| File | Change |
|---|---|
| `internal/ui/countdown_timer.go` | New — custom Fyne widget with 45-segment ring |
| `internal/ui/countdown_timer_test.go` | New — 14 tests |
| `internal/ui/focus_rail.go` | New — persistent left column component |
| `internal/ui/focus_rail_test.go` | New — 19 tests |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | — | — | cccd027 |
| GREEN | Implementer | — | — | d3d1341 |
| REFACTOR | Refactorer | — | — | a0c481b |
