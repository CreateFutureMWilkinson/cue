# Feature 14: Character Animation System

**Phase:** Phase-3-Feature-14
**Status:** Done
**Packages:** `internal/ui/character/`, `internal/ui/presenter/`, `internal/config/`, `cmd/cue/`

---

## Overview

A character animation abstraction layer for Cue's GUI. The system displays an animated character whose visual state reflects application activity (idle, working, notifying, error, etc.). Characters are configurable via `config.toml` and purely visual — they never affect routing, scoring, or business logic.

## Design Decisions

- **Registry pattern** for character lookup — `Register`/`Create`/`Available` with factory functions. Default `"none"` always registered via `init()`.
- **`NoneCharacterName` constant** (`"none"`) is the single source of truth for the no-op character name.
- **CharacterPresenter** consumes the same `ActivityEvent` stream as the activity log via a fan-out bridge, mapping events to states and handling auto-decay timers.
- **FairyCharacter** uses a colored `canvas.Circle` with state-specific colors — simple, no CGO dependencies.
- **Config default is `"none"`** — opt-in, zero behavioral change from existing state when not configured.
- **Fan-out activity events** — `bridgeEvents` in `main.go` accepts variadic output channels, sending each event to both the activity presenter and character presenter channels.

## API

### CharacterState

```go
type CharacterState int // iota: StateIdle, StateStarting, StateWorking, StateNotifying, StateError, StateShuttingDown
func (s CharacterState) String() string
```

### Character Interface

```go
type Character interface {
    Name() string
    TransitionTo(state CharacterState)
    CurrentState() CharacterState
    Widget() fyne.CanvasObject
}
```

### Registry

```go
func Register(name string, factory CharacterFactory)
func Create(name string) (Character, error)
func Available() []string
func ResetRegistry() // test helper
```

### CharacterPresenter

```go
func NewCharacterPresenter(char character.Character, source ActivitySource, decayDuration time.Duration) (*CharacterPresenter, error)
func (p *CharacterPresenter) Start(ctx context.Context)
func (p *CharacterPresenter) Stop()
```

## Event Mapping

| Event Pattern | Character State |
|---|---|
| `IsError == true` | `StateError` |
| Message contains "NOTIFIED" | `StateNotifying` |
| All other events | `StateWorking` |
| No events for decay duration | `StateIdle` (auto-decay) |

## Error Handling

- Unknown character name in config → falls back to `"none"` with log warning
- Character presenter decay timer resets on each new event
- All state transitions are no-op safe (NoOpCharacter accepts any transition silently)

## Integration Points

- **Config**: `gui.character` field in `config.toml`, defaults to `"none"`
- **Main window**: Character widget placed below activity log (right pane) via `container.NewBorder`
- **Main.go**: Fairy registered, character created from config, CharacterPresenter started/stopped alongside other presenters

## UI Placement

Character widget is in the bottom-right corner of the main window, below the activity log:

```
┌───────────────────────────┬──────────────────────────────────────┐
│                           │         Activity Log                 │
│   Notification Queue      │         (scrollable list)            │
│                           │                                      │
│                           ├──────────────────────────────────────┤
│                           │   [Character Widget]                 │
├───────────────────────────┴──────────────────────────────────────┤
│                    [ Review Buffered ]                            │
└──────────────────────────────────────────────────────────────────┘
```

## Test Coverage Summary

| Package | Suite | Tests |
|---|---|---|
| `character` | `CharacterStateSuite` | String representations, distinct values |
| `character` | `CharacterRegistrySuite` | Register/Create, error on unknown, Available |
| `character` | `NoOpCharacterSuite` | Name, transitions, state tracking, widget |
| `character` | `FairyCharacterSuite` | Name, initial state, transitions, widget, registry |
| `presenter` | `CharacterPresenterSuite` | Working/notifying/error events, decay, start/stop |
| `config` | `ConfigSuite` | Character field parse, default value |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Abstraction | RED | Test Designer | 367s | 39,912 | fb7c866 |
| Abstraction | GREEN | Implementer | 68s | 38,749 | 7bdd483 |
| Abstraction | REFACTOR | Refactorer | 126s | 39,603 | 700827c |
| Fairy | RED | Test Designer | 34s | 21,119 | ea0e3c4 |
| Fairy | GREEN | Implementer | 131s | 36,123 | 4e86ad7 |
| Fairy | REFACTOR | Refactorer | 199s | 33,056 | 67a9de8 |
