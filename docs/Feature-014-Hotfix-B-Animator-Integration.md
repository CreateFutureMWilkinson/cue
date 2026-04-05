# Feature 014-Hotfix-B: Fairy Animator Integration

**Phase:** Phase-3-Feature-014-Hotfix-B
**Status:** Planned
**Package:** `internal/ui/character/`
**Parent:** Feature 014 (Character System)

---

## Overview

All Phase 3 fairy animators (idle, working, notification, error, startup, shutdown) were implemented as standalone tested classes but **never wired into FairyCharacter**. The `TransitionTo()` method only updates the state enum and indicator color — it does not create, start, or delegate to any animator. Consequently, all animation code is dead code from the user's perspective: no breathing glow, no drift, no darting, no vibration, no lifecycle transitions.

This hotfix connects the animators to FairyCharacter and ensures visual mutations (position, body color, glow intensity) actually refresh the Fyne canvas.

## Root Cause

`FairyCharacter.TransitionTo()` at `fairy.go:128-133`:
```go
func (f *FairyCharacter) TransitionTo(state CharacterState) {
    f.state = state
    f.indicator.FillColor = stateColor(state)
    f.indicator.Refresh()
}
```

This updates the state enum and a debug indicator circle. It does not:
1. Stop the previous animator
2. Create/start the new state's animator
3. Pass the fairy to the animator for property updates

## Issues to Fix

### 1. FairyCharacter Has No Animator Field

The struct needs a field to track the currently active animator so it can be stopped on state transitions.

```go
type FairyCharacter struct {
    // ... existing fields ...
    currentAnimator StateAnimator  // NEW: active animation goroutine
}
```

### 2. TransitionTo() Must Delegate to Animators

```go
func (f *FairyCharacter) TransitionTo(state CharacterState) {
    // Stop previous animator
    if f.currentAnimator != nil {
        f.currentAnimator.Stop()
    }

    f.state = state
    f.indicator.FillColor = stateColor(state)
    f.indicator.Refresh()

    // Start new animator for the target state
    f.currentAnimator = f.createAnimator(state)
    if f.currentAnimator != nil {
        f.currentAnimator.Start()
    }
}
```

Where `createAnimator` maps states to animator constructors:
- `StateIdle` → `NewIdleAnimator(f, clock, ticker)`
- `StateWorking` → `NewWorkingAnimator(f, clock)`
- `StateNotifying` → `NewNotifyAnimator(f, clock)`
- `StateError` → `NewErrorAnimator(f, clock)`
- `StateStarting` → `NewStartupAnimator(f, clock, onComplete)`
- `StateShuttingDown` → `NewShutdownAnimator(f, clock)`

### 3. SetPosition/SetBodyColor/SetGlowIntensity Must Refresh Canvas

Currently these methods update internal fields but never trigger a visual refresh.

**SetPosition** (`fairy.go:143-146`):
```go
func (f *FairyCharacter) SetPosition(x, y float64) {
    f.posX = x
    f.posY = y
    // MISSING: trigger layout refresh so body/glow circles move
}
```

Fix: After updating position fields, call the custom layout's `Layout()` to reposition circles, then refresh the container.

**SetBodyColor** (`fairy.go:148-152`):
```go
func (f *FairyCharacter) SetBodyColor(c color.Color) {
    f.bodyColor = c
    f.bodyCircle.FillColor = c
    // MISSING: f.bodyCircle.Refresh()
}
```

Fix: Add `f.bodyCircle.Refresh()` after color update.

**SetGlowIntensity** (`fairy.go:159-162`):
```go
func (f *FairyCharacter) SetGlowIntensity(intensity float64) {
    f.glowIntensity = intensity
    // MISSING: update glow layer alpha values and refresh
}
```

Fix: Modulate each glow layer's alpha based on intensity and the layer's base alpha, then refresh each circle.

### 4. Glow Layer Alpha Must Be Dynamic

Current: All 8 glow layers created with hardcoded `glowAlpha = 30`.

Spec: "8 concentric circles... innermost: higher alpha, outermost: lower alpha" with intensity modulating the overall brightness.

Fix: Store base alpha per layer (graduated from ~128 inner to ~16 outer). `SetGlowIntensity(intensity)` multiplies each layer's base alpha by intensity (0.0–1.0) and updates the circle color.

```go
// On creation: store base alphas
f.glowBaseAlphas = []uint8{128, 112, 96, 80, 64, 48, 32, 16}

// On SetGlowIntensity:
for i, gl := range f.glowLayers {
    alpha := uint8(float64(f.glowBaseAlphas[i]) * intensity)
    gl.FillColor = color.RGBA{R: r, G: g, B: b, A: alpha}
    gl.Refresh()
}
```

### 5. Startup Animation on App Launch

Currently `main.go` creates the fairy and transitions to idle immediately. The startup animator (dormant → idle over 1.5s) is never triggered.

Fix: On fairy creation, start in `StateStarting`. The `StartupAnimator` transitions to idle after 1.5s via its `onComplete` callback.

### 6. Shutdown Animation on App Close

Currently `main.go` has no shutdown animation. The `ShutdownAnimator` exists but `FairyCharacter` has no `Shutdown()` method.

Fix: Add `Shutdown() <-chan struct{}` to `FairyCharacter`. When called, it transitions to `StateShuttingDown`, starts `ShutdownAnimator`, and returns a done channel. `main.go` waits on this channel before closing the window.

### 7. Thread Safety for Fyne Canvas Updates

Animators run in background goroutines. Fyne requires canvas mutations from the main thread (especially on Wayland — see Feature-024-Hotfix-A).

Fix: All visual refresh calls (`circle.Refresh()`, `container.Refresh()`) must go through `fyne.Do()` or `canvas.Refresh()` which is thread-safe in recent Fyne versions. Verify each refresh call site.

## Files

| File | Action |
|---|---|
| `internal/ui/character/fairy.go` | Modify — add animator field, fix TransitionTo(), fix Set* methods to refresh canvas, dynamic glow alphas, add Shutdown() |
| `internal/ui/character/fairy_test.go` | Modify — add tests for animator delegation, canvas refresh verification |
| `internal/ui/character/fairy_jar_test.go` | Modify — update glow layer expectations if alpha structure changes |
| `cmd/cue/main.go` | Modify — startup animation (start in StateStarting), shutdown animation (wait on Shutdown channel) |

## Test Coverage

- TransitionTo(StateIdle) starts IdleAnimator (verify Start() called on mock)
- TransitionTo(StateWorking) stops previous animator, starts WorkingAnimator
- TransitionTo while no previous animator — no panic
- SetPosition triggers layout refresh (verify container refreshed)
- SetBodyColor triggers circle refresh
- SetGlowIntensity updates glow layer alphas proportionally
- Glow layers have graduated base alphas (inner brighter than outer)
- Shutdown() returns done channel that closes after animation completes
- Startup → Idle transition fires automatically after 1.5s
- Thread safety: Set* methods safe to call from goroutine (fyne.Do wrapping)
