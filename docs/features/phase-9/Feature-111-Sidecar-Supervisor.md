# Feature 111: Sidecar Supervisor

**Phase:** Phase-9-Feature-111
**Status:** Planning
**Depends on:** Feature 107 (Fyne Client Re-wire), Feature 110 (TOFU Client Bootstrap)
**Packages:** `cmd/cue/sidecar/` (new), `cmd/cue/` (wiring), `internal/config/`

---

## Overview

Add `mode = "sidecar"` to `cue ui`. The client spawns its own binary as a child process (`os.Executable() server`), supervises its lifecycle, pipes its output to a daily-rolling log file, surfaces unexpected death to the user, and reaps it on shutdown.

Until 111 ships, `cue ui` requires `mode = "external"` and a separately-running `cue server`. After 111, `mode = "sidecar"` becomes the default and the user can launch `cue ui` cold and have a working app without managing the server lifecycle.

---

## SDK-Only Client Code

All HTTP and WebSocket interaction with the server flows through `pkg/client`. No raw `net/http`, `coder/websocket`, or URL construction outside the SDK. The supervisor's orphan-detection probe uses the SDK's `*client.APIClient` (specifically the `HealthInfo` method added by this feature — see "SDK Addition" below). Process supervision itself (`os/exec`, signals, file I/O) is unchanged.

---

## SDK Addition (owned by this feature)

`APIClient.Health()` exists today but returns only a nil/error signal — it does not expose the response body, which Decision 3 requires for envelope verification. Feature 111 adds a sibling method:

```go
// HealthResponse is the parsed body of GET /health. Service is the
// stable identifier of the responding service ("cue-server" for our
// server) so callers can distinguish our server from arbitrary other
// services on the same port.
type HealthResponse struct {
    Status  string `json:"status"`
    Service string `json:"service"`
}

// HealthInfo calls GET /health and returns the parsed envelope on 2xx.
// Use this when you need to verify it's actually cue-server responding;
// use Health() when you only need a liveness signal.
func (c *APIClient) HealthInfo(ctx context.Context) (*HealthResponse, error)
```

Existing `Health()` is retained — its callers (e.g., the connect-time check in Feature 107) only need a liveness signal and should not be churned.

The orphan probe in this feature uses `HealthInfo`; the response's `Service == "cue-server"` is the marker check from Decision 3.

---

## Server-Side Change (owned by this feature)

`internal/server/handler` (or wherever HealthHandler lives) currently emits `{"status":"ok"}`. Change to `{"status":"ok","service":"cue-server"}`. Update existing health tests to match. Trivial change — landed inside this feature because no prior feature needed it.

---

## TDD Requirement

**Full per-behavior TDD micro-loops apply to this feature.** Process supervision, signal handling, and crash detection are exactly the kind of subtle systems behavior that earns its keep through tests. Each behavior listed in the Implementation Sequence goes through RED → GREEN → REFACTOR with three commits.

---

## Locked Decisions

### 1. Sidecar binary discovery

The supervisor spawns `os.Executable()` (the running `cue` binary) with `["server"]` as the argument list. No PATH lookup, no configurable binary path. This is only possible because Feature 107 established `cue server` as a real subcommand.

The child inherits stdin (`/dev/null`), with stdout and stderr redirected to the log file (see Decision 5). Environment is inherited unchanged. The config path is inherited via the existing `~/.cue/config.toml` convention; no `--config` flag is passed.

### 2. Port comes from config, not from the child

`[server].port` in `~/.cue/config.toml` is the source of truth. The child reads it; the supervisor reads it. There is no port-file, no stdout parsing for "listening on…" lines. `port = 0` is rejected by both validators (established in Feature 107).

### 3. Orphan detection via API probe

Before spawning, the supervisor probes `GET host:port/api/v1/health` with a 500ms timeout:

