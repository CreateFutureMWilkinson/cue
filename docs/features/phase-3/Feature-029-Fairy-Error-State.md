# Feature 029: Fairy Error State

**Phase:** Phase-3-Feature-029
**Status:** Done
**Packages:** `internal/ui/character/`

---

## Overview

Implements the fairy's error state behavior for errors requiring user attention: centered in the jar, vibrating with short rapid horizontal oscillation, and a rapid 2 Hz glow pulse. The color is close to the notification state (#00C300). This state is reserved for actionable errors only — most background errors do not trigger a state change, per Cue's error philosophy.

## Design Decisions

- **Centered in jar** — the fairy moves to the horizontal and vertical center of the jar interior (0.5, 0.5). This contrasts with idle (floor) and working/notifying (roaming), making the error state immediately visually distinct.
- **Vibration, not drift** — short, rapid horizontal oscillation around center. The fairy jitters left-right by a small amount (~3-5% of jar width) at high frequency. This creates a "shaking" effect that signals something is wrong.
- **2 Hz maximum pulse rate** — the glow pulses at 2 Hz (0.5-second full cycle), which is the maximum specified. This is significantly faster than idle (0.33 Hz) or notification (0.67 Hz), creating urgency.
- **Near-notification color** — uses a color very close to #00C300 but slightly different to distinguish from notification on close inspection. Uses `#00B800` — still bright green, but subtly darker than notification.
- **Only for actionable errors** — the CharacterPresenter maps `IsError == true` events to `StateError`. Per Cue's error philosophy (Section 12 of CLAUDE.md), only errors requiring user action should surface visually. Background retry errors are logged but don't change fairy state.
- **No entry transition** — error state snaps immediately to center position and begins vibrating. Speed of response matters for error signaling.

## Animation Specification

### Vibration Pattern

```go
const (
    errorVibrateAmplitude = 0.04   // ±4% of jar interior width
    errorVibrateFreqHz    = 15.0   // oscillation frequency (visual jitter speed)
)

// Horizontal vibration around center:
func errorPosition(t float64) (x, y float64) {
    // Centered vertically, vibrating horizontally
    x = 0.5 + errorVibrateAmplitude * math.Sin(2*math.Pi*errorVibrateFreqHz*t)
    y = 0.5
    return
}
```

The 15 Hz oscillation frequency creates a visible vibration/shaking effect. The amplitude is small enough to read as "trembling" rather than "moving."

### Glow Pulse

```go
const (
    errorPulseCycleSec = 0.5    // 2 Hz (fastest pulse in the system)
    errorGlowMin       = 0.4
    errorGlowMax       = 0.9
)

func errorGlowIntensity(t float64) float64 {
    normalized := math.Sin(2 * math.Pi * t / errorPulseCycleSec)
    return errorGlowMin + (errorGlowMax - errorGlowMin) * (normalized + 1.0) / 2.0
}
```

### Color

```go
var errorBodyColor = color.RGBA{R: 0x00, G: 0xB8, B: 0x00, A: 0xFF}  // #00B800
```

### Entry Behavior

On transition to error state:
1. Body color immediately set to #00B800.
2. Position snaps to center (0.5, 0.5).
3. Vibration begins immediately.
4. Glow pulse begins at 2 Hz.

## API

### ErrorAnimator

```go
type ErrorAnimator struct {
    fairy     *FairyCharacter
    cancel    context.CancelFunc
    running   bool
    mu        sync.Mutex
    clock     Clock
}

func NewErrorAnimator(clock Clock) *ErrorAnimator
func (a *ErrorAnimator) Start(fairy *FairyCharacter)
func (a *ErrorAnimator) Stop()
func (a *ErrorAnimator) State() CharacterState  // returns StateError
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Vibration amplitude exceeds bounds | Amplitude is small enough (±4%) to never exceed [0.0, 1.0] from center; SetPosition clamps as safety net |
| Rapid error→error re-entry | Stop + Start cycles cleanly |
| Error state decays to idle | CharacterPresenter's 5-second decay timer handles this — no decay logic in ErrorAnimator itself |

## Integration Points

- **Jar Rendering (Feature 025):** Uses `SetPosition()`, `SetGlowIntensity()`, `SetBodyColor()` for vibration and pulse updates.
- **CharacterPresenter (Feature 014):** Maps `IsError == true` events to `StateError`. Decay timer returns to idle after 5 seconds with no new error events.
- **UAT Harness (Feature 024):** Error state triggered via "Error" button for visual validation of vibration and pulse speed.
- **Notification State (Feature 028):** Color is deliberately close but distinct (#00B800 vs #00C300).

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character` | `ErrorAnimatorSuite` | Position oscillates around center (0.5, 0.5), horizontal amplitude ≤ 0.04, vertical position stays at 0.5, body color is #00B800, glow pulse cycle is 0.5s (2 Hz), glow min/max (0.4/0.9), entry snaps to center immediately, Start/Stop lifecycle, no goroutine leak |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Error Animator | RED | Test Designer | 97s | 38,911 | 6d6536d |
| Error Animator | GREEN | Implementer | 85s | 33,064 | 0d3f4cd |
| Error Animator | REFACTOR | Refactorer | 103s | 39,111 | 5fb31f2 |
