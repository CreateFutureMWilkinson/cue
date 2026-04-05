# Feature 026: Fairy Lifecycle States

**Phase:** Phase-3-Feature-026
**Status:** Planned
**Packages:** `internal/ui/character/`

---

## Overview

Implements the fairy's startup and shutdown animations. Starting: the fairy wakes from its dormant state (#004900, diminished glow, on the jar floor), rises and brightens to the idle state (#006100, normal breathing). Shutting down: the fairy drifts down to the jar floor, darkens to #004900 with diminished glow, and the animation cycle completes fully before the application closes. These are one-shot transitions, not steady states.

## Design Decisions

- **Dormant color #004900** — the darkest fairy state, representing powered-off/sleeping. Darker than idle (#006100) but not invisible. The fairy is still visible in the jar.
- **Shutdown does NOT fade out** — the fairy remains visible at #004900 with a dimmed but present glow. This avoids the fairy "dying" — it's sleeping, not gone.
- **Startup is a rising animation** — the fairy is initially at (0.5, 1.0) on the jar floor at #004900 with minimal glow. Over ~1.5 seconds it rises to idle position (still 0.5, 1.0 — floor), brightens to #006100, and glow intensifies to idle breathing range. Then the IdleAnimator takes over.
- **Shutdown is a settling animation** — from whatever current position, the fairy drifts down to (0.5, 1.0), darkens to #004900, and glow diminishes. Duration ~1.5 seconds.
- **Animation must complete before app close** — shutdown animation is non-interruptible. The main application waits for the animation to finish (via a done channel) before proceeding with cleanup. This ensures the fairy always settles gracefully.
- **One-shot, not looping** — unlike idle/working/notifying/error which loop indefinitely, lifecycle animations run once and then either transition to idle (startup) or signal completion (shutdown).
- **Smooth interpolation** — both startup and shutdown use ease-in-out interpolation for position, color, and glow transitions. This produces a gentle, organic feel.

## Animation Specification

### Startup Animation (StateStarting)

```
Duration: 1.5 seconds

t=0.0s:  Position (0.5, 1.0)  Color #004900  Glow intensity 0.1
         └── Dormant: on jar floor, very dim

t=0.75s: Position (0.5, 1.0)  Color #005500  Glow intensity 0.3
         └── Midpoint: brightening, glow growing

t=1.5s:  Position (0.5, 1.0)  Color #006100  Glow intensity 0.5
         └── Idle reached: transitions to IdleAnimator
```

```go
const (
    startupDurationSec = 1.5
)

var (
    dormantColor = color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}  // #004900
    idleColor    = color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}  // #006100
)

// Ease-in-out interpolation (smooth start and end)
func easeInOut(t float64) float64 {
    return t * t * (3.0 - 2.0*t)  // Hermite smoothstep
}

// Color interpolation at progress p (0.0–1.0):
func lerpColor(from, to color.RGBA, p float64) color.RGBA {
    e := easeInOut(p)
    return color.RGBA{
        R: uint8(float64(from.R) + e*float64(int(to.R)-int(from.R))),
        G: uint8(float64(from.G) + e*float64(int(to.G)-int(from.G))),
        B: uint8(float64(from.B) + e*float64(int(to.B)-int(from.B))),
        A: 0xFF,
    }
}
```

### Shutdown Animation (StateShuttingDown)

```
Duration: 1.5 seconds

t=0.0s:  Position (current)    Color (current)  Glow (current)
         └── Whatever state the fairy was in

t=0.75s: Position (lerp → 0.5, 1.0)  Color (lerp → #004900)  Glow (lerp → 0.15)
         └── Midpoint: drifting down, dimming

t=1.5s:  Position (0.5, 1.0)  Color #004900  Glow intensity 0.15
         └── Dormant: settled on jar floor, dim but visible
         └── Signals completion via done channel
```

```go
const (
    shutdownDurationSec  = 1.5
    dormantGlowIntensity = 0.15  // dim but visible
)
```

