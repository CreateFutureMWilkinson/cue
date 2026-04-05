# Feature 024 Hotfix A: Wayland Thread Safety

**Phase:** Phase-3-Feature-024-Hotfix-A (bugfix)
**Status:** Done
**Packages:** `cmd/character-uat/`

---

## Overview

Fixes Fyne thread-safety violation in the character UAT harness that caused runtime errors on Wayland. The `updateFPSLoop` goroutine called `widget.Label.SetText()` directly from a background thread, but Fyne requires all widget mutations to go through `fyne.Do()`.

## Symptom

```
*** Error in Fyne call thread, this should have been called in fyne.Do[AndWait] ***
  From: cmd/character-uat/uat_window.go:209
```

Reproduced on Linux/Wayland. X11 silently tolerated the violation.

## Root Cause

`UATWindow.updateFPSLoop()` ran in a goroutine (started in `Run()`) and called `w.fpsLabel.SetText(...)` directly — a Fyne widget mutation from a non-UI thread.

## Fix

Extracted the FPS update loop into a standalone `FPSLoop` type with an injectable `OnFPSUpdate` callback. The UAT window wires this callback to `fyne.Do(func() { label.SetText(text) })`, ensuring thread safety.

### New Type: `FPSLoop`

```go
type FPSLoopConfig struct {
    Counter     *FPSCounter
    Interval    time.Duration
    OnFPSUpdate func(string)    // called with formatted "FPS: %.1f" text
}
```

- `NewFPSLoop(cfg)` validates all fields (panics on nil/zero)
- `Start()` launches the background goroutine
- `Stop()` is idempotent (mutex-guarded, safe to call multiple times)

### UATWindow Changes

- Replaced `stopFPS chan struct{}` + `updateFPSLoop()` with `fpsLoop *FPSLoop`
- `Run()` calls `fpsLoop.Start()` / `fpsLoop.Stop()`
- Callback: `fyne.Do(func() { w.fpsLabel.SetText(text) })`

## Design Decisions

- **Callback injection over direct fix** — wrapping `SetText` in `fyne.Do()` inline would have been a one-line fix, but extracting `FPSLoop` makes the threading boundary testable without a real Fyne app.
- **Idempotent Stop** — `sync.Once`-style guard prevents double-close panic on the stop channel.
- **Constructor validation** — panics on nil Counter/OnFPSUpdate or zero Interval to fail fast during wiring rather than at runtime.

## Test Coverage

| Suite | Tests | Focus |
|---|---|---|
| `FPSUpdateThreadSafetySuite` | 5 | Callback invocation, formatted text, clean stop, multiple stop safety, config validation |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 73s | 26,789 | d533f84 |
| GREEN | Implementer | 103s | 29,102 | dca8d1e |
| REFACTOR | Refactorer | 77s | 27,821 | 39a63b4 |
