# Feature 076: Integrated Character UAT Mode

**Phase:** Phase-7-Feature-076
**Type:** Refactor
**Severity:** N/A (prerequisite for Feature 075)
**Status:** Planned
**Packages:** `cmd/cue/`, `internal/ui/`, `internal/ui/uat/`, `cmd/cue-uat/` (removed), `cmd/character-uat/` (removed)
**Related:** Feature 075 (WASM Character Plugins), Feature 014 (Character System), Feature 024 (Character UAT Harness)

---

## Problem

The character UAT harness is a standalone binary (`cmd/cue-uat/` + `cmd/character-uat/` package) with its own window, layout, and entry point. This creates several issues:

1. **Separate binary** — goreleaser builds `cue-uat` alongside `cue` for every release, doubling CGO build time per platform.
2. **Divergent UI context** — the UAT window is a custom 800x600 60/40 split layout that looks nothing like the real app. Characters render differently in the UAT harness than in the three-column layout, making the UAT a poor proxy for production behavior.
3. **Blocks Feature 075** — when characters become WASM plugins loaded at runtime, the UAT needs access to the same plugin discovery and host infrastructure as the main binary. Maintaining a separate binary that duplicates this is unsustainable.
4. **Broken FPS counter** — the FPS diagnostics don't work correctly and add dead code.

## Goal

Merge character UAT functionality into the main binary as `cue uat`. In this mode, the full UI starts (three-column layout, all views, all navigation) but with no service backing — no watchers, no database, no Ollama, no planner engine. The **notification panel (right 30%)** is replaced with a **UAT control panel** containing a character chooser dropdown and state trigger buttons. Characters render in the real center column layout, giving accurate visual testing.

## Architecture

### `cue uat` Subcommand

New CLI subcommand alongside existing `cue config`:

```go
func main() {
    app := &cli.Command{
        Name:  "cue",
        Usage: "ADHD-friendly productivity assistant",
        Action: func(ctx context.Context, cmd *cli.Command) error {
            return run()
        },
        Commands: []*cli.Command{
            configCommand(),
            uatCommand(),  // NEW
        },
    }
    // ...
}
```

`uatCommand()` calls `runUAT()` which sets up a minimal app with the full UI but no services.

### What `runUAT()` Does

```
┌─────────────────────────────────────────────────────┐
│              cue uat  startup sequence               │
│                                                     │
│  1. Load config (for GUI dimensions only)           │
│  2. Register all characters (fairy + future plugins)│
│  3. Create Fyne app                                 │
│  4. Build MainWindow in UAT mode:                   │
│     - Focus rail: present, buttons functional        │
│     - Center column: character + all views           │
│     - Right column: UAT panel (NOT notif panel)      │
│  5. Wire view router (all views navigable)          │
│  6. Wire AppBinder with no-op planner               │
│  7. Run window                                       │
└─────────────────────────────────────────────────────┘
```

### What `runUAT()` Skips

| Component | Normal Mode | UAT Mode |
|---|---|---|
| SQLite database | Opened | Skipped |
| Encryption / secret.key | Created | Skipped |
| Ollama client | Connected, models validated | Skipped |
| Vector store (chromem-go) | Opened | Skipped |
| Router / decision engine | Created | Skipped |
| Buffer service | Created | Skipped |
| Alert service | Created | Skipped |
| Orchestrator | Created + started | Skipped |
| Watchers (Slack, Email) | Built from DB, attached | Skipped |
| Service config repository | Created | Skipped |
| Planner repositories | Created | Skipped |
| Calendar provider | Created | Skipped |
| Planner engine | Created | Skipped |
| Timer alert service | Created | Skipped |
| Notification presenter | Created (backed by repo) | Created (no-op, empty cards) |
| Activity presenter | Created (backed by events) | Created (no-op, no events) |
| Feedback presenter | Created (backed by buffer) | Created (no-op) |
| Planner presenter | Created (backed by repos) | Created (no-op stub) |
| Timer presenter | Created (backed by clock) | Created (no-op stub) |
| Settings presenter | Created (backed by alert svc) | Created (no-op stub) |
| Service settings presenter | Created (backed by repo) | Skipped (settings tab still renders, but saves go nowhere) |
| Character registration | Fairy registered | All characters registered |
| Character presenter | Created (backed by events) | Skipped (UAT panel drives state directly) |
| Orchestrator start | Started | Skipped |
| App presenter start | Started | Skipped |

### UAT Control Panel

Replaces the notification panel in the right column (30% width). Built as `internal/ui/uat/UATPanel`:

```
┌─────────────────────────┐
│  Character UAT Controls │
│                         │
│  Character:             │
│  ┌───────────────────┐  │
│  │ fairy          ▼  │  │
│  └───────────────────┘  │
│                         │
│  State Triggers:        │
│  ┌────────┐ ┌────────┐  │
│  │  Idle  │ │Starting│  │
│  └────────┘ └────────┘  │
│  ┌────────┐ ┌────────┐  │
│  │Working │ │Notif'g │  │
│  └────────┘ └────────┘  │
│  ┌────────┐ ┌────────┐  │
│  │ Error  │ │Shutdown│  │
│  └────────┘ └────────┘  │
│                         │
│  ─────────────────────  │
│  Current State: Idle    │
│  Character: fairy       │
│                         │
└─────────────────────────┘
```

#### UATPanel

```go
// package uat (internal/ui/uat/)

type UATPanel struct {
    root            fyne.CanvasObject
    characterSelect *widget.Select
    stateButtons    []*widget.Button
    stateLabel      *widget.Label
    charLabel       *widget.Label

    currentChar     character.Character
    onCharChanged   func(character.Character) // callback to swap character widget in center column
}
```

**Character chooser:**
- `widget.NewSelect` populated from `character.Available()` (sorted, includes `"none"`)
- On selection change: creates new character via `character.Create(name)`, calls `onCharChanged` callback so `MainWindow` can swap the character widget in the center column

**State trigger buttons:**
- 6 buttons in a 2x3 grid: Idle, Starting, Working, Notifying, Error, Shutdown
- Each calls `currentChar.TransitionTo(state)`
- Disabled when no character selected (or `"none"` selected)

**Diagnostics:**
- `stateLabel`: shows current state name after each trigger
- `charLabel`: shows current character name

### MainWindow Changes

`NewMainWindow` needs to accept the notification pane as an injectable component rather than always creating a `NotificationPanel`. Currently (line 139-146):

```go
// Current: hardcoded NotificationPanel
var notifPane fyne.CanvasObject
if np != nil {
    notifPanel = NewNotificationPanel(np, win)
    notifPane = notifPanel.Container()
} else {
    notifPane = widget.NewLabel("")
}
```

**Change:** Accept an optional override for the right column content:

```go
type MainWindowOpts struct {
    // ... existing params stay as positional or move into this struct ...
    RightPanelOverride fyne.CanvasObject // if set, replaces notification panel
}
```

Or more simply, add a `rightPanel fyne.CanvasObject` parameter. When non-nil, it replaces the notification panel entirely. When nil, the existing `NotificationPanel` is created as before. This keeps the change minimal and avoids restructuring the entire constructor.

The character widget also needs to be swappable at runtime (when the UAT dropdown changes character). The center column's character content area should use a `container.NewStack` that the UAT panel can update via a callback.

### No-Op Presenters

UAT mode needs presenters that satisfy the interfaces but do nothing. Rather than creating full presenter infrastructure, use minimal stubs:

**No-op notification presenter:** Returns empty card list, never refreshes.
**No-op activity presenter:** Returns empty log, no event source.
**No-op feedback presenter:** Returns empty buffer, saves are no-ops.
**No-op planner/timer/wizard view models:** Return zero values, button callbacks are no-ops.
**No-op settings presenter:** Volume get/set are no-ops.

These can be either concrete stub types or nil-safe wrappers. The simplest approach: pass `nil` for presenters that `NewMainWindow` already handles (it checks for nil and falls back to placeholder labels). For presenters that require non-nil, create minimal `internal/ui/uat/noop_presenters.go` stubs.

### Character Widget Swapping

When the user picks a different character from the UAT dropdown, the center column needs to display the new character's widget. This requires:

1. The character content area in `MainWindow` uses a `container.NewStack` (it already does — `centerStack`)
2. The activity log drawer wraps the character widget — for UAT mode, skip the activity log drawer and use the character widget directly, OR wrap it and replace the whole thing
3. The UAT panel gets a callback: `onCharChanged func(character.Character)` which:
   - Closes the old character
   - Updates the `ViewCharacter` content in `viewContents` map
   - Refreshes the center stack if currently viewing character view

The cleanest approach: `MainWindow` exposes a `SetCharacterWidget(w fyne.CanvasObject)` method that updates the character view content and refreshes if active.

## Files to Create

| File | Purpose |
|---|---|
| `internal/ui/uat/uat_panel.go` | `UATPanel` — character chooser + state triggers + diagnostics |
| `internal/ui/uat/uat_panel_test.go` | Tests for UAT panel |
| `internal/ui/uat/noop_presenters.go` | No-op presenter stubs for UAT mode |
| `internal/ui/uat/noop_presenters_test.go` | Tests for no-op presenters |
| `internal/ui/uat/doc.go` | Package documentation |
| `cmd/cue/uat.go` | `uatCommand()` + `runUAT()` — CLI subcommand and UAT startup |

## Files to Change