| Probe result | Default action (`policy = "adopt"`) | Alternative (`policy = "kill"`) |
|---|---|---|
| 200 with our expected envelope | Adopt — skip spawn, return existing process info | SIGTERM old, wait, spawn new |
| Connection refused | Spawn fresh | Spawn fresh |
| 200 with unrecognized envelope | Refuse to start; another service is on this port | Refuse to start |
| Timeout | Spawn fresh (assume previous server is dead/unresponsive) | Spawn fresh |

`policy` is read from `[server].sidecar_orphan_policy`. Default `"adopt"` per the rationale that the probe positively identifies our server.

The "expected envelope" check is a marker field in the health response (e.g., `service: "cue-server"`). Server work, if needed, is small and lands as part of this feature.

### 4. PID file for diagnostic visibility only

The supervisor writes `~/.cue/runtime/server.pid` after spawn so external tools can find the process. The PID file is **not** load-bearing for orphan logic — orphan detection uses the API probe. The file is removed on clean stop; stale files are tolerated (the API probe handles correctness).

### 5. Daily-rolling logs

Sidecar stdout + stderr are piped to `~/.cue/logs/server-YYYY-MM-DD.log`. The supervisor opens the file at spawn time and runs a midnight ticker that closes-and-reopens against the new date. No external rotation library; pure stdlib.

Old log files are retained indefinitely. Configurable retention is a future feature.

### 6. Crash channel

The supervisor exposes a `Crashed() <-chan error` channel. When `cmd.Wait()` returns a non-nil exit error during normal operation (i.e., not during `Stop()`), the error is published on the channel. Feature 107's signal-handler / dialog code consumes this to surface the same error UI used for initial-connect failure.

The channel is buffered (capacity 1) and never closed during the supervisor's lifetime; receivers must use `select` with a context done channel.

### 7. Stop semantics

`Stop(ctx)` sends SIGTERM, waits up to 5 seconds, falls back to SIGKILL. After SIGKILL the supervisor returns even if `Wait()` hasn't returned (it eventually will, but Stop doesn't block on it).

`Stop` is idempotent. Calling Stop twice (e.g., signal handler + deferred cleanup) is safe.

### 8. Injectability for Feature 107's boot test

The supervisor exposes a small interface so Feature 107's loop 13 boot test can substitute a stub backed by `httptest.NewServer`:

```go
type Supervisor interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Crashed() <-chan error
}
```

Production code uses `*ProcessSupervisor`; tests use `*StubSupervisor` from a test-helper package or in-test type. The `cue ui` action takes a `Supervisor` interface, not a concrete type.

### 9. Sidecar test isolation via `TestMain` sentinel

Tests in `cmd/cue/sidecar/` use a self-re-exec pattern: when `CUE_TEST_FAKE_SERVER` is set, the test binary runs as a fake cue-server (writing pidfile, listening on port, optionally crashing or hanging). This avoids needing a separate fake-binary build artifact and runs cleanly under `just test-pkg`.

```go
const fakeServerEnv = "CUE_TEST_FAKE_SERVER"

func TestMain(m *testing.M) {
    if mode := os.Getenv(fakeServerEnv); mode != "" {
        runFakeServer(mode)
        return
    }
    os.Exit(m.Run())
}
```

Production code never reads the sentinel.

### 10. Config schema additions

`internal/config` gains:

