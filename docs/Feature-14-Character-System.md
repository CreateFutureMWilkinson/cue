# Feature 14: Character Animation System

**Phase:** Phase-3-Feature-14
**Status:** Not Started
**Packages:** `internal/ui/character/`, `internal/ui/presenter/`, `internal/config/`, `cmd/cue/`

---

## Prompt

Implement a character animation abstraction layer for Cue's GUI. The system displays an animated character in the UI whose visual state reflects what the application is doing (idle, working, notifying, error, etc.). The first character implementation is "fairy" but the system must support swapping characters via config. Characters are purely visual — they observe system state but never affect routing, scoring, or any business logic.

**Read these files before starting:**
- `CLAUDE.md` and `.claude/CLAUDE.md` — project conventions, hard constraints, TDD workflow
- `docs/UI-SPEC.md` — current UI layout and design tokens
- `docs/UI-DESIGN-GUIDE.md` — design-to-implementation pipeline and testing tiers
- `internal/ui/presenter/interfaces.go` — existing presenter interfaces
- `internal/ui/window.go` — current main window composition
- `internal/config/config.go` — current config structure
- `.claude/agents/` — TDD agent role definitions

---

## Requirements

### Character State Machine

Define a `CharacterState` enum representing application states:

| State          | Trigger                                           |
|----------------|---------------------------------------------------|
| `Idle`         | No activity (default)                             |
| `Starting`     | Application startup                               |
| `Working`      | Fetching or routing messages (batch in progress)  |
| `Notifying`    | A NOTIFIED message was just routed                |
| `Error`        | A system error occurred                           |
| `ShuttingDown` | Graceful shutdown initiated                       |

State transitions are driven by the same `ActivityEvent` stream that feeds the activity log. The character presenter observes events and maps them to states. States auto-decay back to `Idle` after a configurable duration (e.g., `Notifying` → `Idle` after 5 seconds) unless a new event arrives.

### Character Interface

```go
// Character represents an animated UI character with state-driven visuals.
type Character interface {
    // Name returns the character's identifier (matches config value).
    Name() string

    // TransitionTo changes the character's visual state.
    TransitionTo(state CharacterState)

    // CurrentState returns the character's current state.
    CurrentState() CharacterState

    // Widget returns a Fyne CanvasObject that can be embedded in the UI.
    Widget() fyne.CanvasObject
}
```

### Character Registry

A registry maps config names to `Character` factory functions:

```go
type CharacterFactory func() Character

func Register(name string, factory CharacterFactory)
func Create(name string) (Character, error)  // returns error if name not registered
func Available() []string                     // list registered names
```

Built-in registrations:
- `"none"` — a no-op character (returns an empty/invisible widget, accepts all state transitions silently)
- `"fairy"` — the fairy character (Phase 3 deliverable)

### Configuration

Add to `config.toml`:

```toml
[gui]
character = "none"         # "none", "fairy", or any registered character
window_width = 1200
window_height = 800
```

Default is `"none"` so the character system is opt-in and doesn't affect existing behavior.

### Character Presenter

A new `CharacterPresenter` that:
- Implements `ActivitySource` consumption (same event channel pattern as `ActivityPresenter`)
- Maps `ActivityEvent` types to `CharacterState` transitions
- Handles state decay timers (e.g., return to `Idle` after timeout)
- Delegates visual changes to the `Character` interface
- Is fully testable at the presenter level (Tier 1) with no Fyne dependency

### Fairy Character (First Implementation)

For Phase 3, the fairy character uses static images or simple Fyne canvas primitives to represent each state. The exact visual design is secondary — what matters is:
- Each state has a visually distinct representation
- Transitions are smooth (not jarring)
- The widget fits in a constrained area of the UI

**Placement in the UI:** Add the character widget to the main window layout. Suggested location: bottom-right corner or as a small panel below the activity log. The exact placement should be defined in `UI-SPEC.md` before implementation begins — update the spec as part of the design phase.

### No-Op Character

The `"none"` character is the default and must:
- Return a minimal/invisible `fyne.CanvasObject` (e.g., empty container)
- Accept all `TransitionTo` calls without error
- Be usable as a drop-in replacement so all code paths work with or without a visible character

