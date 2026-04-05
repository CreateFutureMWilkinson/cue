# Feature 025-Hotfix-A — Animator Wiring

## Overview

Wires the six existing state animators (idle, startup, working, notify, error, shutdown) into `FairyCharacter.TransitionTo()` so that clicking state trigger buttons in the UAT harness — and state transitions in the main app — produce visible animations. Also fixes `SetPosition()` and `SetGlowIntensity()` to trigger container re-layout and glow alpha updates.

## Problem

1. `TransitionTo()` only updated a hidden indicator circle; animators were never instantiated or started.
2. `SetPosition()` and `SetGlowIntensity()` updated internal state without calling `container.Refresh()`, so the custom `fairyJarLayout` never repositioned circles.
3. `SetGlowIntensity()` stored the value but never applied it to glow layer alpha channels.
4. No production `Clock` implementation existed — only a `mockClock` in tests.

## Design Decisions

### Keep `NewFairyCharacter()` signature unchanged
Over 60 call sites reference `NewFairyCharacter()`. Adding a `Clock` parameter would be highly invasive. Instead, the constructor defaults to `WallClock{}` and exposes `SetClock(Clock)` for test injection.

### Injectable refresh callback
`refreshFunc` defaults to `func() { f.container.Refresh() }` (initialized as no-op, set post-construction). `DisableRefresh()` replaces it with a no-op for unit tests that don't have a running Fyne app.

### Async startup auto-transition
`StartupAnimator`'s `onComplete` callback fires from inside its goroutine. Calling `TransitionTo(StateIdle)` directly would deadlock (the goroutine can't exit while `Stop()` waits on it). Solution: `go f.TransitionTo(StateIdle)` dispatches asynchronously.

### Mutex-protected transitions
`sync.Mutex` on `FairyCharacter` serializes animator stop/start to prevent races from concurrent or async transitions (e.g., startup auto-transition racing with a manual button press).

### `Close()` on Character interface
Added `Close()` to the `Character` interface for clean animator goroutine shutdown. `NoOpCharacter.Close()` is a no-op.

## API

### New methods on FairyCharacter

| Method | Purpose |
|---|---|
| `SetClock(Clock)` | Inject mock clock for testing |
| `DisableRefresh()` | Replace refresh callback with no-op for tests |
| `Close()` | Stop current animator, nil it out |

### Modified methods

| Method | Change |
|---|---|
| `TransitionTo(state)` | Stops current animator, creates+starts new animator for state |
| `SetPosition(x, y)` | Now calls `refreshFunc()` to trigger re-layout |
| `SetBodyColor(c)` | Now calls `refreshFunc()` after `bodyCircle.Refresh()` |
| `SetGlowIntensity(intensity)` | Now updates glow layer alpha channels and calls `refreshFunc()` |
| `CurrentState()` | Now mutex-protected for thread safety |

### New types

| Type | Purpose |
|---|---|
| `WallClock` | Production `Clock` implementation wrapping `time.Now()` / `time.NewTicker()` |
| `wallTicker` | Production `Ticker` implementation wrapping `*time.Ticker` |

## Error Handling

No new error paths. Animator creation is infallible. `Close()` and `Stop()` are idempotent.

## Integration Points

- **UAT harness** — No changes needed; `TransitionTo()` now produces visible animations automatically.
- **CharacterPresenter** — Updated mock in tests to satisfy `Close()` interface method.
- **Main app** — `character.Register("fairy", ...)` call unchanged; fairy now animates when orchestrator activity events trigger state transitions.

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestWallClockNowReturnsReasonableTime` | WallClock.Now() is within 1s of time.Now() |
| `TestWallClockNewTickerDelivers` | Ticker channel fires within 2x interval |
| `TestTransitionToIdleStartsAnimator` | Position (0.5, 1.0), IdleBodyColor, non-zero glow |
| `TestTransitionToWorkingStartsAnimator` | WorkingBodyColor after entry, position drifts |
| `TestTransitionToErrorStartsAnimator` | ErrorBodyColor, position y=0.5 |
| `TestTransitionToNotifyStartsAnimator` | NotifyBodyColor |
| `TestTransitionStopsPreviousAnimator` | Working→Idle resets position to idle origin |
| `TestStartupAutoTransitionsToIdle` | State becomes Idle after 1.5s |
| `TestSetGlowIntensityUpdatesGlowAlpha` | Alpha=30 at intensity 1.0, alpha=0 at intensity 0.0 |
| `TestCloseStopsAnimator` | No panic, state preserved |
| `TestCloseIdempotent` | Double close, no panic |
| `TestCloseWithoutAnimator` | Fresh fairy close, no panic |
| `TestConcurrentTransitions` | 10 goroutines racing TransitionTo, no panic |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 108s | 54,403 | 58837ab |
| GREEN | Implementer | 4310s | 79,345 | 313aac8 |
| REFACTOR | Refactorer | 136s | 36,248 | 99e3dd3 |