| File | Change |
|---|---|
| `cmd/cue/main.go` | Add `uatCommand()` to CLI commands list |
| `internal/ui/window.go` | Accept optional right panel override; expose `SetCharacterWidget()` method |
| `.goreleaser.yml` | Remove `cue-uat` build target |
| `.github/workflows/ci.yml` | Remove `cue-uat` build step |
| `.github/workflows/release.yml` | Remove `cue-uat` build/artifact steps |
| `.gitea/workflows/ci.yml` | Remove `cue-uat` build step |
| `.gitea/workflows/release.yml` | Remove `cue-uat` build/artifact steps (if present) |
| `justfile` | Change `build-uat` to build main binary, change `run-uat` to `just run -- uat`, remove separate `build-uat` CGO step |

## Files to Remove

| File/Dir | Reason |
|---|---|
| `cmd/cue-uat/main.go` | Entry point replaced by `cue uat` subcommand |
| `cmd/character-uat/` (entire package) | `UATWindow`, `FPSCounter`, `FPSLoop`, `doc.go` — all replaced by `internal/ui/uat/` |

## Test Strategy

### Behaviours

1. **UAT panel renders** — character dropdown + 6 state buttons + state/char labels
2. **Character selection** — choosing from dropdown creates character, updates center widget, updates label
3. **State triggers** — each button calls `TransitionTo` with correct state, updates state label
4. **Buttons disabled for "none"** — selecting `"none"` character disables state buttons
5. **View navigation in UAT** — Plan/Back/Settings buttons work, center view switches
6. **Character visible across views** — switching to Plan and back shows character correctly
7. **No-op presenters** — notification panel shows empty, planner shows no-plan state, settings renders
8. **`cue uat` CLI** — subcommand recognized, calls `runUAT()`
9. **MainWindow right panel override** — passing override replaces notification panel
10. **Character widget swap** — `SetCharacterWidget()` updates center column live

### Existing Tests

All existing `cmd/character-uat/` tests (`uat_window_test.go`, `fps_counter_test.go`, `fps_update_thread_safety_test.go`, `uat_none_test.go`) are **deleted** along with the package. Replacement coverage comes from `internal/ui/uat/uat_panel_test.go`.

FPS-related tests (`fps_counter_test.go`, `fps_update_thread_safety_test.go`) are dropped entirely — FPS counter is removed per user direction.

### TDD Micro-Loops

| # | Behavior | Scope |
|---|---|---|
| 1 | UATPanel renders with dropdown + 6 buttons + labels | `internal/ui/uat/` |
| 2 | Character selection creates character and fires callback | `internal/ui/uat/` |
| 3 | State triggers call TransitionTo, update label | `internal/ui/uat/` |
| 4 | Buttons disabled for "none" character | `internal/ui/uat/` |
| 5 | No-op presenters satisfy interfaces, return zero values | `internal/ui/uat/` |
| 6 | MainWindow accepts right panel override | `internal/ui/` |
| 7 | MainWindow.SetCharacterWidget swaps center content | `internal/ui/` |
| 8 | `cue uat` subcommand wiring | `cmd/cue/` |
| 9 | View navigation functional in UAT mode (integration) | `tests/ui/` |

## CI/CD Changes

### goreleaser (`.goreleaser.yml`)

Remove the `cue-uat` build entry entirely:

```yaml
# REMOVE:
  - id: cue-uat
    main: ./cmd/cue-uat
    binary: cue-uat
    # ...
```

Only the `cue` build remains. The `cue uat` subcommand is part of the same binary.

### CI Workflows

Remove `cue-uat` build steps from all pipelines:

```yaml
# REMOVE from build step:
go build -o _build/cue-uat ./cmd/cue-uat
```

The `cue uat` subcommand is tested as part of the normal test suite. No separate binary build needed.

### Justfile

```just
# BEFORE:
build-uat:
    CGO_ENABLED=1 go build -o _build/character-uat ./cmd/cue-uat

run-uat: build-uat
    ./_build/character-uat

# AFTER:
run-uat: build
    ./_build/cue uat
```

## Acceptance Criteria

- [ ] `cue uat` launches the full UI in UAT mode
- [ ] Three-column layout intact: focus rail (10%), center (60%), UAT panel (30%)
- [ ] UAT panel shows character dropdown populated from registry (sorted, includes "none")
- [ ] UAT panel shows 6 state trigger buttons in 2x3 grid
- [ ] Selecting a character swaps the center column widget
- [ ] State buttons trigger character animations
- [ ] State buttons disabled when "none" is selected
- [ ] State and character labels update on interaction
- [ ] Focus rail navigation works: Plan, Back, Settings all switch views
- [ ] Settings view renders (with no-op backing)
- [ ] Plan view renders (shows no-plan state)
- [ ] Wizard view renders (shows empty wizard)
- [ ] No database, Ollama, or network connections made in UAT mode
- [ ] `cmd/cue-uat/` and `cmd/character-uat/` removed
- [ ] `cue-uat` build target removed from goreleaser and CI
- [ ] All existing non-UAT tests pass
- [ ] `just run-uat` runs `cue uat` via the main binary
