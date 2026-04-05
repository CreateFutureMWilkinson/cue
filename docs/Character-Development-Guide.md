# Character Development Guide

This guide covers how to implement a new character for the Cue UI. A character is an animated visual element that reacts to application state changes (idle, working, notifying, error, etc.).

---

## Architecture Overview

```
internal/ui/character/              ← shared interface, registry, states
  character.go                      ← Character interface + registry
  state.go                          ← CharacterState enum
  animator.go                       ← Clock, Ticker interfaces
  noop.go                           ← NoOpCharacter (default)
  fairy/                            ← one sub-package per character
    fairy.go                        ← FairyCharacter implementation
    layout.go                       ← custom Fyne layout
    animators.go                    ← per-state animation goroutines
    colors.go                       ← colour constants, helpers
    assets/                         ← embedded images
      jar_back.png
      jar_front.png
    embed.go                        ← go:embed declarations
    fairy_test.go
```

Each character lives in its own sub-package of `internal/ui/character/`. The sub-package imports the parent for the `Character` interface, `CharacterState` enum, and `Clock`/`Ticker` interfaces. Everything else — animators, layout, assets, helpers — is private to the sub-package.

The parent package owns the registry. A `"none"` no-op character is always registered as the default fallback.

## Step 1: Create Your Sub-Package

Create a directory under `internal/ui/character/` named after your character:

```
internal/ui/character/mychar/
  mychar.go
  animators.go
  assets/
    sprite.png
  embed.go
  mychar_test.go
```

Your package declaration:

```go
package mychar
```

Import the parent for shared types:

```go
import "github.com/CreateFutureMWilkinson/cue/internal/ui/character"
```

## Step 2: Embed Assets

All image assets must be embedded into the binary using `go:embed`. Do not use `canvas.NewImageFromFile` — it relies on relative paths at runtime and silently fails if the working directory is wrong.

### Image format

Use **PNG** for all character artwork. Fyne's SVG rendering is inconsistent across platforms and driver backends. PNG renders reliably everywhere.

### Embedding pattern

Create an `embed.go` in your sub-package:

```go
package mychar

import "embed"

//go:embed assets/sprite.png
var spritePNG []byte
```

Use `fyne.NewStaticResource` and `canvas.NewImageFromResource` in your constructor:

```go
import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
)

sprite := canvas.NewImageFromResource(
    fyne.NewStaticResource("sprite.png", spritePNG),
)
sprite.FillMode = canvas.ImageFillContain
```

This compiles the image data into the binary — no filesystem dependency at runtime.

### Asset directory structure

Keep assets co-located with the character code:

```
internal/ui/character/mychar/
  assets/
    background.png
    overlay.png
  embed.go              ← all //go:embed declarations in one file
```

`go:embed` paths are relative to the source file, so `assets/background.png` works from any file in the `mychar/` directory.

## Step 3: Implement the Character Interface

Your character must satisfy:

```go
// internal/ui/character/character.go
type Character interface {
    Name() string
    TransitionTo(state CharacterState)
    CurrentState() CharacterState
    Widget() fyne.CanvasObject
    Close()
}
```

### States

Every character responds to the same set of states:

| State | Meaning |
|---|---|
| `StateIdle` | Default resting state |
| `StateStarting` | App is starting up |
| `StateWorking` | Processing a batch |
| `StateNotifying` | High-importance message arrived |
| `StateError` | Something went wrong |
| `StateShuttingDown` | App is closing |

You don't need a unique animation for every state — you can reuse animators or return `nil` to do nothing for a given state.

### Minimal skeleton

```go
package mychar

import (
    "image/color"
    "sync"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/layout"

    "github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

type MyCharacter struct {
    state           character.CharacterState
    container       *fyne.Container
    currentAnimator StateAnimator  // your own animator interface, local to this package
    mu              sync.Mutex
    refreshFunc     func()

    body *canvas.Circle
}

func NewMyCharacter() *MyCharacter {
    body := canvas.NewCircle(color.RGBA{R: 0x33, G: 0x99, B: 0xFF, A: 0xFF})

    c := &MyCharacter{
        state:       character.StateIdle,
        body:        body,
        refreshFunc: func() {}, // Temporary no-op; wired below.
    }

    c.container = container.New(layout.NewCenterLayout(), body)

    // Wire refreshFunc AFTER the container exists.
    c.refreshFunc = func() { fyne.Do(func() { c.container.Refresh() }) }

    return c
}

func (c *MyCharacter) Name() string                              { return "mychar" }
func (c *MyCharacter) Widget() fyne.CanvasObject                 { return c.container }
func (c *MyCharacter) CurrentState() character.CharacterState {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.state
}
```