- `[server].mode = "sidecar"` is now an accepted value (was rejected by Feature 107's `ValidateForClient` until this feature).
- `[server].sidecar_orphan_policy = "adopt" | "kill"` (default `"adopt"`).
- Default `mode` flips from `"external"` to `"sidecar"` once this feature ships.

---

## Implementation Sequence (TDD micro-loops)

Each row is one behavior. Three commits per row.

### Phase A: Sidecar package

| # | Behavior | Test |
|---|---|---|
| A1 | `Supervisor.Start` spawns child via `os.Executable() server` | `TestMain` sentinel; assert child PID alive after Start. |
| A2 | Spawned child writes PID file at `~/.cue/runtime/server.pid` | Assert file exists with correct PID after Start. |
| A3 | `Supervisor.Stop` sends SIGTERM and child exits cleanly | Start → Stop → assert child exited with non-error within budget. |
| A4 | `Supervisor.Stop` falls back to SIGKILL after 5s | Sentinel mode "ignore-sigterm"; Stop with 5s budget; assert child killed. |
| A5 | `Supervisor.Stop` is idempotent | Start → Stop → Stop; assert second Stop returns nil and does not panic. |
| A6 | Orphan probe via `client.HealthInfo` returns `service=cue-server` + `policy=adopt` → skip spawn | Pre-start a fake server on port returning the cue-server envelope; supervisor.Start should not spawn a second; assert single PID. |
| A7 | Orphan probe healthy + `policy=kill` → SIGTERM + spawn | Pre-start fake server; supervisor.Start with policy=kill; assert old PID dead, new PID alive. |
| A8 | Orphan probe via `client.HealthInfo` returns transport error (connection refused) → spawn fresh | Empty port; assert spawn occurs. |
| A9 | Orphan probe via `client.HealthInfo` returns 200 with `service != "cue-server"` → refuse to start | Pre-start non-cue server returning `{"status":"ok","service":"other"}`; assert Start returns error and does not spawn. |
| A10 | `Crashed()` emits when child exits unexpectedly | Sentinel mode "exit-1"; Start → wait → assert error received on channel. |
| A11 | `Crashed()` does NOT emit during normal Stop | Start → Stop → assert no value on channel within budget. |
| A12 | Daily log writer opens `~/.cue/logs/server-YYYY-MM-DD.log` | Start → assert log file exists with today's date in name. |
| A13 | Daily log writer reopens file when date changes | Inject a clock that advances past midnight; assert new file opened, old file closed. |
| A14 | Sidecar stdout + stderr land in the log file | Sentinel mode "noisy"; assert log file contains expected output. |

### Phase B: Server-side health envelope marker + SDK `HealthInfo`

| # | Behavior | Test |
|---|---|---|
| B1 | `/health` (and `/api/v1/health`) returns `{"status":"ok","service":"cue-server"}` | Update existing health handler tests; assert `service` field present and equal to `"cue-server"`. |
| B2 | SDK `APIClient.HealthInfo` returns parsed envelope on 2xx | `httptest.NewServer` returning the new shape; assert returned struct populated correctly. |
| B3 | SDK `APIClient.HealthInfo` returns `*APIError` on non-2xx | `httptest.NewServer` returning 500; assert `errors.As(err, &apiErr)` succeeds and `apiErr.StatusCode == 500`. |

### Phase C: Config schema and validation

| # | Behavior | Test |
|---|---|---|
| C1 | `ValidateForClient` accepts `mode = "sidecar"` | Was rejected in Feature 107; flip the table-driven test row. |
| C2 | `ValidateForClient` accepts `sidecar_orphan_policy ∈ {"adopt", "kill"}` and rejects others | Table-driven test. |
| C3 | Default value for `mode` changes from `"external"` to `"sidecar"` | Load minimal TOML; assert mode resolves to "sidecar". |

### Phase D: Wiring into `cue ui` action

| # | Behavior | Test |
|---|---|---|
| D1 | `cue ui` with `mode=sidecar` starts the supervisor before health check | Boot test with stub supervisor; assert `Start` called before health probe. |
| D2 | `cue ui` with `mode=external` does NOT start the supervisor | Boot test; assert stub supervisor never invoked. |
| D3 | Signal handler calls `supervisor.Stop` before exit | Boot test; trigger SIGINT-equivalent; assert Stop invoked. |
| D4 | Crash channel surfaces error dialog mid-session | Boot test; emit on Crashed(); assert dialog state visible. |

Each loop ends with `just fmt` before its commit.

---

## Wiring Verification

After all phases:

1. `grep -rn ErrNotImplemented cmd/cue/sidecar` (non-test) — empty.
2. `cmd/cue` `ui` action consumes the `Supervisor` interface; production code instantiates `*ProcessSupervisor`.
3. `os.Executable()` is called only from `cmd/cue/sidecar/`; not duplicated.
4. PID file path and log path use the `internal/config` helpers, not hardcoded strings.
5. `just check`, `just test`, `just test-ui`, `just security`, `just vulncheck` all green.
6. Manual smoke (macOS): `cue ui` (cold) → spawns sidecar → app works → SIGINT → both processes exit cleanly.

---

## Acceptance Criteria

- `cmd/cue/sidecar/` package exists with `Supervisor` interface and `*ProcessSupervisor` implementation.
- All 24 behaviors above have passing tests.
- `pkg/client.APIClient.HealthInfo` exists and is the only mechanism the supervisor uses to probe the server. No raw `net/http` calls in `cmd/cue/sidecar/`.
- `cue ui` with `mode = "sidecar"` (the new default) spawns the server child, claims a TOFU token via Feature 110, opens the Fyne window, all UI features function.
- `cue ui` with `mode = "external"` continues to work as in Feature 107 (no spawn).
- Orphan running server is adopted by default; `policy = "kill"` SIGTERMs and respawns.
- Sidecar logs land in `~/.cue/logs/server-YYYY-MM-DD.log` and roll at midnight.
- Sidecar crash mid-session surfaces the same error dialog as initial-connect failure.
- SIGINT/SIGTERM/window-close on `cue ui` cleanly stops the sidecar (SIGTERM, fallback SIGKILL after 5s).
- `[server].mode` defaults to `"sidecar"` in `ValidateForClient`.
- `[server].sidecar_orphan_policy` is validated and defaults to `"adopt"`.
- `/api/v1/health` includes a `service: "cue-server"` marker field.

---

## Risk Areas

1. **macOS process behaviors.** Primary target. `os.Executable()` is reliable on macOS; PID-based signaling works as expected. Linux secondary; Windows out of scope.

2. **Orphan from `cue ui` SIGKILL.** The UI process can be killed without running cleanup, leaving the sidecar orphaned. The next `cue ui` startup probes `/api/v1/health` and (per `policy=adopt` default) reuses the running sidecar. Feature, not bug.

3. **PID reuse race.** If our PID file points at a long-dead PID that the kernel has reused for another process, the API probe (not the PID file) is the correctness check. Worst case: the probe returns 200 with the wrong envelope and the supervisor refuses to start, surfacing a clear error. Mitigated by Decision 3.

4. **Log file growth.** Indefinite retention. Acceptable for v1; revisit if it bites.

5. **Sentinel `TestMain` interaction with package isolation.** Since the test binary self-re-execs, fake-server invocations don't share state with the test process. Each fake spawn is fresh. Worth a comment in the package docstring.

6. **Concurrent `cue ui` invocations during the gap before Feature 112 lands.** Two `cue ui` started simultaneously both probe an empty port, both spawn a sidecar, second bind fails. After Feature 112 (UI lock), this is impossible. During the 111-but-not-yet-112 window, document as a known limitation.

7. **Default mode flip is a behavior change.** Today's `cue ui` (after 107, before 111) requires `mode = "external"`. After 111, default flips to `"sidecar"`. Users upgrading mid-development cycle will see different behavior on cold start. No real users to worry about; flag in CHANGELOG.

8. **`pkg/client` public-API addition.** `HealthInfo` and `HealthResponse` become part of the SDK's published surface. Cue is pre-1.0 and Cue itself is currently the only SDK consumer, so the impact is zero, but the addition is noted in CHANGELOG under `### Added` for `pkg/client`.

---

## Estimate

- New code: ~250 LOC supervisor + ~80 LOC config schema + ~40 LOC wiring + ~25 LOC SDK `HealthInfo` + ~30 LOC SDK tests + ~30 LOC server health envelope marker + ~400 LOC supervisor tests.
- Behaviors: 24, each one full TDD micro-loop.
- Total: ~72 commits + 1 docs commit. ≈ 3.3 working days.
