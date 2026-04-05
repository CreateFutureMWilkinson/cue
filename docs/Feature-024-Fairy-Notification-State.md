# Feature 024: Fairy Notification State

**Phase:** Phase-3-Feature-024
**Status:** Planned
**Packages:** `internal/ui/character/`

---

## Overview

Implements the fairy's notification state behavior: rapid, erratic darting to random positions within the jar every 0.5 seconds, at the brightest green (#00C300), with an accelerated 1.5-second breathing glow cycle. This state signals to the user that urgent notifications need attention.

## Design Decisions

- **Random dart every 0.5 seconds** — the fairy jumps to a new pseudo-random position within the jar interior. Movement is instant (snap to new position), not interpolated — this creates the erratic, attention-grabbing visual.
- **Pseudo-random target positions** — each dart target is generated using `math/rand` seeded from the system clock. Positions are uniformly distributed across the jar interior bounds. No bias toward center or edges.
- **Brightest green #00C300** — maximum brightness in the green progression. Instant color transition on state entry.
- **1.5-second breathing cycle** — double the speed of idle/working breathing. Creates a sense of urgency through the faster pulse. Same sinusoidal shape, just compressed in time.
- **Glow range elevated** — minimum glow raised to 0.5 (from 0.3 in idle/working) so the fairy is never dim during notification state. Maximum stays at 0.9.
- **No smooth entry transition** — notification state snaps immediately to full brightness and begins darting. Urgency trumps smoothness.

## Animation Specification

### Movement Pattern

```go
const (
    notifyDartIntervalSec = 0.5  // new random position every 0.5 seconds
)

// Every 0.5 seconds, generate a new target position:
func notifyDartTarget(rng *rand.Rand) (x, y float64) {
    x = rng.Float64()  // 0.0–1.0 within jar interior
    y = rng.Float64()  // 0.0–1.0 within jar interior
    return
}
```

Between darts, the fairy holds position (no interpolation, no drift). The snap-to-position creates the characteristic erratic behavior.

### Breathing Cycle

```go
const (
    notifyBreathCycleSec = 1.5   // accelerated from 3.0
    notifyGlowMin        = 0.5   // elevated minimum
    notifyGlowMax        = 0.9   // near-maximum
)

func notifyGlowIntensity(t float64) float64 {
    normalized := math.Sin(2 * math.Pi * t / notifyBreathCycleSec)
    return notifyGlowMin + (notifyGlowMax - notifyGlowMin) * (normalized + 1.0) / 2.0
}
```

### Color

```go
var notifyBodyColor = color.RGBA{R: 0x00, G: 0xC3, B: 0x00, A: 0xFF}  // #00C300
```

### Entry Behavior

On transition to notification state:
1. Body color immediately set to #00C300.
2. Glow intensity jumps to `notifyGlowMax` (0.9).
3. First dart to random position occurs immediately.
4. 0.5-second dart timer begins.

## API

### NotifyAnimator

```go
type NotifyAnimator struct {
    fairy     *FairyCharacter
    cancel    context.CancelFunc
    running   bool
    mu        sync.Mutex
    clock     Clock
    rng       *rand.Rand
}

func NewNotifyAnimator(clock Clock, rng *rand.Rand) *NotifyAnimator
func (a *NotifyAnimator) Start(fairy *FairyCharacter)
func (a *NotifyAnimator) Stop()
func (a *NotifyAnimator) State() CharacterState  // returns StateNotifying
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Random position at bounds edge | Clamped via SetPosition (0.0–1.0) |
| Rapid notification→notification re-entry | Stop + Start cycles cleanly; dart timer resets |
| Nil RNG | Default to `rand.New(rand.NewSource(time.Now().UnixNano()))` |

## Integration Points

- **Jar Rendering (Feature 021):** Uses `SetPosition()`, `SetGlowIntensity()`, `SetBodyColor()` for dart and glow updates.
- **Working State (Feature 023):** Shares `StateAnimator` interface. Transition from working to notifying is immediate (no interpolation).
- **CharacterPresenter (Feature 014):** Calls `TransitionTo(StateNotifying)` when activity event contains "NOTIFIED".
- **UAT Harness (Feature 020):** Notification state triggered via "Notifying" button for visual validation of dart pattern and glow speed.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character` | `NotifyAnimatorSuite` | Dart positions change every 0.5s, positions are within jar bounds, body color is #00C300, glow cycle is 1.5s, glow min/max elevated (0.5/0.9), entry sets immediate brightness, entry triggers immediate dart, Start/Stop lifecycle, no goroutine leak, deterministic RNG produces reproducible positions (for testing) |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Notify Animator | RED | Test Designer | — | — | — |
| Notify Animator | GREEN | Implementer | — | — | — |
| Notify Animator | REFACTOR | Refactorer | — | — | — |
