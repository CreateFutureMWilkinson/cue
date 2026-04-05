# Feature 027: Fairy Working State

**Phase:** Phase-3-Feature-027
**Status:** Planned
**Packages:** `internal/ui/character/`

---

## Overview

Implements the fairy's working state behavior: gentle pseudo-random drift around the jar interior in lazy circles/spirals, completing one "circuit" roughly every 4 seconds. The fairy brightens from the idle color to an intermediate green. The breathing glow cycle from idle state is maintained at the same 3-second rate.

## Design Decisions

- **Pseudo-random smooth movement** — uses Perlin noise or layered sinusoidal functions (multiple sine waves at different frequencies and phases) to produce organic-looking drift. Pure `math/rand` would produce jittery movement; layered sine produces smooth, natural paths.
- **4-second circuit** — the fairy completes one approximate loop around the jar interior every ~4 seconds. The path is never perfectly circular — noise offsets create variation so no two circuits look identical.
- **Constrained to jar interior** — position is clamped to the jar interior bounds (Feature 025). The fairy body never clips through the glass.
- **Breathing maintained** — the same 3-second glow breathing cycle from idle state continues during working. This provides visual continuity across states.
- **Intermediate body color** — working state uses a green between idle (#006100) and notifying (#00C300). Calculated as midpoint: `#009200`.
- **Smooth transition from idle** — when entering working state from idle, the fairy rises from the jar floor and begins drifting. The rise is animated over ~0.5 seconds, not instantaneous.

## Animation Specification

### Movement Pattern

```go
const (
    workingCircuitSec  = 4.0    // approximate time for one circuit
    workingDriftRadius = 0.35   // normalized radius of drift area (0.0–0.5)
)

// Position at time t using layered sinusoidal movement:
func workingPosition(t float64) (x, y float64) {
    // Primary circular motion (4-second period)
    px := 0.5 + workingDriftRadius * math.Sin(2*math.Pi*t/workingCircuitSec)
    py := 0.5 + workingDriftRadius * math.Cos(2*math.Pi*t/workingCircuitSec)

    // Secondary noise (slower, smaller amplitude for organic variation)
    nx := 0.08 * math.Sin(2*math.Pi*t/7.3 + 1.2)   // 7.3s period, offset phase
    ny := 0.08 * math.Cos(2*math.Pi*t/5.7 + 2.8)   // 5.7s period, offset phase

    // Tertiary wobble (fastest, smallest)
    wx := 0.03 * math.Sin(2*math.Pi*t/1.9 + 0.7)
    wy := 0.03 * math.Cos(2*math.Pi*t/2.3 + 3.1)

    x = clamp(px + nx + wx, 0.0, 1.0)
    y = clamp(py + ny + wy, 0.0, 1.0)
    return
}
```

The layered approach uses three sine waves per axis at incommensurate frequencies (4.0, 7.3, 5.7, 1.9, 2.3 seconds), producing quasi-random paths that never exactly repeat.

### Breathing Cycle

Same as idle state — 3-second sinusoidal glow oscillation:

```go
const (
    workingBreathCycleSec = 3.0     // same as idle
    workingGlowMin        = 0.3
    workingGlowMax        = 0.8
)
```

### Color

```go
var workingBodyColor = color.RGBA{R: 0x00, G: 0x92, B: 0x00, A: 0xFF}  // #009200
```

### Entry Transition

When entering working from idle:
1. Fairy rises from (0.5, 1.0) to the first drift position over 0.5 seconds (linear interpolation).
2. Body color transitions from #006100 to #009200 over the same 0.5 seconds.
3. Drift motion begins once the rise completes.

```go
const workingEntryDurationSec = 0.5
```

## API

### WorkingAnimator

```go
type WorkingAnimator struct {
    fairy     *FairyCharacter
    cancel    context.CancelFunc
    running   bool
    mu        sync.Mutex
    clock     Clock
    startTime time.Time
}

func NewWorkingAnimator(clock Clock) *WorkingAnimator
func (a *WorkingAnimator) Start(fairy *FairyCharacter)
func (a *WorkingAnimator) Stop()
func (a *WorkingAnimator) State() CharacterState  // returns StateWorking
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Position calculation exceeds bounds | Clamped to [0.0, 1.0] via SetPosition |
| Rapid idle→working→idle transitions | Entry transition interrupted cleanly; Stop() cancels context |
| Clock drift during long running | Position based on elapsed time, not frame count — self-correcting |

## Integration Points

- **Jar Rendering (Feature 025):** Uses `SetPosition()`, `SetGlowIntensity()`, `SetBodyColor()` for visual updates.
- **Idle State (Feature 026):** Shares the `StateAnimator` interface and breathing constants. Transition from idle triggers the rise animation.
- **CharacterPresenter (Feature 014):** Calls `TransitionTo(StateWorking)` on non-error, non-notification events.
- **UAT Harness (Feature 024):** Working state triggered via "Working" button for visual validation of drift patterns.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character` | `WorkingAnimatorSuite` | Position stays within jar bounds over full circuit, 4-second approximate periodicity, body color is #009200, breathing glow maintained (same min/max as idle), entry transition interpolates position from floor to drift, entry transition interpolates color from idle to working, Start/Stop lifecycle, no goroutine leak, position function produces different values at different times (not static) |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Working Animator | RED | Test Designer | — | — | — |
| Working Animator | GREEN | Implementer | — | — | — |
| Working Animator | REFACTOR | Refactorer | — | — | — |
