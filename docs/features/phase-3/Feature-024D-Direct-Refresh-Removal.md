# Feature 024 Hotfix D: Direct Refresh Removal

**Phase:** Phase-3-Feature-024D (bugfix)
**Status:** Done
**Depends on:** Feature 024C (refreshFunc Wiring)
**Packages:** `internal/ui/character/fairy/`

---

## Overview

Removes two direct `.Refresh()` calls on Fyne canvas objects (`bodyCircle`, `indicator`) in `FairyCharacter` that bypassed `fyne.Do()` thread safety. These calls caused repeated "Error in Fyne call thread, this should have been called in fyne.Do[AndWait]" warnings when animators invoked `SetBodyColor` or `TransitionTo` from goroutines.

## Symptom

- Running the Character UAT harness and triggering the "Starting" animation flooded the terminal with Fyne thread-safety errors
- Errors originated from `fairy.go:260` (`bodyCircle.Refresh()`) and `fairy.go:160` (`indicator.Refresh()`)

## Root Cause

Two methods called `.Refresh()` directly on canvas objects instead of routing through `refreshFunc` (which wraps calls in `fyne.Do()`):

1. **`SetBodyColor`** — called `f.bodyCircle.Refresh()` before `f.refreshFunc()`, making the first call redundant and unsafe
2. **`stopAndUpdateState`** — called `f.indicator.Refresh()` without any `refreshFunc()` call in the same code path

Both are called from animator goroutines (e.g., `StartupAnimator.runAnimationLoop`), which are not the Fyne main thread.

## Fix

1. **`SetBodyColor`**: Removed `f.bodyCircle.Refresh()` — the existing `f.refreshFunc()` call already refreshes the entire container including the body circle
2. **`stopAndUpdateState`**: Replaced `f.indicator.Refresh()` with `f.refreshFunc()` to route through `fyne.Do()`

## Test Coverage

Two regression tests added using log-capture technique (redirect `log.SetOutput` to a buffer, assert empty after method call):

- `TestSetBodyColorDoesNotDirectlyRefreshCanvasObject` — verifies `SetBodyColor` produces no Fyne error log output
- `TestTransitionToDoesNotDirectlyRefreshCanvasObject` — verifies `TransitionTo` produces no Fyne error log output

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (behavior 1) | Test Designer | ~306s | ~36,700 | 314482b |
| GREEN (behavior 1) | Implementer | ~26s | ~19,600 | 53170ca |
| RED (behavior 2) | Test Designer | ~56s | ~32,700 | 318e2e3 |
| GREEN (behavior 2) | Implementer | ~41s | ~21,000 | 9739566 |
