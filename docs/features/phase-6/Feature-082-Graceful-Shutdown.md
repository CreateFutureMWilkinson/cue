# Feature 082 — Graceful Shutdown (SIGINT + Clean Exit)

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | Critical |
| Status | Done |
| Depends on | — |
| UI Tests | No |

## Problem

Two related shutdown issues:

1. **SIGINT doesn't trigger shutdown**: No signal handler in the application. `cmd/cue/main.go` blocks on `mainWindow.Run()` (Fyne event loop) with no mechanism to intercept OS signals.

2. **No shutdown timeout**: When shutdown occurs, cleanup could theoretically block indefinitely if a component hangs.

## Design

### New Package: `internal/shutdown`

Two exported components:

- **`SignalHandler`** — Listens for `os.Interrupt` and `syscall.SIGTERM` via `signal.Notify`. Calls a `quitFn` callback exactly once (guarded by `sync.Once`). Started with a context; stops listening on context cancellation.

- **`RunCleanup(timeout, ...fns)`** — Runs cleanup functions sequentially in a goroutine with a `time.After` deadline. Returns the first error from any cleanup function, or a timeout error if the deadline fires first.

### Wiring in main.go

```
Signal received (SIGINT/SIGTERM)
    ↓
SignalHandler calls fyneApp.Quit()
    ↓
mainWindow.Run() returns
    ↓
RunCleanup(5s): character shutdown → charPresenter.Stop → appPresenter.Shutdown → orch.Stop
    ↓
If timeout: log warning and exit
```

The orchestrator's existing `select` on `ctx.Done()` already responds to cancellation immediately, so no orchestrator changes were needed.

## API

```go
// SignalHandler listens for OS signals and calls quitFn once.
func NewSignalHandler(quitFn func()) *SignalHandler
func (h *SignalHandler) Start(ctx context.Context)

// RunCleanup runs fns sequentially with a timeout.
func RunCleanup(timeout time.Duration, fns ...func() error) error
```

## Error Handling

- `RunCleanup` returns the first error from any cleanup function
- If the timeout fires, returns `"shutdown cleanup timeout after <duration>"`
- main.go logs timeout warnings but exits cleanly regardless

## Test Coverage

| Test | Description |
|---|---|
| `TestCallsQuitOnInterrupt` | Sends SIGINT to process, asserts quitFn called within 1s |
| `TestRunCleanupCompletesWithinTimeout` | Two fast fns complete without error |
| `TestRunCleanupReturnsTimeoutError` | Slow fn triggers timeout error |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (SignalHandler) | Test Designer | ~47s | ~23,500 | f549da9 |
| GREEN (SignalHandler) | Implementer | ~20s | ~22,700 | 21b132c |
| RED (RunCleanup) | Test Designer | ~39s | ~25,000 | d30feaf |
| GREEN (RunCleanup) | Implementer | ~21s | ~22,100 | 72306da |
| WIRING | orchestrator | ~2min | — | f1b8974 |
