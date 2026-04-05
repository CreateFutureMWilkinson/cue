# Feature 024 Hotfix B: Fairy Refresh Thread Safety

**Phase:** Phase-3-Feature-024-Hotfix-B (bugfix)
**Status:** Planned
**Depends on:** Feature 041 (Character Package Restructure)
**Packages:** `internal/ui/character/fairy/`

---

## Overview

Fixes three related Fyne thread-safety violations in `FairyCharacter` that cause the app to freeze and become unresponsive on Wayland. Animator goroutines call `.Refresh()` on Fyne canvas objects directly, but Fyne requires all widget mutations to go through `fyne.Do()`. Additionally, `refreshFunc` was never wired to the container, so position and glow animations never visually rendered.

## Symptom

```
*** Error in Fyne call thread, this should have been called in fyne.Do[AndWait] ***
  From: /home/delphicokami/Projects/cuefix/internal/ui/character/fairy.go:286
```

- 2,854 occurrences of this error in a single UAT session
- Fairy rendered but was completely frozen — state transitions caused a small visual change but no animation
- App entirely froze, did not respond to SIGINT, had to be terminated by the OS
- Reproduced on Linux/Wayland. X11 silently tolerated the violation.

## Root Cause

Three bugs, all the same class of violation. Line numbers reference the pre-restructure layout; post-Feature-041 these will be in `internal/ui/character/fairy/fairy.go`:

| # | Bug | Effect |
|---|---|---|
| 1 | `SetBodyColor` calls `f.bodyCircle.Refresh()` directly | Primary error source (2,854 errors), eventual deadlock |
| 2 | `refreshFunc` initialized as no-op, never wired to container | `SetPosition` and `SetGlowIntensity` update state but never trigger visual re-layout |
| 3 | `stopAndUpdateState` calls `f.indicator.Refresh()` directly | Thread-safety violation from `CharacterPresenter` goroutines |

### Call chain for Bug 1 (from stack trace)

```
WorkingAnimator.Start (goroutine)
  → runAnimationLoop
    → updateFrame
      → fairy.SetBodyColor
        → bodyCircle.Refresh()
          → Fyne detects non-UI thread
            → DoAndWait blocks on channel
              → main thread also blocked → DEADLOCK
```

### Why `refreshFunc` was never wired (Bug 2)

`refreshFunc` is set to `func() {}` in the struct literal before the container exists. After the container is constructed, `refreshFunc` is never updated. The Feature 025-Hotfix-A design doc states it should default to `func() { f.container.Refresh() }`, but this was never implemented.

### Why `indicator.Refresh()` is unsafe (Bug 3)

`stopAndUpdateState` is called by `TransitionTo`, which is invoked from:
- `CharacterPresenter` event loop goroutine
- `CharacterPresenter` decay timer goroutine
- `StartupAnimator` completion callback via `go f.TransitionTo(StateIdle)`

All non-UI thread contexts.

## Fix (Option B — Single Dispatch Point)

Wire `refreshFunc` to dispatch through `fyne.Do()` after container construction. Remove all direct `.Refresh()` calls on child objects. All visual mutations route through the single thread-safe callback.

`fyne.Do` is non-blocking (`DoFromGoroutine(fn, false)` at `fyne/v2@v2.7.3/thread.go:19`), so calling it under `f.mu` is safe — no deadlock risk.

All changes apply to `internal/ui/character/fairy/fairy.go` (post-Feature-041 location).

### Change 1: Wire `refreshFunc` (Bug 2)

After container construction, add:

```go
f.refreshFunc = func() { fyne.Do(func() { f.container.Refresh() }) }
```

### Change 2: Replace `indicator.Refresh()` (Bug 3)

In `stopAndUpdateState`:

```go
// Before:
f.indicator.FillColor = stateColor(state)
f.indicator.Refresh()

// After:
f.indicator.FillColor = stateColor(state)
f.refreshFunc()
```

### Change 3: Remove `bodyCircle.Refresh()` (Bug 1)

In `SetBodyColor`, remove the direct refresh (container-level refresh covers children):

```go
// Before:
f.bodyCircle.FillColor = c
f.bodyCircle.Refresh()
f.refreshFunc()

// After:
f.bodyCircle.FillColor = c
f.refreshFunc()
```

### New Method: `SetRefreshFunc`

```go
// SetRefreshFunc replaces the refresh callback. Used for testing.
func (f *FairyCharacter) SetRefreshFunc(fn func()) { f.refreshFunc = fn }
```

Generalizes the existing `DisableRefresh()` pattern with an injectable callback for test assertions.

## Design Decisions

- **Single dispatch point over per-call wrapping** — wrapping each `.Refresh()` call in `fyne.Do()` individually would fix the immediate errors, but leaves the door open for future code to add new direct `.Refresh()` calls. Routing everything through `refreshFunc` creates a single chokepoint that is always thread-safe.
- **Container-level refresh over child refresh** — `f.container.Refresh()` propagates to all children (`bodyCircle`, `indicator`, glow layers). Individual `.Refresh()` calls are redundant and create additional thread-safety surface area.
- **`SetRefreshFunc` over internal-only fix** — an exported setter enables tests to inject a counting callback to verify all mutation paths route through `refreshFunc`, without requiring a running Fyne app.
- **`DisableRefresh` refactored as alias** — `DisableRefresh()` becomes `f.SetRefreshFunc(func() {})`, reducing duplication.

## Test Coverage

| Suite | Tests | Focus |
|---|---|---|
| `FairyCharacterSuite` | `TestAllVisualMutationsRouteThroughRefreshFunc` | Verifies `TransitionTo`, `SetBodyColor`, `SetPosition`, `SetGlowIntensity` all invoke `refreshFunc` |

### Test Strategy

Inject a counting `refreshFunc` via `SetRefreshFunc`, then assert each mutation path increments the counter:
- `TransitionTo(StateIdle)` + `Close()` → `refreshCount >= 1` (covers Bug 3: `stopAndUpdateState` now calls `refreshFunc`)
- `SetBodyColor(color)` → `refreshCount == 1` (covers Bug 1: no extra direct `bodyCircle.Refresh()`)
- `SetPosition(x, y)` → `refreshCount == 1`
- `SetGlowIntensity(v)` → `refreshCount == 1`

## Character Development Guidelines

Thread safety rules for character implementations are documented in `docs/Character-Development-Guide.md` (Step 4: Thread Safety). The guide includes a checklist, correct/incorrect code examples, and the `refreshFunc` pattern that this hotfix implements.

## Verification

```bash
just fmt && just lint && just tidy && just test
go test -race -count=1 ./internal/ui/character/fairy/...
```

Manual UAT on Wayland:
```bash
FYNE_DRIVER=wayland just run-uat 2>&1 | tee /tmp/character-uat-postfix.log
```

Expected: zero `Error in Fyne call thread` messages, fairy animates smoothly, app responds to SIGINT.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | — | — | — |
| GREEN | Implementer | — | — | — |
| REFACTOR | Refactorer | — | — | — |