### Completion Signaling

```go
// Shutdown signals completion via a channel so the app can wait.
type LifecycleAnimator struct {
    // ...
    done chan struct{}  // closed when animation finishes
}

// Done returns a channel that is closed when the shutdown animation completes.
func (a *LifecycleAnimator) Done() <-chan struct{}
```

## API

### StartupAnimator

```go
type StartupAnimator struct {
    fairy     *FairyCharacter
    cancel    context.CancelFunc
    running   bool
    mu        sync.Mutex
    clock     Clock
    onComplete func()  // called when startup finishes; triggers transition to idle
}

func NewStartupAnimator(clock Clock, onComplete func()) *StartupAnimator
func (a *StartupAnimator) Start(fairy *FairyCharacter)
func (a *StartupAnimator) Stop()
func (a *StartupAnimator) State() CharacterState  // returns StateStarting
```

### ShutdownAnimator

```go
type ShutdownAnimator struct {
    fairy      *FairyCharacter
    cancel     context.CancelFunc
    running    bool
    mu         sync.Mutex
    clock      Clock
    done       chan struct{}
    startPos   [2]float64    // captured on Start()
    startColor color.RGBA    // captured on Start()
    startGlow  float64       // captured on Start()
}

func NewShutdownAnimator(clock Clock) *ShutdownAnimator
func (a *ShutdownAnimator) Start(fairy *FairyCharacter)
func (a *ShutdownAnimator) Stop()
func (a *ShutdownAnimator) State() CharacterState  // returns StateShuttingDown
func (a *ShutdownAnimator) Done() <-chan struct{}
```

### FairyCharacter Integration

```go
// Shutdown initiates the shutdown animation and returns a channel
// that closes when the animation completes.
func (f *FairyCharacter) Shutdown() <-chan struct{}
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Startup interrupted by another state | StartupAnimator stopped cleanly, new state begins immediately |
| Shutdown interrupted | Shutdown cannot be interrupted — it always runs to completion |
| App force-quit during shutdown | Context cancelled, goroutine exits, done channel closed |
| Shutdown from dormant state (already at #004900) | Animation still runs (1.5s) but visually is a no-op — consistent timing |

## Integration Points

- **Idle State (Feature 022):** Startup animation completes and triggers `TransitionTo(StateIdle)` via the `onComplete` callback.
- **CharacterPresenter (Feature 014):** May call `TransitionTo(StateStarting)` on app start and `TransitionTo(StateShuttingDown)` on app close. Alternatively, the composition root manages lifecycle directly.
- **Main.go (cmd/cue/):** Shutdown sequence waits on `fairy.Shutdown()` channel before proceeding with graceful cleanup.
- **UAT Harness (Feature 020):** Starting and Shutdown states triggered via buttons for visual validation.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character` | `StartupAnimatorSuite` | Initial state is dormant (#004900, glow 0.1, floor position), final state is idle (#006100, glow ~0.5, floor position), duration is ~1.5s, onComplete callback fires on completion, color interpolation at midpoint, glow interpolation at midpoint, easeInOut produces smooth curve, Stop cancels cleanly |
| `character` | `ShutdownAnimatorSuite` | Captures current state on Start, final state is dormant (#004900, glow 0.15, floor position), duration is ~1.5s, Done channel closes on completion, position interpolates from start to floor, color interpolates from start to #004900, animation from any starting state (idle, working, notifying, error), non-interruptible (Stop waits for completion) |
| `character` | `InterpolationSuite` | easeInOut at 0.0/0.5/1.0, lerpColor correctness, glow lerp correctness |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Startup Animator | RED | Test Designer | — | — | — |
| Startup Animator | GREEN | Implementer | — | — | — |
| Startup Animator | REFACTOR | Refactorer | — | — | — |
| Shutdown Animator | RED | Test Designer | — | — | — |
| Shutdown Animator | GREEN | Implementer | — | — | — |
| Shutdown Animator | REFACTOR | Refactorer | — | — | — |
