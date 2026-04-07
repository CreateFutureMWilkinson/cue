# Feature 083 — Fyne Call Thread Safety Violations

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | High |
| Status | Planned |
| Depends on | — |
| UI Tests | No |

## Problem

The application produces Fyne threading errors on stdout:

```
*** Error in Fyne call thread, this should have been called in fyne.Do[AndWait] ***
  From: /home/delphicokami/Projects/cue/internal/ui/window.go:26
```

Fyne requires all canvas/widget modifications to happen on the event loop thread via `fyne.Do()` or `fyne.DoAndWait()`. Several call sites modify UI objects from non-Fyne threads.

## Affected Call Sites

### Missing `fyne.Do()` Wrapping

1. **`MainWindow.switchCenterView()`** (window.go:219-223)
   - Directly modifies `centerStack.Objects` and calls `Refresh()`
   - Called from `CenterViewRouter.OnViewChange` callbacks, which fire from button handlers that may not be on the Fyne thread

2. **`MainWindow.SetCharacterWidget()`** (window.go:228-233)
   - Directly modifies container objects and calls `Refresh()`
   - Called from character plugin loading, which runs on a background goroutine

3. **`AppBinder` callbacks** (app_binder.go:93-103)
   - Call `Refresh()` on views from presenter callbacks
   - Presenters fire callbacks from background goroutines (orchestrator polling, timer ticks)

4. **`TimerLoop`** (timer_loop.go:79+)
   - Calls `widget.SetProgress()`, `widget.SetFlashVisible()`, `taskView.SetCurrentTask()`
   - Timer tick runs on a background goroutine

### Already Correct

- `window.go:262-264` — periodic refresh correctly wraps in `fyne.Do()`
- `main.go:387` — character refresh hook correctly wraps in `fyne.Do()`

## Required Changes

Wrap all UI mutations in `fyne.Do()`:

```go
// Before (broken)
func (m *MainWindow) switchCenterView(v CenterView) {
    content := m.viewContents[v]
    m.centerStack.Objects = []fyne.CanvasObject{content}
    m.centerStack.Refresh()
}

// After (correct)
func (m *MainWindow) switchCenterView(v CenterView) {
    content := m.viewContents[v]
    fyne.Do(func() {
        m.centerStack.Objects = []fyne.CanvasObject{content}
        m.centerStack.Refresh()
    })
}
```

Apply the same pattern to all affected call sites listed above.

## Acceptance Criteria

- No "Error in Fyne call thread" messages on stdout during normal operation
- View switching, character loading, timer updates all work without threading errors
- All UI mutations from non-Fyne threads are wrapped in `fyne.Do()` or `fyne.DoAndWait()`
