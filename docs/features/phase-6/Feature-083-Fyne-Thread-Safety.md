# Feature 083 — Fyne Call Thread Safety Violations

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | High |
| Status | Done |
| Depends on | — |
| UI Tests | No |

## Problem

The application produces Fyne threading errors on stdout:

```
*** Error in Fyne call thread, this should have been called in fyne.Do[AndWait] ***
  From: /home/delphicokami/Projects/cue/internal/ui/window.go:26
```

Fyne requires all canvas/widget modifications to happen on the event loop thread via `fyne.Do()` or `fyne.DoAndWait()`. Several call sites modified UI objects from non-Fyne threads.

## Affected Call Sites

### Fixed

1. **`MainWindow.switchCenterView()`** — Already wrapped in `fyne.Do()` prior to this feature (Feature 077).

2. **`MainWindow.SetCharacterWidget()`** (window.go)
   - Directly modified `centerStack.Objects` and called `Refresh()` without thread scheduling.
   - **Fix:** Wrapped container mutations in `fyne.Do()`.

3. **`AppBinder` callbacks** (app_binder.go)
   - `SetOnStepChange` callback called `Refresh()` on views and `SetActivePlan()` from presenter callbacks running on background goroutines.
   - **Fix:** Introduced injectable `UIScheduler` type and `scheduleUI()` helper. All view mutations in the step-change callback dispatch through the scheduler. Production wires `fyne.Do`; tests inject a synchronous function.

4. **`TimerLoop.TickOnce()`** (timer_loop.go)
   - Called `widget.SetProgress()`, `widget.SetFlashVisible()`, `taskView.SetCurrentTask()` directly from background goroutine.
   - **Fix:** Same injectable `UIScheduler` pattern. Timer state values are captured before the scheduled closure to avoid race conditions.

### Already Correct (No Changes Needed)

- `window.go` periodic refresh — correctly wraps in `fyne.Do()`
- `main.go` character refresh hook — correctly wraps in `fyne.Do()`

## Design Decisions

### Injectable UIScheduler vs Direct `fyne.Do()`

`SetCharacterWidget` could use `fyne.Do()` directly because its tests already use `test.NewApp()` (which initialises the Fyne test driver). `AppBinder` and `TimerLoop` tests use plain mocks without a Fyne app, so `fyne.Do()` would panic. The `UIScheduler` function type makes the scheduling dependency explicit and testable:

```go
type UIScheduler func(func())
```

Both `AppBinder` and `TimerLoop` expose `SetUIScheduler(fn UIScheduler)`. When unset, calls dispatch directly (backward compatible). In `main.go`, both are wired to `fyne.Do`.

### Timer State Capture

`TimerLoop.TickOnce()` reads timer state (elapsed fraction, flash visibility, task name) before entering the scheduled closure. This prevents race conditions where the timer state changes between scheduling and execution.

## Files Changed

| File | Change |
|---|---|
| `internal/ui/window.go` | `SetCharacterWidget` wraps container mutations in `fyne.Do()` |
| `internal/ui/app_binder.go` | Added `UIScheduler` type, `SetUIScheduler`, `scheduleUI` helper; step-change callback dispatches through scheduler |
| `internal/ui/timer_loop.go` | Added `SetUIScheduler`, `scheduleUI` helper; `TickOnce` dispatches widget updates through scheduler |
| `internal/ui/app_binder_test.go` | Added `TestBindStepChangeCallbackUsesUIScheduler` |
| `internal/ui/timer_loop_test.go` | Added `TestTimerLoopTickOnceUsesUIScheduler` |
| `cmd/cue/main.go` | Wires `fyne.Do` as UIScheduler for AppBinder and TimerLoop |

## Test Coverage

| Behavior | Test |
|---|---|
| SetCharacterWidget thread safety | Existing `TestSetCharacterWidgetSwapsCenterContent` (regression) |
| AppBinder UIScheduler dispatch | `TestBindStepChangeCallbackUsesUIScheduler` |
| TimerLoop UIScheduler dispatch | `TestTimerLoopTickOnceUsesUIScheduler` |

## TDD Agent Stats

| Behavior | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| SetCharacterWidget | GREEN | Implementer | ~26s | ~23,000 | e7239ea |
| AppBinder UIScheduler | RED | Test Designer | ~45s | ~31,500 | 3821427 |
| AppBinder UIScheduler | GREEN | Implementer | ~30s | ~25,000 | 3c3552b |
| TimerLoop UIScheduler | RED | Test Designer | ~41s | ~27,300 | 2c0b6be |
| TimerLoop UIScheduler | GREEN | Implementer | ~29s | ~23,500 | 9e550f6 |
| Production wiring | GREEN | orchestrator | manual | — | 1195e61 |
