# Feature 112: UI Single-Instance Lock

**Phase:** Phase-9-Feature-112
**Status:** Planning
**Depends on:** Feature 107 (Fyne Client Re-wire)
**Packages:** `cmd/cue/uilock/` (new), `cmd/cue/` (wiring)

---

## Overview

Enforce that only one `cue ui` process can run at a time. A second invocation acquires the lock, fails fast with a clear message, and exits non-zero — without spawning a sidecar, claiming a token, or initializing Fyne.

The mechanism is a kernel-managed advisory file lock (`flock(2)`) on `~/.cue/runtime/ui.lock`. The kernel auto-releases the lock when the holding process dies, so no PID-file staleness logic is needed.

This feature is small but earns full TDD discipline because file locking is easy to get subtly wrong, and a broken lock means either silent overlap (two UIs racing on tokens, sidecars, audio) or false-positive lockout (user can't start the app at all).

---

## TDD Requirement

**Full per-behavior TDD micro-loops apply to this feature.** Each behavior listed in the Implementation Sequence goes through RED → GREEN → REFACTOR with three commits.

---

## Locked Decisions

### 1. Lock primitive: `flock(2)` via `golang.org/x/sys/unix`

`golang.org/x/sys/unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)` is the cleanest cross-Unix advisory lock. Returns `EWOULDBLOCK` immediately if the lock is held; no busy-wait. macOS and Linux both supported by the same call. Windows is out of scope (Cue is Unix-only).

The lock file is opened with `O_RDWR|O_CREAT` mode `0600`. The fd is held for the lifetime of the process. The kernel releases the lock on process exit (clean or crashed), so no cleanup logic is required for the crashed case.

### 2. Lock file location

`~/.cue/runtime/ui.lock`. The directory is shared with Feature 111's PID file. The supervisor and uilock packages are independent — neither reads the other's files.

### 3. Optional content: PID for diagnostic message

After acquiring the lock, the process writes its own PID to the file (truncating any previous content). This is purely informational — readers can show "another cue ui is running (PID N)" — but the lock semantics are determined by `flock`, not the file content.

### 4. macOS notarization caveat

Documented in the package docstring: macOS App Sandbox places restrictions on file locks within sandboxed containers. Cue ships unsigned today, so this is fine. If notarization for App Store distribution becomes a goal in the future, the lock path may need to move to a sandbox container directory.

### 5. Public API

```go
package uilock

// Acquire takes an exclusive advisory lock on the ui.lock file. Returns a
// Handle that holds the lock until Release is called, or until the process
// exits (whichever comes first).
//
// Returns ErrAlreadyHeld if another process holds the lock. The error wraps
// the PID currently holding it (read from the lock file content).
//
// Returns wrapped errors for filesystem failures (permission denied,
// directory missing, etc.). The lock file directory ~/.cue/runtime is
// expected to exist; callers should ensure it has been created before
// calling Acquire.
func Acquire() (*Handle, error)

type Handle struct { /* unexported */ }

// Release explicitly releases the lock. Idempotent. Process exit also
// releases the lock automatically; explicit Release is for orderly shutdown.
func (h *Handle) Release() error

// HoldingPID returns the PID encoded in the error, if known.
type ErrAlreadyHeld struct {
    PID int
}
func (e *ErrAlreadyHeld) Error() string { ... }
```

### 6. Error message in `cue ui`

When `cue ui` receives `*ErrAlreadyHeld` from `uilock.Acquire`, it prints to stderr and exits with code 1:

```
another cue ui is running (PID N); if it appears unresponsive, kill -9 N
```

If `PID == 0` (file empty or unparseable), the message drops the PID hint:

```
another cue ui is running; if it appears unresponsive, kill it manually
```

### 7. Acquisition order in `cue ui` action

The lock is the **first** thing the action does after argument parsing — before config load, before sidecar spawn, before TOFU bootstrap. Failing fast is the whole point. Release is in the deferred cleanup at the bottom of the action, after sidecar.Stop.

```
ParseArgs → uilock.Acquire → defer Release →
  config.Load → ... → sidecar.Start → ... → app.Run() →
  (cleanup)
```

If sidecar.Start fails after the lock is held, Release still runs via defer.

---

## Implementation Sequence (TDD micro-loops)

| # | Behavior | Test |
|---|---|---|
| 1 | `Acquire` returns `*Handle` on first call when no lock exists | `s.T().TempDir()`-based lock path; assert handle non-nil, no error. |
| 2 | `Acquire` returns `*ErrAlreadyHeld` when another process holds the lock | Spawn subprocess that holds the lock and signals readiness; main process calls Acquire; assert `errors.As(err, &ErrAlreadyHeld{})`. |
| 3 | `*ErrAlreadyHeld` includes the holding PID | Subprocess writes its PID; main reads error; assert `err.PID == subprocess PID`. |
| 4 | `*ErrAlreadyHeld` falls back to PID 0 when file is empty/unparseable | Manually create empty lock file with another process holding flock; assert `err.PID == 0`. |
| 5 | `Release` releases the lock; subsequent Acquire succeeds | Acquire → Release → Acquire; assert second Acquire succeeds. |
| 6 | `Release` is idempotent | Acquire → Release → Release; assert no error or panic. |
| 7 | Lock auto-releases on process death | Subprocess acquires then dies (kill -9); main Acquire after death; assert success. |
| 8 | Acquire returns wrapped error when lock directory does not exist | `s.T().TempDir() + "/missing"`; assert error wraps a clear message identifying the missing directory. |
| 9 | Acquire returns wrapped error on permission denied | Lock file with no read/write perms (test as non-root only); assert wrapped permission error. |
| 10 | Lock file content after Acquire is the holder's PID as decimal text | Acquire; read file; assert `strconv.Atoi` returns `os.Getpid()`. |
| 11 | Wiring: `cue ui` calls `Acquire` before any other initialization | Boot test with hooks asserting order; second concurrent invocation exits with prescribed message and code 1. |
| 12 | Wiring: prescribed error message includes PID when available | Run two `cue ui` invocations in test; capture stderr of second; assert message format. |
| 13 | Wiring: `cue ui` cleanup releases the lock | Boot test; trigger shutdown; assert lock file unlocked (third invocation succeeds). |

Each loop ends with `just fmt` before its commit. Conventional commits:

- `test(uilock): failing test for ...`
- `feat(uilock): implement ... [tests pass]`
- `refactor(uilock): improve ...`

---

## Wiring Verification

After all loops:

1. `grep -rn ErrNotImplemented cmd/cue/uilock` (non-test) — empty.
2. `cmd/cue` `ui` action calls `uilock.Acquire` before any other initialization (search for `uilock.Acquire` in `cmd/cue/`).
3. Lock path uses the `internal/config` runtime-dir helper, not a hardcoded string.
4. `just check`, `just test`, `just test-ui`, `just security`, `just vulncheck` all green.
5. Manual smoke (macOS): start `cue ui`, then start a second `cue ui` in another terminal — second exits non-zero with the prescribed message; first continues unaffected.

---

## Acceptance Criteria

- `cmd/cue/uilock/` package exists with `Acquire`, `*Handle.Release`, `*ErrAlreadyHeld`.
- Lock file at `~/.cue/runtime/ui.lock`, contains holder PID, exclusive flock.
- All 13 behaviors above have passing tests.
- Coverage ≥80% on the package.
- Second `cue ui` invocation exits non-zero with the prescribed message.
- Kernel auto-release on process death verified by subprocess test.
- macOS notarization caveat documented in package docstring.
- `just security` and `just vulncheck` clean.

---

## Risk Areas

1. **Subprocess test pattern.** Behaviors 2, 3, 4, 7 require a subprocess that holds the lock and either signals readiness or dies. Use the `TestMain` self-re-exec pattern from Feature 111's playbook, or alternatively `exec.Command(os.Args[0], "-test.run=TestSubprocess$")` with a hidden test that holds the lock until killed. Settle on one pattern early in the loop.

2. **NFS / network filesystems.** `flock(2)` semantics on NFS are notoriously inconsistent. `~/.cue/` is in the user's home directory, which on most macOS/Linux setups is local. If a user mounts their home over NFS, the lock may not work correctly. Out of scope; document in the package docstring.

3. **PID parse safety.** Lock file content is read by `cue ui` to format the error message. If a previous holder wrote garbage, parsing must not panic — fall back to PID 0. Covered by behavior 4.

4. **Race between Acquire and PID write.** Acquire takes the lock, then writes the PID. Between those two operations, another process trying to Acquire will fail with the lock held but read an empty/old PID. Acceptable: error message falls back to PID 0 path. The user still knows to kill the holder.

5. **Cleanup of ui.lock file itself.** The file is never deleted (only truncated/rewritten). It persists across runs as a permanent fixture in `~/.cue/runtime/`. This is intentional — the file's existence is meaningless; only the flock state matters.

---

## Estimate

- New code: ~80 LOC implementation + ~250 LOC tests + ~20 LOC wiring.
- Behaviors: 13, each one full TDD micro-loop.
- Total: ~39 commits + 1 docs commit. ≈ 1.5 working days.
