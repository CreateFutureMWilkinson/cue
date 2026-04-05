# Feature 026: Fairy Idle State

**Phase:** Phase-3-Feature-026
**Status:** Done
**Packages:** `internal/ui/character/`

---

## Overview

Implements the fairy's idle/rest behavior: stationary on the jar floor with a steady 3-second breathing cycle expressed through glow brightening and dimming. The fairy's body remains at `#006100` (darkest operational green). This is the default state and the state the fairy returns to after the CharacterPresenter's 5-second decay timer.

## Design Decisions

- **Stationary on jar floor** — position fixed at normalized (0.5, 1.0), bottom-center of the jar interior. No movement.
- **Breathing through glow only** — the body color stays constant at `#006100`. The breathing effect is achieved by oscillating glow intensity between a minimum (~0.3) and maximum (~0.8) using a smooth sinusoidal curve.
- **3-second full cycle** — one complete inhale + exhale. Glow intensity follows `sin(2π * t / 3.0)` mapped to the [min, max] range. This produces a natural, calming rhythm.
- **Animation goroutine with ticker** — a background goroutine drives the animation at ~30 FPS (33ms tick). Each tick recalculates glow intensity and calls `canvas.Refresh()`. The goroutine is started/stopped on state transitions.
- **Smooth transition into idle** — when transitioning from another state, the fairy should settle to the floor position and begin breathing from the current glow intensity, not snap to a fixed value.
- **Animation interface** — introduces a `StateAnimator` internal interface that each state's animation logic implements. The `FairyCharacter` delegates to the active animator on `TransitionTo()`.

## Animation Specification

### Breathing Cycle

```
Glow intensity over time (3-second cycle):

1.0 ─
0.8 ─ ·  ·  ·  ·  ·  ·  ·  ·  ·  peak
     ╱    ╲
0.5 ─      ╲      ╱
              ╲  ╱
0.3 ─ ·  ·  ·  ╳  ·  ·  ·  ·  ·  trough
     │         │         │
     0s       1.5s      3.0s
```

```go
const (
    idleBreathCycleSec  = 3.0
    idleGlowMin         = 0.3
    idleGlowMax         = 0.8
    animationFPS        = 30
    animationTickMs     = 1000 / animationFPS  // ~33ms
)

// Glow intensity at time t:
func idleGlowIntensity(t float64) float64 {
    normalized := math.Sin(2 * math.Pi * t / idleBreathCycleSec)
    return idleGlowMin + (idleGlowMax - idleGlowMin) * (normalized + 1.0) / 2.0
}
```

### Position

```go
// Fixed at jar floor center
const (
    idlePositionX = 0.5
    idlePositionY = 1.0  // bottom of jar interior
)
```

### Color

```go
var idleBodyColor = color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}  // #006100
```

## API

### StateAnimator Interface (Internal)

```go
// StateAnimator drives per-state animation logic.
type StateAnimator interface {
    // Start begins the animation loop. Called when entering this state.
    Start(fairy *FairyCharacter)

    // Stop halts the animation loop. Called when leaving this state.
    Stop()

    // State returns the CharacterState this animator handles.
    State() CharacterState
}
```

### IdleAnimator

```go
type IdleAnimator struct {
    fairy    *FairyCharacter
    cancel   context.CancelFunc
    running  bool
    mu       sync.Mutex
    clock    Clock           // injectable for testing
}

func NewIdleAnimator(clock Clock) *IdleAnimator
func (a *IdleAnimator) Start(fairy *FairyCharacter)
func (a *IdleAnimator) Stop()
func (a *IdleAnimator) State() CharacterState  // returns StateIdle
```

### FairyCharacter Changes

```go
// TransitionTo now delegates to the appropriate StateAnimator.
// Stops the current animator, starts the new one.
func (f *FairyCharacter) TransitionTo(state CharacterState)
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Animation goroutine leak | Context-based cancellation ensures goroutine exits on Stop() |
| Rapid state transitions | Stop() is synchronous — waits for goroutine exit before Start() of new state |
| Clock injection nil | Default to `time.Now` / `time.NewTicker` |

## Integration Points

- **Jar Rendering (Feature 025):** Uses `SetPosition()`, `SetGlowIntensity()`, and `SetBodyColor()` to drive the visual state.
- **CharacterPresenter (Feature 014):** Calls `TransitionTo(StateIdle)` after 5-second decay. No changes to presenter needed.
- **UAT Harness (Feature 024):** Idle state triggered via "Idle" button for visual validation of breathing cycle.
- **Subsequent States (Features 023–026):** Each implements `StateAnimator` and follows the same Start/Stop pattern.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character` | `IdleAnimatorSuite` | Breathing glow min/max values at cycle extremes, 3-second cycle period, position stays at (0.5, 1.0), body color is #006100, Start/Stop lifecycle, no goroutine leak after Stop, glow intensity function correctness (table-driven at t=0, 0.75, 1.5, 2.25, 3.0), smooth transition from arbitrary starting glow |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Idle Animator | RED | Test Designer | 125s | 32,418 | 7be13b9 |
| Idle Animator | GREEN | Implementer | 68s | 34,796 | 3ba7378 |
| Idle Animator | REFACTOR | Refactorer | 220s | 39,896 | 2cebeec |
