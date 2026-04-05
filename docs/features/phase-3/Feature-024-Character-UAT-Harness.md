# Feature 024: Character UAT Harness

**Phase:** Phase-3-Feature-024
**Status:** Done
**Packages:** `cmd/character-uat/`, `internal/ui/character/`

---

## Overview

Standalone entry point (`cmd/character-uat/main.go`) that launches a Fyne window for visually testing and validating character animations. Provides buttons to trigger each character state, a dropdown to select registered characters, and diagnostic overlays (state name, character name, frame rate). Designed for reuse — any character registered via the character registry can be tested, not just the fairy.

## Design Decisions

- **Separate binary, not a CLI subcommand** — avoids pulling UAT dependencies into the main Cue binary. Built as `_build/character-uat` via a new just recipe.
- **Uses the existing character registry** — discovers all registered characters via `character.Available()` and creates them via `character.Create()`. No hardcoded character knowledge.
- **State buttons match the CharacterState enum** — one button per state (Idle, Starting, Working, Notifying, Error, ShuttingDown). Clicking a button calls `TransitionTo()` directly, bypassing the CharacterPresenter event mapping.
- **Frame rate display** — measures actual render FPS by counting `Refresh()` calls per second. Displayed as a live-updating label.
- **No dependency on orchestrator, repository, or config** — the UAT harness is self-contained. It only imports the character package and Fyne.
- **Character dropdown triggers widget swap** — selecting a different character from the dropdown replaces the displayed widget in the container. The current state is preserved across swaps where possible.

## UI Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  Character UAT                                                    │
├──────────────────────────────┬───────────────────────────────────┤
│                              │  Diagnostics                      │
│                              │                                   │
│                              │  Character: Cue                   │
│                              │  State: Idle                      │
│   ┌──────────────────────┐   │  FPS: 60                          │
│   │                      │   │                                   │
│   │                      │   ├───────────────────────────────────┤
│   │   Character Widget   │   │  Controls                         │
│   │                      │   │                                   │
│   │                      │   │  Character: [Cue (fairy)     ▾]   │
│   │                      │   │                                   │
│   └──────────────────────┘   │  [ Idle    ] [ Starting  ]        │
│                              │  [ Working ] [ Notifying ]        │
│                              │  [ Error   ] [ Shutdown  ]        │
│                              │                                   │
├──────────────────────────────┴───────────────────────────────────┤
│  UAT Harness — select character and trigger states               │
└──────────────────────────────────────────────────────────────────┘

  Window default: 800w × 600h
  Split: 60/40 horizontal (character display / controls+diagnostics)
```

## API

### Entry Point

```go
// cmd/character-uat/main.go
func main()
```

### FPS Counter

```go
type FPSCounter struct {
    mu        sync.Mutex
    frames    int
    lastCheck time.Time
    current   float64
}

func NewFPSCounter() *FPSCounter
func (f *FPSCounter) Tick()           // called on each render frame
func (f *FPSCounter) FPS() float64    // returns current FPS
```

### UAT Window

```go
type UATWindow struct {
    window       fyne.Window
    charWidget   fyne.CanvasObject
    character    character.Character
    fps          *FPSCounter
    stateLabel   *widget.Label
    charLabel    *widget.Label
    fpsLabel     *widget.Label
}

func NewUATWindow(app fyne.App) *UATWindow
func (w *UATWindow) Run()
func (w *UATWindow) selectCharacter(name string)
func (w *UATWindow) triggerState(state character.CharacterState)
```

## Character Registration

The UAT harness must register all characters before use. It imports and registers the same characters as the main Cue binary:

```go
func init() {
    character.Register("fairy", func() character.Character {
        return character.NewFairyCharacter()
    })
    // Future characters registered here
}
```

## Build

```bash
just build-uat    # compile to _build/character-uat
just run-uat      # build and run
```

### Just Recipes

```makefile
uat:
    go build -o _build/character-uat ./cmd/character-uat

run-uat: uat
    ./_build/character-uat
```

## Error Handling

| Scenario | Behavior |
|---|---|
| No characters registered | Show "No characters available" in dropdown, disable state buttons |
| Character creation fails | Show error in diagnostics panel, keep previous character |
| Unknown state triggered | No-op (CharacterState is a typed int, buttons cover all values) |

## Integration Points

- **Character Registry (Feature 014):** Uses `character.Register()`, `character.Create()`, `character.Available()` to discover and instantiate characters.
- **Character Interface (Feature 014):** Calls `TransitionTo()`, `CurrentState()`, `Name()`, `Widget()` on selected character.
- **Future Characters:** Any character registered with the registry is automatically available in the dropdown — no UAT code changes needed.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character-uat` | `FPSCounterSuite` | Tick counting, FPS calculation, zero-time handling, concurrent access |

Note: The UAT window itself is a manual testing tool — its primary value is enabling visual validation of character features. The FPS counter is the only unit-testable component.

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| FPS Counter | RED | Test Designer | 57s | 19,239 | fa6bd5f |
| FPS Counter | GREEN | Implementer | 45s | 19,863 | dae8bbe |
| FPS Counter | REFACTOR | Refactorer | — | — | — |
| UAT Window | RED | Test Designer | 53s | 24,445 | b758676 |
| UAT Window | GREEN | Implementer | 104s | 37,938 | dfec3c7 |
| UAT Window | REFACTOR | Refactorer | — | — | 7df8f41 |