---

## Package Structure

```
internal/ui/character/
    state.go          # CharacterState enum, string representations
    character.go      # Character interface + CharacterFactory + registry
    noop.go           # NoOpCharacter implementation
    fairy.go          # FairyCharacter implementation

internal/ui/presenter/
    character_presenter.go       # CharacterPresenter (state machine logic)
    character_presenter_test.go  # Tier 1 tests
```

---

## TDD Agent Team Instructions

### Test Designer (RED Phase)

Write failing tests for:

1. **CharacterState** (`internal/ui/character/`)
   - All states have correct string representations
   - States are distinct values

2. **Registry** (`internal/ui/character/`)
   - Register and create a character by name
   - Create returns error for unregistered name
   - Available() lists all registered names
   - `"none"` is always registered

3. **NoOpCharacter** (`internal/ui/character/`)
   - Name() returns `"none"`
   - TransitionTo() accepts all states without error
   - CurrentState() returns the last transitioned state
   - Widget() returns a non-nil CanvasObject

4. **CharacterPresenter** (`internal/ui/presenter/`)
   - Maps "working" activity events to `Working` state
   - Maps "notifying" activity events to `Notifying` state
   - Maps "error" activity events to `Error` state
   - State decays back to `Idle` after timeout
   - Start/Stop lifecycle works correctly
   - Works with any `Character` implementation (use NoOp in tests)

5. **Config** (`internal/config/`)
   - `gui.character` field parses from TOML
   - Defaults to `"none"` when not specified
   - Accepts any string value (validation happens at character creation time, not config load)

```bash
# Confirm tests fail
go test -count=1 -v -run TestCharacterState ./internal/ui/character/
go test -count=1 -v -run TestCharacterRegistry ./internal/ui/character/
go test -count=1 -v -run TestNoOpCharacter ./internal/ui/character/
go test -count=1 -v -run TestCharacterPresenter ./internal/ui/presenter/
go test -count=1 -v -run TestConfig ./internal/config/
```

Commit: `test(character): failing tests for character animation system`

### Implementer (GREEN Phase)

Read the failing tests. Implement minimal code to pass them:

1. `state.go` — CharacterState type and constants
2. `character.go` — Character interface, registry with Register/Create/Available
3. `noop.go` — NoOpCharacter
4. `character_presenter.go` — CharacterPresenter with event mapping and decay timers
5. Update `config.go` — add `Character` field to `GUIConfig`
6. Do NOT implement the fairy character yet — that's a separate step after the abstraction is proven
7. Do NOT wire into `main.go` yet

```bash
# Confirm all tests pass
just fmt && go test ./...
```

Commit: `feat(character): implement character animation abstraction [tests pass]`

### Refactorer (REFACTOR Phase)

- Clean up any duplication in registry or presenter
- Ensure interface is minimal and consumer-focused
- Verify no circular dependencies introduced
- All tests stay green

```bash
just fmt && just lint && just tidy && just test
```

Commit: `refactor(character): clean up character system`

### Follow-Up: Fairy Implementation (Separate TDD Cycle)

After the abstraction is proven with NoOp + tests:

1. Test Designer: write tests for `FairyCharacter` — each state returns a distinct widget, transitions update visual
2. Implementer: implement `fairy.go` with state-specific Fyne canvas objects
3. Refactorer: clean up
4. Wire into `main.go`: create character from config, pass to main window, connect event stream

### Follow-Up: Docs Commit

After all implementation is complete:
1. Update `docs/UI-SPEC.md` with character widget placement
2. Create `docs/Feature-14-Character-System.md` design doc (overwrite this prompt file)
3. Update `docs/agent-log.md` with TDD phase stats
4. Update `CHANGELOG.md` and `README.md`

---

## Constraints

- No CGO dependencies for the character system (Fyne canvas primitives or image loading are fine)
- Character logic must never import from `decisionengine`, `repository`, or `watcher` — it only observes events
- The `Character` interface must be narrow enough that a new character can be implemented in a single file
- Default config (`"none"`) must not change any existing behavior — zero visual or behavioral difference from current state
- All tests use testify `suite.Suite` in `_test` package suffix
