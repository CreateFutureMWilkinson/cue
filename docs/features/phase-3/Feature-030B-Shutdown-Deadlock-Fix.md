# Feature 030B — Shutdown Animator Deadlock Fix

## Overview

Fixed a deadlock in `ShutdownAnimator` where calling `Stop()` before `Start()` would block forever. This caused `TestConcurrentTransitions` to hang (10-minute timeout) when concurrent goroutines extracted an un-started animator and tried to stop it.

## Root Cause

`NewShutdownAnimator` eagerly created a `done` channel in the constructor:

```go
func NewShutdownAnimator(clock Clock) *ShutdownAnimator {
    return &ShutdownAnimator{
        clock: clock,
        done:  make(chan struct{}),  // BUG: never closed unless Start() is called
    }
}
```

`Stop()` waits on this channel (`<-done`), but no goroutine exists to close it if `Start()` was never called. In `TestConcurrentTransitions`, a race between goroutines could extract a newly-created (but not yet started) `ShutdownAnimator` from `currentAnimator` and call `Stop()`, causing a permanent deadlock.

## Fix

Removed the `done` channel initialization from the constructor, making it nil until `Start()` creates it. This matches `StartupAnimator`'s pattern. `Stop()` already handles `done == nil` by skipping the wait.

The `Shutdown()` codepath (which holds the app open until animation completes) is unaffected because it always calls `Start()` before `Done()`.

## Files Changed

| File | Change |
|---|---|
| `internal/ui/character/shutdown_animator.go` | Remove `done: make(chan struct{})` from constructor |
| `internal/ui/character/shutdown_animator_test.go` | Add `TestStopWithoutStartDoesNotBlock`, update `TestDoneChannelBeforeStart` to expect nil |

## Test Coverage

- `TestStopWithoutStartDoesNotBlock` — verifies Stop() returns within 1s without prior Start()
- `TestDoneChannelBeforeStart` — verifies Done() returns nil before Start()
- `TestConcurrentTransitions` — no longer deadlocks (was timing out at 10 minutes)

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 24s | 26,776 | 6a92097 |
| GREEN | Implementer | 82s | 33,554 | 1723e37 |
| REFACTOR | Refactorer | 32s | 27,052 | 1723e37 |
