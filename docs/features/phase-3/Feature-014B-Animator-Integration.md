# Feature 014-Hotfix-B: Fairy Animator Integration

**Phase:** Phase-3-Feature-014-Hotfix-B
**Status:** Done
**Package:** `internal/ui/character/`
**Parent:** Feature 014 (Character System)

---

## Overview

Phase 3 fairy animators (idle, working, notification, error, startup, shutdown) were implemented as standalone tested classes but had incomplete integration with FairyCharacter. Specifically: glow layers used flat alpha values instead of graduated inner-to-outer brightness, there was no `Shutdown()` method for graceful app close, the app didn't trigger startup/shutdown animations, and the `Set*` mutation methods lacked thread safety for concurrent animator goroutine access.

This hotfix completes the animator integration by adding graduated glow alphas, a `Shutdown()` lifecycle method, startup/shutdown wiring in `main.go`, and mutex protection on all `Set*` methods.

## Changes

### 1. Graduated Glow Layer Base Alphas

Replaced flat `glowAlpha = 30` constant with per-layer graduated base alphas:

```go
glowBaseAlphas = [8]uint8{128, 112, 96, 80, 64, 48, 32, 16}
```

Index 0 (innermost) is brightest at 128; index 7 (outermost) is dimmest at 16. `SetGlowIntensity(intensity)` computes each layer's alpha as `base_alpha * intensity`, producing smooth breathing effects where inner layers glow brighter than outer.

### 2. Shutdown() Method

Added `Shutdown() <-chan struct{}` to `FairyCharacter`:
- Stops current animator
- Transitions to `StateShuttingDown`
- Creates and starts `ShutdownAnimator`
- Returns `ShutdownAnimator.Done()` channel for callers to await completion

### 3. Startup/Shutdown Wiring in main.go

- After character creation, `main.go` calls `TransitionTo(StateStarting)` to trigger the 1.5s dormant-to-idle fade animation
- On window close, `main.go` type-asserts to `shutdownable` interface and waits on `Shutdown()` channel before exiting

### 4. Thread Safety

Added `f.mu.Lock()`/`f.mu.Unlock()` to `SetPosition`, `SetBodyColor`, `SetGlowIntensity`, `Position`, and `GlowIntensity`.

Refactored `TransitionTo`, `Close`, and `Shutdown` to unlock the mutex before calling `animator.Stop()`, avoiding deadlock when animator goroutines call `Set*` methods that acquire the same mutex.

### 5. Code Quality (Refactor)

Extracted `stopAndUpdateState()` helper shared by `TransitionTo()` and `Shutdown()`, encapsulating the three-phase mutex pattern: (1) extract and clear animator under lock, (2) stop animator outside lock, (3) update state and create new animator under lock.

## Design Decisions

- **Interface assertion for Shutdown**: Rather than extending the `Character` interface (which would require changes to `NoOpCharacter` and all future implementations), `main.go` uses a local `shutdownable` interface assertion. This keeps the core interface minimal.
- **No fyne.Do() wrapping**: The design doc suggested `fyne.Do()` for thread safety, but `sync.Mutex` on the fairy's fields is sufficient since the mutex protects all shared state. Fyne canvas object `Refresh()` calls are already thread-safe in Fyne v2.7+.

## Files Modified

| File | Action |
|---|---|
| `internal/ui/character/fairy.go` | Graduated alphas, Shutdown(), mutex on Set*, stopAndUpdateState helper |
| `internal/ui/character/fairy_test.go` | 5 new tests: graduated alpha, half-intensity, inner>outer, shutdown channel, concurrent Set* |
| `cmd/cue/main.go` | Startup animation trigger, shutdown animation wait |

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestSetGlowIntensityUpdatesGlowAlphaGraduated` | At intensity=1.0, each layer has its graduated base alpha; at 0.0, all zeros |
| `TestSetGlowIntensityHalfGraduated` | At intensity=0.5, each alpha is half the base |
| `TestGlowAlphaInnerBrighterThanOuter` | Layer 0 alpha > layer 7 alpha at intensity=1.0 |
| `TestShutdownReturnsCompletionChannel` | Shutdown() transitions state and returns channel that closes after animation |
| `TestConcurrentSetMethodsDoNotPanic` | 30 goroutines calling Set* concurrently without panic |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 428s | 51,352 | b9cc323 |
| GREEN | Implementer + orchestrator | — | — | d3c2343 |
| REFACTOR | Refactorer | 76s | 33,344 | e573b14 |
| WIRING | orchestrator | — | — | 79c3baa |
