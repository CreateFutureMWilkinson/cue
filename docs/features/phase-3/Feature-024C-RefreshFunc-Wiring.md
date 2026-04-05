# Feature 024 Hotfix C: refreshFunc Wiring

**Phase:** Phase-3-Feature-024C (bugfix)
**Status:** Done
**Depends on:** Feature 024B (Fairy Refresh Thread Safety)
**Packages:** `internal/ui/character/fairy/`

---

## Overview

Fixes a bug where `refreshFunc` in `FairyCharacter` was initialized as a no-op (`func() {}`) and never replaced with an actual refresh function after container creation. This caused all position-based animations (working drift, notify dart, error vibrate) to be invisible — internal `posX`/`posY` values updated correctly but the Fyne layout was never told to reposition circles.

## Symptom

- Fairy character visible in UAT harness and main app
- Glow breathing (color/alpha changes) partially visible via `bodyCircle.Refresh()` side-effect
- **No movement animations observable** — fairy stayed fixed at its initial position regardless of state transitions
- Working (drift), Notifying (dart), Error (vibrate) states all appeared static

## Root Cause

In `NewFairyCharacter()`, the struct literal set `refreshFunc: func() {}` (line 110). After the container was created (line 123), `refreshFunc` was never reassigned. The `SetPosition()`, `SetBodyColor()`, and `SetGlowIntensity()` methods all called `refreshFunc()` at the end, expecting it to trigger a container re-layout — but the no-op meant nothing happened.

`SetBodyColor()` appeared to work because it also called `bodyCircle.Refresh()` directly, which triggered a repaint of that single object.

## Fix

After container creation, wire `refreshFunc` to call `fyne.Do(func() { f.container.Refresh() })` with a `fyne.CurrentApp() != nil` guard. The guard is necessary because `fyne.Do()` panics with a nil pointer dereference when no Fyne app is running (as in unit tests).

```go
f.container = container.New(&fairyJarLayout{fairy: f}, objects...)
f.refreshFunc = func() {
    if fyne.CurrentApp() != nil {
        fyne.Do(func() { f.container.Refresh() })
    }
}
```

Also added `SetRefreshHook(fn func())` for test observability of refresh calls.

## Design Decisions

| Decision | Rationale |
|---|---|
| `fyne.CurrentApp()` nil guard | `fyne.Do()` panics without a running app; 60+ test sites create `FairyCharacter` without a Fyne app. Guard is cheaper than adding `DisableRefresh()` to every test. |
| `fyne.Do()` wrapper | Required for Wayland thread safety — animator goroutines must not call `Refresh()` directly on the main thread. |
| `SetRefreshHook` method | Allows tests to inject a counting callback to verify refresh is triggered without needing a running Fyne app. |
| Removed dead no-op init | The `refreshFunc: func() {}` in the struct literal was immediately overwritten — dead code removed in refactor phase. |

## Test Coverage

| Test | Verifies |
|---|---|
| `TestRefreshFuncCalledOnSetPosition` | `SetPosition` triggers `refreshFunc` via `SetRefreshHook` counter |

Existing tests continue to pass unchanged — the nil guard ensures no panic without a Fyne app.

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | test-designer | ~39s | ~28,000 | c7d2cb3 |
| GREEN | implementer + orchestrator | ~107s | ~30,000 | a097582 |
| REFACTOR | refactorer | ~43s | ~23,000 | 48d4c84 |
