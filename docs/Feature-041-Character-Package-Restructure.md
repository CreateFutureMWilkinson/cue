# Feature 041: Character Package Restructure

**Phase:** Phase-3-Feature-041
**Status:** Planned
**Packages:** `internal/ui/character/`, `internal/ui/character/fairy/`, `cmd/cue/`, `cmd/cue-uat/`

---

## Overview

Restructure the character system so each character lives in its own sub-package of `internal/ui/character/` with co-located assets. The fairy moves from the parent package into `internal/ui/character/fairy/`. Jar artwork switches from runtime-loaded SVGs to `go:embed` PNGs. The `"none"` no-op character becomes visible in the UAT harness dropdown.

## Motivation

The current layout puts all fairy-specific code (6 animators, jar layout, colour constants, glow helpers) directly in the `internal/ui/character/` package alongside the shared `Character` interface and registry. As more characters are added, this becomes unsustainable. Each character should own its own package, assets, animators, and tests — the parent package provides only the shared contract.

Secondary issues resolved:
- Jar SVGs loaded via `canvas.NewImageFromFile("build_assets/images/...")` — silently fails if the working directory is wrong, and Fyne's SVG rendering is inconsistent across platforms. Switching to `go:embed` PNGs eliminates both problems.
- The `"none"` character is filtered out of the UAT dropdown, but it should be available as a baseline for testing.

## Target Structure

```
internal/ui/character/
  character.go              ← Character interface, CharacterFactory, registry
  state.go                  ← CharacterState enum, String()
  animator.go               ← Clock, Ticker interfaces, WallClock, wallTicker
  noop.go                   ← NoOpCharacter (registered as "none")
  noop_test.go
  state_test.go

  fairy/
    fairy.go                ← FairyCharacter struct, constructor, Widget, TransitionTo, Close, Shutdown
    layout.go               ← fairyJarLayout (custom Fyne layout)
    colors.go               ← colour constants, stateColor, lerpColor, glowIntensity, clamp01
    idle_animator.go
    working_animator.go
    startup_animator.go
    notify_animator.go
    error_animator.go
    shutdown_animator.go
    assets/
      jar_back.png
      jar_front.png
    embed.go                ← //go:embed declarations
    fairy_test.go
    fairy_jar_test.go
    idle_animator_test.go
    working_animator_test.go
    startup_animator_test.go
    notify_animator_test.go
    error_animator_test.go
    shutdown_animator_test.go
```

## What Stays in the Parent Package

| File | Contents |
|---|---|
| `character.go` | `Character` interface, `CharacterFactory` type, `Register`, `Create`, `Available`, `ResetRegistry` |
| `state.go` | `CharacterState` enum (Idle, Starting, Working, Notifying, Error, ShuttingDown), `String()` |
| `animator.go` | `Clock` interface, `Ticker` interface, `WallClock` struct, `wallTicker` struct, `AnimationFPS`, `AnimationTickMs`, `AnimationFrameInterval` constants |
| `noop.go` | `NoOpCharacter` — satisfies `Character` with no visuals, registered as `"none"` |

## What Moves to `fairy/`

Everything else currently in `internal/ui/character/`:

| Current file | Destination | Notes |
|---|---|---|
| `fairy.go` | `fairy/fairy.go` + `fairy/layout.go` + `fairy/colors.go` | Split into logical files |
| `idle_animator.go` | `fairy/idle_animator.go` | Package becomes `fairy` |
| `working_animator.go` | `fairy/working_animator.go` | |
| `startup_animator.go` | `fairy/startup_animator.go` | |
| `notify_animator.go` | `fairy/notify_animator.go` | |
| `error_animator.go` | `fairy/error_animator.go` | |
| `shutdown_animator.go` | `fairy/shutdown_animator.go` | |
| All `*_test.go` for above | `fairy/*_test.go` | Package becomes `fairy_test` |
| `build_assets/images/jar_back.png` | `fairy/assets/jar_back.png` | Copied, not moved (build_assets may serve other purposes) |
| `build_assets/images/jar_front.png` | `fairy/assets/jar_front.png` | |

## Key Changes

### 1. Package and Import Updates

The fairy package imports the parent for shared types:

```go
package fairy

import "github.com/CreateFutureMWilkinson/cue/internal/ui/character"
```

All references to `CharacterState`, `Clock`, `Ticker`, `AnimationTickMs`, etc. become `character.CharacterState`, `character.Clock`, etc.

### 2. StateAnimator Becomes Fairy-Local