## Step 4: Thread Safety

**This is the most important section.** Fyne requires all widget mutations to happen on the UI thread. Your animators run in background goroutines. If you call `.Refresh()` on a canvas object from a goroutine, the app will freeze on Wayland and behave unpredictably on X11.

### The rule

> **Never call `.Refresh()` on any Fyne canvas object directly.** All visual updates must go through your `refreshFunc`, which dispatches via `fyne.Do()`.

### The refreshFunc pattern

Every character must have a `refreshFunc` field:

```go
refreshFunc func()
```

Wire it after constructing your container:

```go
c.refreshFunc = func() { fyne.Do(func() { c.container.Refresh() }) }
```

`fyne.Do` is non-blocking — it queues work to the UI thread and returns immediately. This means it is safe to call while holding a mutex.

All setter methods that change visual state call `refreshFunc()` instead of refreshing individual objects:

```go
func (c *MyCharacter) SetBodyColor(col color.Color) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.body.FillColor = col
    c.refreshFunc()           // Container refresh propagates to all children.
}
```

**Do not** do this:

```go
func (c *MyCharacter) SetBodyColor(col color.Color) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.body.FillColor = col
    c.body.Refresh()          // WRONG: called from animator goroutine → crash on Wayland
}
```

Container-level refresh propagates to all children, so individual `.Refresh()` calls are unnecessary.

### Testing

In tests you don't have a running Fyne app, so `fyne.Do()` would panic. Replace `refreshFunc` with a no-op or a counter:

```go
func (c *MyCharacter) DisableRefresh() { c.refreshFunc = func() {} }

// Or inject a test spy:
func (c *MyCharacter) SetRefreshFunc(fn func()) { c.refreshFunc = fn }
```

```go
func (s *MySuite) TestSetBodyColorCallsRefresh() {
    c := NewMyCharacter()
    var count int
    c.SetRefreshFunc(func() { count++ })
    c.SetBodyColor(color.White)
    s.Equal(1, count)
}
```

### Checklist

Before marking your character implementation complete, verify:

- [ ] `refreshFunc` is wired to `fyne.Do(func() { container.Refresh() })` after container construction
- [ ] No method calls `.Refresh()` on any individual canvas object (`Circle`, `Image`, `Line`, etc.)
- [ ] All setter methods that change visual properties call `refreshFunc()` exactly once
- [ ] `TransitionTo` / state update methods call `refreshFunc()` for indicator or state-driven visual changes
- [ ] All tests use `DisableRefresh()` or `SetRefreshFunc()`
- [ ] `go test -race ./internal/ui/character/...` passes

## Step 5: Implement Animators

Each character defines its own animator interface locally. Animators are internal to the sub-package — the parent `character` package does not need to know about them.

```go
// internal/ui/character/mychar/animators.go
package mychar

type stateAnimator interface {
    Start(char *MyCharacter)
    Stop()
}
```

### Animator structure

Animators follow a consistent pattern:

```go
type idleAnimator struct {
    clock  character.Clock
    mu     sync.Mutex
    cancel context.CancelFunc
    done   chan struct{}
}

func (a *idleAnimator) Start(char *MyCharacter) {
    a.Stop() // Idempotent: stop any previous run.

    a.mu.Lock()
    defer a.mu.Unlock()

    // Set initial visual state.
    char.SetBodyColor(idleColor)

    startTime := a.clock.Now()
    ctx, cancel := context.WithCancel(context.Background())
    a.cancel = cancel
    a.done = make(chan struct{})

    ticker := a.clock.NewTicker(time.Duration(character.AnimationTickMs) * time.Millisecond)

    go func() {
        defer close(a.done)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.Chan():
                elapsed := a.clock.Now().Sub(startTime).Seconds()
                char.SetGlowIntensity(breathe(elapsed))
            }
        }
    }()
}

func (a *idleAnimator) Stop() {
    a.mu.Lock()
    cancel := a.cancel
    done := a.done
    a.cancel = nil
    a.done = nil
    a.mu.Unlock()

    if cancel != nil {
        cancel()
    }
    if done != nil {
        <-done // Wait for goroutine to exit.
    }
}
```

Key points:
- **`Start` is called on the UI thread** (from `TransitionTo`), but the goroutine it spawns is not.
- **`Stop` must block** until the goroutine exits. This prevents races during state transitions.
- **Use the `character.Clock` interface** so tests can control time without `time.Sleep`.
- **Call character setter methods** (`SetBodyColor`, etc.) — never touch canvas objects directly from the animator.

### Deadlock prevention in TransitionTo

