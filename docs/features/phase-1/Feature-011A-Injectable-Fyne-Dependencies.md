# Feature 011A — Injectable Fyne Dependencies

**Type:** Hotfix (Feature 011 — Fyne GUI)
**Status:** Done
**Priority:** High — 50+ Fyne error log lines in CI on every test run

---

## Problem

`NewFairyCharacter()` sets `refreshFunc` to a closure that calls `fyne.CurrentApp()` at invocation time. Even though the result is nil-guarded, `fyne.CurrentApp()` itself logs an error (`"Attempt to access current Fyne app when none is started"`) as a side effect. This produces ~50 noisy error lines in CI test output.

Similarly, `internal/ui/window.go` uses a package-level `var newFyneApp` overridden via `export_test.go` `init()` — a testing workaround that should be replaced with proper dependency injection.

Both issues stem from the same root cause: Fyne app dependencies are implicitly coupled rather than explicitly injected.

## Design

### Principle

Make Fyne runtime dependencies (app creation, thread-safe refresh) **injectable** — passed in by callers rather than assumed from global state.

### Part 1: Fairy Package — Injectable `refreshFunc`

**Current state (`fairy.go:126–131`):**
```go
f.container = container.New(&fairyJarLayout{fairy: f}, objects...)
f.refreshFunc = func() {
    if fyne.CurrentApp() != nil {
        fyne.Do(func() { f.container.Refresh() })
    }
}
```

**Change:** Default `refreshFunc` to a **no-op** in the constructor. Production callers wire the real `fyne.Do` refresh via the existing `SetRefreshHook()` method after construction.

```go
f.refreshFunc = func() {}
```

**Production wiring (`cmd/cue/main.go:290–292`):**
```go
character.Register("fairy", func() character.Character {
    f := fairy.NewFairyCharacter()
    f.SetRefreshHook(func() {
        fyne.Do(func() { f.Widget().Refresh() })
    })
    return f
})
```

Same pattern for `cmd/cue-uat/main.go` and `cmd/character-uat/uat_window_test.go`.

**Why no-op default?** The `SetClock()` pattern already established in this codebase (Feature 025A) shows that post-construction wiring is acceptable when the constructor has many call sites (~79 for `NewFairyCharacter`). A no-op default is safe because:
- Tests already get the behavior they want (most call `DisableRefresh()` or `SetRefreshHook()`)
- Production always has a running Fyne app and wires the real refresh immediately after construction
- The hook is set before the fairy is returned or used, so no animation frames are missed

**`DisableRefresh()` deprecation:** Mark as deprecated in godoc. It becomes a no-op on a no-op, but remains for backward compatibility. `SetRefreshHook()` remains the primary API.

### Part 2: UI Package — Injectable `fyne.App`

**Current state (`window.go:15–17`):**
```go
var newFyneApp = func() fyne.App { return app.New() }
```
Overridden in `export_test.go`:
```go
func init() {
    newFyneApp = func() fyne.App { return test.NewApp() }
}
```

**Change:** Add `fyne.App` as the first parameter of `NewMainWindow`. Remove the `newFyneApp` package-level variable and `export_test.go`.

```go
func NewMainWindow(
    fyneApp fyne.App,  // NEW — injected
    cfg config.GUIConfig,
    np *presenter.NotificationPresenter,
    // ... rest unchanged
) *MainWindow {
```

**`fyne.CurrentApp()` in About menu (`window.go:124`):** Replace with the stored `fyneApp` field already on `MainWindow`.

**Production caller (`cmd/cue/main.go:328`):**
```go
fyneApp := app.New()
mainWindow := ui.NewMainWindow(fyneApp, cfg.GUI, ...)
```

**Test caller (`window_layout_test.go:30`):**
```go
return ui.NewMainWindow(
    test.NewApp(),
    cfg,
    // ...
)
```

**Delete:** `internal/ui/export_test.go` — no longer needed.

## Behavior Decomposition

| # | Behavior | Package | Scope |
|---|---|---|---|
| 1 | Constructor defaults `refreshFunc` to no-op | `fairy` | Change 1 line in `fairy.go` |
| 2 | Production fairy wiring calls `SetRefreshHook` with `fyne.Do` | `cmd/cue`, `cmd/cue-uat`, `cmd/character-uat` | 3 call sites |
| 3 | `NewMainWindow` accepts `fyne.App` parameter | `ui` | Signature change + 2 production callers + 1 test helper |
| 4 | About menu uses stored `fyneApp` instead of `fyne.CurrentApp()` | `ui` | 1 line in `window.go` |
| 5 | Delete `export_test.go` | `ui` | Remove file |

### Sequencing

1. **Behavior 1** first — highest-value change, eliminates ~50 CI errors, zero test impact
2. **Behavior 2** — wire production callers (depends on behavior 1)
3. **Behaviors 3–5** as a group — `NewMainWindow` signature change, About menu fix, and `export_test.go` deletion are tightly coupled

## Files Changed

### Part 1 (Behaviors 1–2)
| File | Change |
|---|---|
| `internal/ui/character/fairy/fairy.go` | Default `refreshFunc` to no-op; deprecate `DisableRefresh()` |
| `cmd/cue/main.go` | Wire `SetRefreshHook` in fairy registration closure |
| `cmd/cue-uat/main.go` | Wire `SetRefreshHook` in fairy registration closure |
| `cmd/character-uat/uat_window_test.go` | Wire `SetRefreshHook` in fairy registration closure |

### Part 2 (Behaviors 3–5)
| File | Change |
|---|---|
| `internal/ui/window.go` | Add `fyne.App` param; remove `newFyneApp` var; use stored `fyneApp` in About menu |
| `internal/ui/export_test.go` | Delete |
| `internal/ui/window_layout_test.go` | Pass `test.NewApp()` to `NewMainWindow` |
| `cmd/cue/main.go` | Create `app.New()` and pass to `NewMainWindow` |

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Fairy visuals stop updating in production | Behavior 2 explicitly wires `fyne.Do` refresh at all production entry points |
| `SetRefreshHook` called after animation starts | Hook is set inside `character.Register` closure, before `return` — no frames missed |
| `NewMainWindow` signature change breaks callers | Only 2 production callers and 1 test helper to update |
| `DisableRefresh` tests break | No-op on no-op is still no-op — existing tests pass unchanged |

## Test Coverage

- **Behavior 1:** `TestConstructorDefaultRefreshIsNoOp` — verifies `IsNoopRefresh()` returns true for fresh fairy
- **Behavior 2:** `TestProductionFairyWiresRefreshHook` — verifies factory wires `SetRefreshHook`, `IsNoopRefresh()` returns false
- **Behaviors 3–5:** `TestNewMainWindowAcceptsFyneApp` — verifies `NewMainWindow` accepts injected `fyne.App`; all existing layout tests updated to pass `test.NewApp()`

## TDD Agent Stats

| Behavior | TDD Phase | Agent | Commit |
|---|---|---|---|
| 1 | RED | Test Designer | eaf98f5 |
| 1 | GREEN | Implementer | c5911ca |
| 1 | REFACTOR | Refactorer | (no changes) |
| 2 | RED | Test Designer | 92a216a |
| 2 | GREEN | Implementer | a42b395 |
| 3–5 | RED | Test Designer | 8e16851 |
| 3–5 | GREEN | Implementer | 0c9fa15 |
| 3–5 | REFACTOR | Refactorer | (no changes) |