The `StateAnimator` interface currently lives in `animator.go` (parent package) and takes `*FairyCharacter`. Since each character owns its animators, this interface moves into the fairy package as an unexported type:

```go
// fairy/fairy.go
type stateAnimator interface {
    Start(fairy *FairyCharacter)
    Stop()
    State() character.CharacterState
}
```

The parent `animator.go` retains only `Clock`, `Ticker`, `WallClock`, `wallTicker`, and the animation timing constants.

### 3. Exported Constants and Vars

Several fairy-specific constants are currently exported from the parent package and referenced by tests or other code:

| Symbol | Current location | Action |
|---|---|---|
| `IdleOriginX`, `IdleOriginY` | `fairy.go` | Move to `fairy/colors.go`, keep exported |
| `IdleBodyColor` | `fairy.go` | Move to `fairy/colors.go`, keep exported |
| `IdleBreathCycleSec`, `IdleGlowMin`, `IdleGlowMax` | `idle_animator.go` | Move to `fairy/idle_animator.go`, keep exported |
| `IdleGlowIntensity()` | `idle_animator.go` | Move to `fairy/idle_animator.go`, keep exported |
| All `*BodyColor`, `*GlowIntensity()`, `*Position()` funcs | Various animators | Move to `fairy/`, keep exported for test access |
| `AnimationFPS`, `AnimationTickMs`, `AnimationFrameInterval` | `idle_animator.go` | Stay in parent `animator.go` (shared across characters) |

### 4. Asset Embedding

New file `fairy/embed.go`:

```go
package fairy

import "embed"

//go:embed assets/jar_back.png
var jarBackPNG []byte

//go:embed assets/jar_front.png
var jarFrontPNG []byte
```

Constructor changes in `fairy/fairy.go`:

```go
// Before:
jarBack := canvas.NewImageFromFile("build_assets/images/jar_back.svg")

// After:
jarBack := canvas.NewImageFromResource(fyne.NewStaticResource("jar_back.png", jarBackPNG))
```

### 5. UAT Harness: Show "none" Character

In `cmd/character-uat/uat_window.go`, remove the filter that excludes `NoneCharacterName`:

```go
// Before (availableCharacterNames):
for _, name := range all {
    if name != character.NoneCharacterName {
        names = append(names, name)
    }
}

// After:
names = all
```

Sort is already applied. The `"none"` character appears in the dropdown alongside `"fairy"`.

### 6. Registration Updates

`cmd/cue-uat/main.go`:

```go
import (
    "github.com/CreateFutureMWilkinson/cue/internal/ui/character"
    "github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
)

func init() {
    character.Register("fairy", func() character.Character {
        return fairy.NewFairyCharacter()
    })
}
```

Same pattern in `cmd/cue/main.go`.

### 7. Presenter Updates

`internal/ui/presenter/character_presenter.go` imports `character` for the `Character` interface and `CharacterState` — no change needed since it doesn't reference fairy-specific types.

## What This Does NOT Change

- The `Character` interface — identical API
- The `CharacterState` enum — stays in parent
- The `Clock`/`Ticker` interfaces — stay in parent
- The registry (`Register`, `Create`, `Available`) — stays in parent
- `CharacterPresenter` — unchanged, talks to `Character` interface
- Animator behaviour — all 6 animators work identically, just in a new package
- Test coverage — all existing tests move with their code

## Design Decisions

- **Sub-package per character, not a flat package with prefixes** — Keeps assets co-located with code, allows `go:embed`, prevents naming collisions, and makes it obvious which code belongs to which character.
- **Animator interface is character-local** — Each character's animators know the concrete type they're animating. Forcing them through a generic interface would require type assertions or a bloated shared API. The parent package doesn't need to know about animators.
- **PNG over SVG** — Fyne's SVG rendering is inconsistent across platforms and driver backends. PNGs render reliably everywhere. The SVG source files remain in `build_assets/` for future re-export if needed.
- **`go:embed` over runtime file loading** — Eliminates working-directory dependency and silent failures. The binary is self-contained.
- **`"none"` visible in UAT** — Provides a zero-animation baseline for testing. Useful for verifying that the harness itself works without character-specific side effects.

## Test Coverage

All existing fairy tests move to `fairy/` with package declaration changed to `package fairy_test`. Import paths update from `character` to `fairy`. No test logic changes — this is a pure move.

Verify with:

```bash
just fmt && just lint && just tidy && just test
go test -race -count=1 ./internal/ui/character/...
go test -race -count=1 ./internal/ui/character/fairy/...
```

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | — | — | — |
| GREEN | Implementer | — | — | — |
| REFACTOR | Refactorer | — | — | — |