When transitioning states, the previous animator must be stopped before starting the new one. The stop must happen **outside** the character's mutex, because the animator's goroutine may be mid-call to a setter method that also acquires the mutex:

```go
func (c *MyCharacter) TransitionTo(state character.CharacterState) {
    // Phase 1: Extract and clear animator under lock.
    c.mu.Lock()
    prev := c.currentAnimator
    c.currentAnimator = nil
    c.mu.Unlock()

    // Phase 2: Stop outside lock (animator goroutine may hold mu via Set* calls).
    if prev != nil {
        prev.Stop()
    }

    // Phase 3: Update state and start new animator under lock.
    c.mu.Lock()
    c.state = state
    animator := c.createAnimatorForState(state)
    c.currentAnimator = animator
    c.mu.Unlock()

    // Phase 4: Trigger visual refresh for state change.
    c.refreshFunc()

    if animator != nil {
        animator.Start(c)
    }
}
```

## Step 6: Register Your Character

Register your character factory so the app and UAT harness can discover it. The idiomatic place is an `init()` function in the entry point that imports your package:

```go
// cmd/cue/main.go (or cmd/cue-uat/main.go)
import (
    "github.com/CreateFutureMWilkinson/cue/internal/ui/character"
    "github.com/CreateFutureMWilkinson/cue/internal/ui/character/mychar"
)

func init() {
    character.Register("mychar", func() character.Character {
        return mychar.NewMyCharacter()
    })
}
```

Users select it via `config.toml`:

```toml
[gui]
character = "mychar"
```

The character UAT harness automatically discovers all registered characters and presents them in a dropdown. The `"none"` no-op character is always available as a baseline.

## Step 7: Testing

All tests use testify suites in the `_test` package:

```go
package mychar_test

import (
    "testing"

    "github.com/stretchr/testify/suite"
)

type MyCharacterSuite struct {
    suite.Suite
}

func TestMyCharacter(t *testing.T) {
    suite.Run(t, new(MyCharacterSuite))
}
```

### Mock clock

Use a mock clock to control animation timing without real delays:

```go
clock := newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
char.SetClock(clock)

// Advance time and let goroutine process.
clock.Advance(1 * time.Second)
time.Sleep(5 * time.Millisecond)
```

### What to test

- State transitions produce correct `CurrentState()`
- Each setter calls `refreshFunc` exactly once (use a counting spy)
- Animator `Stop()` is idempotent (call it twice, no panic)
- Position/colour/glow values are correct after animator initialisation
- `Close()` stops the animator cleanly
- Embedded assets are non-empty (`len(spritePNG) > 0`)

### UAT verification

Run the character UAT harness to visually verify your character:

```bash
just run-uat
```

Select your character from the dropdown and cycle through all six states. Verify animations are smooth and the app does not produce any thread-safety errors. On Wayland:

```bash
FYNE_DRIVER=wayland just run-uat 2>&1 | tee /tmp/uat.log
grep "Error in Fyne call thread" /tmp/uat.log
```

Expect zero matches.

## Reference: Existing Implementation

The fairy character (`internal/ui/character/fairy/`) is the reference implementation. It demonstrates:

- Embedded PNG jar artwork via `go:embed`
- Custom `fairyJarLayout` for proportional circle sizing within jar bounds
- 8-layer glow system with graduated alpha
- 6 animators (idle, startup, working, notify, error, shutdown)
- Normalised 0–1 coordinate system with clamping
- `lerpColor` for smooth colour transitions
- `glowIntensity` sinusoidal breathing function
- `refreshFunc` pattern for Wayland-safe rendering

## Common Pitfalls

| Pitfall | Fix |
|---|---|
| Calling `.Refresh()` on a canvas object from a goroutine | Use `refreshFunc()` — it dispatches via `fyne.Do()` |
| Forgetting to wire `refreshFunc` after container construction | Set it immediately after `container.New(...)` |
| Using `canvas.NewImageFromFile` for artwork | Use `go:embed` + `canvas.NewImageFromResource` — file paths break at runtime |
| Using SVG for character artwork | Use PNG — Fyne's SVG rendering is inconsistent across platforms |
| Stopping animator while holding the character mutex | Extract animator under lock, stop it outside the lock |
| Using `time.Sleep` in animator loops | Use `character.Clock.NewTicker` for testability |
| Not waiting for goroutine exit in `Stop()` | Block on `<-done` channel after cancelling context |
| Running tests without `DisableRefresh()` | Tests without a Fyne app will panic on `fyne.Do()` |
| Putting assets in a shared directory | Keep assets in your sub-package's `assets/` directory |
