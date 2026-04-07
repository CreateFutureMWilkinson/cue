# Feature 082 — Graceful Shutdown (SIGINT + Clean Exit)

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | Critical |
| Status | Planned |
| Depends on | — |
| UI Tests | No |

## Problem

Two related shutdown issues:

1. **SIGINT doesn't trigger shutdown**: There is no signal handler in the application. `cmd/cue/main.go` blocks on `mainWindow.Run()` (Fyne event loop) with no mechanism to intercept OS signals. Ctrl+C in the terminal does nothing useful.

2. **Shutdown takes up to 60 seconds**: When shutdown does occur (via Quit menu), the orchestrator's polling loop waits for the next ticker tick before observing context cancellation. With a 600-second poll interval, this could theoretically block for 10 minutes (in practice, the OS kills the process first).

## Root Cause Analysis

### No Signal Handling

`cmd/cue/main.go` has zero signal handling code:
- No `signal.Notify()` for `os.Interrupt` or `syscall.SIGTERM`
- The cleanup code (lines 463-474) only runs after `mainWindow.Run()` returns
- The only way to exit gracefully is via the Fyne Quit menu item

### Slow Shutdown

`orchestrator.go` Start() method:
```go
ticker := time.NewTicker(interval)
select {
case <-ticker.C:  // blocks up to `interval` duration
case <-ctx.Done():
    return
}
```

The select properly checks `ctx.Done()`, but only at the top of each loop iteration. If the goroutine is blocked waiting on `ticker.C`, it won't observe cancellation until the next tick.

## Required Changes

### Signal Handling

1. Install a signal handler in `main.go` that listens for `os.Interrupt` and `syscall.SIGTERM`
2. On signal receipt, call `fyneApp.Quit()` to break the Fyne event loop
3. Cleanup code then runs naturally after `mainWindow.Run()` returns

### Fast Shutdown

1. Orchestrator `Stop()` should complete within a bounded timeout (e.g., 5 seconds)
2. Use a done channel or context cancellation that the ticker select also listens to, ensuring immediate exit from the polling wait
3. Add a shutdown timeout in `main.go` — if cleanup doesn't complete in N seconds, force exit

### Shutdown Sequence

```
Signal received (SIGINT/SIGTERM)
    ↓
fyneApp.Quit() — breaks Fyne event loop
    ↓
mainWindow.Run() returns
    ↓
Cleanup: orchestrator.Stop(), DB close, etc.
    ↓
Timeout guard: if cleanup exceeds 5s, log and force exit
```

## Acceptance Criteria

- Ctrl+C in terminal triggers clean shutdown
- SIGTERM triggers clean shutdown
- Shutdown completes within 5 seconds
- All resources (DB connections, goroutines) are cleaned up
- No goroutine leaks after shutdown
