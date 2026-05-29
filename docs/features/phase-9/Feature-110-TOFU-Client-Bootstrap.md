# Feature 110: TOFU Client Bootstrap

**Phase:** Phase-9-Feature-110
**Status:** Planning
**Depends on:** Feature 106 (API Client SDK), Feature 108 (TOFU Pairing — server side)
**Enables:** Feature 107 (Fyne Client Re-wire) — consumer
**Packages:** `cmd/cue/auth/` (new)

---

## Overview

Build the client-side TOFU bootstrap library that Feature 107's `cue ui` action consumes. This feature owns:

1. **On-disk persistence** of the plaintext bearer token (`~/.cue/client-token`, mode `0600`, atomic save).
2. **Bootstrap flow** that loads the token if present, or triggers the SDK's transparent first-client auto-issue (Feature 106 Loop 4) and persists the issued token.
3. **Error taxonomy** so Feature 107 can render meaningful Fyne dialogs.

This is a **library feature** — the `cmd/cue/auth/` package is built standalone with full TDD discipline and waits for Feature 107 to consume it.

---

## SDK-Only Client Code

All HTTP and WebSocket interaction with the server flows through `pkg/client`. No raw `net/http`, `coder/websocket`, or URL construction outside the SDK. If a needed primitive is missing, the work to add it is owned by whichever feature first needs it. This keeps transport concerns — auth headers, retry logic, error wrapping, WebSocket reconnection — in one place. Feature 110 consumes `*client.APIClient` and `client.AuthClient` only; it does not import `net/http`.

---

## Important Context: SDK Already Handles First-Client Auto-Issue

The SDK's `pkg/client.APIClient.doJSON` (Feature 106 Loop 4) transparently handles the TOFU first-client flow at the transport layer:

1. SDK sends an authenticated request without a token.
2. Server responds `401` with body `{"error":{"code":"TOKEN_ISSUED"},"token":"..."}` if no tokens exist on the server.
3. SDK auto-detects the `TOKEN_ISSUED` error code, calls `SetToken(token)`, and retries the request once.
4. Caller sees a normal successful response. `client.Token()` now returns the issued token.

This means Feature 110's bootstrap does NOT make an explicit "claim" call. It makes **any authenticated probe request**, then reads `sdk.Token()` afterwards.

For paired-but-unknown clients (the second client connecting to a server with existing tokens), the response is plain `401 UNAUTHORIZED` without a `TOKEN_ISSUED` body. The SDK does not retry; the caller sees the error. Feature 110 surfaces this as `ErrPairingRequired` and Feature 107 routes it to a clear error dialog.

---

## TDD Requirement

**Full per-behavior TDD micro-loops apply to this feature.** Each behavior listed in the Implementation Sequence below goes through RED → GREEN → REFACTOR with three commits. CLAUDE.md §13 applies in full.

---

## Locked Decisions

### 1. Pairing-as-second-client is OUT of scope

Feature 110 ships only the **auto-issue path**. The Initiate → Poll → user-approves flow (used when a second client arrives at a server that already has tokens) is deferred. Rationale: Cue is a single-user local-first app; the typical case is fresh-server-fresh-client, which auto-issue covers. If a user invalidates their on-disk token while the server retains it, the recovery is `cue server reset-auth` followed by re-pairing as first-client.

Adding the pairing flow later is non-breaking — it adds a new function alongside `Bootstrap`, not a change to it.

### 2. Token storage location and permissions

The client persists its plaintext bearer token to `~/.cue/client-token` (file mode `0600`). The path is fixed; no per-user override. The directory `~/.cue/` is assumed to exist (created by the existing config bootstrap).

The token file contains exactly the plaintext token — no JSON wrapper, no metadata. Trailing whitespace is stripped on read.

### 3. Storage abstraction

```go
type TokenStore interface {
    Load(ctx context.Context) (string, error)         // returns ErrNoToken if absent
    Save(ctx context.Context, token string) error     // 0600 perms, atomic via temp + rename
    Delete(ctx context.Context) error                 // for reset
}

var ErrNoToken = errors.New("no client token on disk")
```

The default `FileStore` implementation persists to `~/.cue/client-token`. Tests inject an in-memory store.

### 4. Bootstrap entry point

```go
// Probe is a function that performs any authenticated request through the
// SDK. Its purpose is to trigger the SDK's TOKEN_ISSUED auto-retry on a
// fresh server. It returns nil on success, *client.APIError on HTTP error.
type Probe func(ctx context.Context, sdk *client.APIClient) error

// DefaultProbe issues GET /api/v1/auth/tokens via auth.ListTokens. Cheap,
// idempotent, and only succeeds when the client is authenticated.
var DefaultProbe Probe = func(ctx context.Context, sdk *client.APIClient) error {
    _, err := client.NewAuthClient(sdk).ListTokens(ctx)
    return err
}

// Bootstrap loads the on-disk token into the SDK if present. If no token
// is on disk, it calls probe to trigger the SDK's transparent first-client
// auto-issue, then persists whatever token the SDK ends up holding.
//
// On return:
//   - nil error: sdk.Token() is non-empty and the token is on disk.
//   - ErrPairingRequired: server has tokens but this client isn't paired.
//   - ErrServerUnreachable: network error during probe.
//   - ErrTokenStore{Unreadable,Unwritable}: disk error.
func Bootstrap(ctx context.Context, store TokenStore, sdk *client.APIClient, probe Probe) error
```

`Probe` is parameterised so tests can inject a stub that doesn't require a real server. Production callers pass `DefaultProbe`.

### 5. Bootstrap semantics

```
1. token, err = store.Load(ctx)
2. if err == nil:
       sdk.SetToken(token)
       return nil   // happy path: existing token loaded
3. if err is not ErrNoToken:
       return ErrTokenStoreUnreadable wrapping err   // do NOT probe
4. // err == ErrNoToken: trigger auto-issue path
5. probeErr = probe(ctx, sdk)
6. if probeErr == nil:
       // SDK silently auto-issued a token via TOKEN_ISSUED retry
       newToken = sdk.Token()
       if newToken == "":
           return error: probe succeeded without auto-issue (server bug?)
       saveErr = store.Save(ctx, newToken)
       if saveErr != nil:
           return ErrTokenStoreUnwritable wrapping saveErr
                  (error message includes the orphaned token for recovery)
       return nil
7. // probe returned an error
8. if probeErr is *client.APIError with Code == UNAUTHORIZED:
       return ErrPairingRequired
9. if probeErr is a transport/network error:
       return ErrServerUnreachable wrapping probeErr
10. return probeErr   // unexpected, surface as-is
```

Key safety: step 3 refuses to probe when the disk read fails for any reason other than "file absent." A transient permission error must NOT silently re-pair against a server that has already issued us a token.

### 6. Error taxonomy

| Sentinel | Meaning | Caller action |
|---|---|---|
| `ErrPairingRequired` | server has tokens; this client isn't paired (plain 401, no TOKEN_ISSUED) | Feature 107: dialog suggesting `cue server reset-auth` or future pairing UX |
| `ErrServerUnreachable` | network error during probe | Feature 107: Retry / Quit dialog |
| `ErrTokenStoreUnreadable` | disk error reading the token file | log path + permission, advise user |
| `ErrTokenStoreUnwritable` | disk error writing the claimed token | log path; error message includes the orphaned token for emergency recovery |

Each sentinel wraps the underlying cause via `fmt.Errorf("...: %w", err)`.

### 7. Atomic save

`FileStore.Save` writes to `~/.cue/client-token.tmp` first, fsyncs, then renames to `~/.cue/client-token`. Crash mid-write leaves the previous token (if any) intact.

### 8. Concurrency

Bootstrap is called once at process startup, before any other client work. There is no concurrent caller. `FileStore` does not need internal locking; the single-instance UI lock (Feature 112) guarantees mutual exclusion across processes.

### 9. No retry, no backoff

A single probe attempt is made. If the server is unreachable, the user is shown the error and chooses Retry — which calls Bootstrap again. No internal backoff/retry logic in this library.

---

## Implementation Sequence (TDD micro-loops)

Each row is one behavior — one failing test, one minimal implementation, one refactor pass. Three commits per row.

### Phase A: TokenStore (FileStore implementation)

| # | Behavior | Test |
|---|---|---|
| A1 | `FileStore.Load` returns the token from disk, trailing whitespace stripped | Write `"abc\n"` to a `s.T().TempDir()` path; assert `"abc"` returned. |
| A2 | `FileStore.Load` returns `ErrNoToken` when the file is absent | Empty TempDir; assert `errors.Is(err, ErrNoToken)`. |
| A3 | `FileStore.Load` returns wrapped error on permission denied | File with `0000` mode; assert error wraps the permission failure (not a sentinel — `Bootstrap` wraps with the appropriate caller-facing sentinel). |
| A4 | `FileStore.Save` writes the file with mode `0600` | Save token; `os.Stat`; assert `Mode().Perm() == 0600`. |
| A5 | `FileStore.Save` is atomic (crash mid-write preserves previous content) | Pre-populate with token A; inject failure mid-write via a writer that errors; assert token A still readable from disk and no `.tmp` file remains visible to readers. |
| A6 | `FileStore.Delete` removes the file | Save then Delete; assert subsequent `Load` returns `ErrNoToken`. |

### Phase B: Bootstrap

| # | Behavior | Test |
|---|---|---|
| B1 | Bootstrap returns nil and calls `sdk.SetToken` when store has a token | Mock store returns `"T1"`; mock probe panics if called; assert `sdk.Token() == "T1"` after return. |
| B2 | Bootstrap probes when store returns `ErrNoToken`, persists `sdk.Token()` after success | Mock store returns `ErrNoToken`; probe stub calls `sdk.SetToken("T2")` and returns nil; assert `store.Save("T2")` invoked and Bootstrap returns nil. |
| B3 | Bootstrap returns `ErrPairingRequired` on plain 401 | Probe returns `*client.APIError{Code: ErrCodeUnauthorized}`; assert `errors.Is(err, ErrPairingRequired)`. |
| B4 | Bootstrap returns `ErrServerUnreachable` on transport error | Probe returns a wrapped `net.OpError` or context-based error (anything that is NOT `*client.APIError`); assert `errors.Is(err, ErrServerUnreachable)`. |
| B5 | Bootstrap returns `ErrTokenStoreUnreadable` on non-`ErrNoToken` Load error AND does not probe | Mock store returns IO error; probe stub panics if called; assert `errors.Is(err, ErrTokenStoreUnreadable)` and probe was not invoked. |
| B6 | Bootstrap returns `ErrTokenStoreUnwritable` when probe succeeds but Save fails | Probe stub sets token; mock store.Save returns IO error; assert `errors.Is(err, ErrTokenStoreUnwritable)` and the error message contains the orphaned token. |
| B7 | Bootstrap returns a clear error when probe succeeds but `sdk.Token()` remains empty | Probe stub returns nil without setting a token; assert error mentions "probe succeeded without auto-issue" (server-bug guard). |

### Phase C: Default probe + integration

| # | Behavior | Test |
|---|---|---|
| C1 | `DefaultProbe` issues `GET /api/v1/auth/tokens` | Use `httptest.NewServer` recording the path; call `DefaultProbe`; assert the recorded path. |
| C2 | End-to-end: Bootstrap with `DefaultProbe` against a fresh-server `httptest` performs auto-issue and persists | Spin up `httptest` returning 401 + TOKEN_ISSUED on first request, then 200 on retry; verify `sdk.Token()` non-empty post-Bootstrap, file persisted, second Bootstrap call short-circuits. |

Each loop ends with `just fmt` immediately before its commit.

Conventional commits:
- `test(auth): failing test for ...`
- `feat(auth): implement ... [tests pass]`
- `refactor(auth): improve ...`

---

## Wiring Verification

This is a library feature, so wiring verification is light:

1. `grep -rn ErrNotImplemented cmd/cue/auth/` (non-test) — empty.
2. `cmd/cue/auth/` package compiles with `just check ./cmd/cue/auth/...`.
3. `just test-pkg ./cmd/cue/auth/...` green.
4. `just test-coverage` reports ≥80% on `cmd/cue/auth/`.

The package has no consumer until Feature 107 lands. This is acceptable per project guidance: a fully-tested library waiting for its caller is not a half-implementation.

---

## Acceptance Criteria

- `cmd/cue/auth/` package exists and exports: `TokenStore` interface, `FileStore` impl, `Probe` type, `DefaultProbe` value, `Bootstrap` function, and the four error sentinels.
- Token persisted at `~/.cue/client-token` with mode `0600`.
- Atomic save (temp + rename); crash mid-save preserves previous token.
- All 15 behaviors above have passing tests (6 + 7 + 2).
- Coverage ≥80% on the package.
- Bootstrap correctly drives the SDK's transparent TOKEN_ISSUED auto-issue — no explicit pairing-claim call.
- `just security` and `just vulncheck` clean.
- `cmd/cue` does not yet import `cmd/cue/auth/` (consumer arrives in Feature 107).

---

## Risk Areas

1. **Probe choice (`/api/v1/auth/tokens`).** Listing tokens returns the freshly-auto-issued token and any prior tokens. Slightly chatty for a probe, but semantically clean and uses an existing endpoint. Alternative: introduce a `/api/v1/whoami` endpoint in a future feature; for 110 this is sufficient.

2. **macOS keychain alternative.** Storing a plaintext token in a `0600` file is acceptable for a local-first dev tool but weaker than macOS Keychain. Out of scope for v1; revisit if Cue ever ships notarized.

3. **Token leakage in `ErrTokenStoreUnwritable`.** `ErrTokenStoreUnwritable` deliberately includes the orphaned plaintext token so the user can recover manually. Documented in the package docstring as a deliberate trade-off (losing a token because we redacted it from the error would be worse than the small leak risk in a local error dialog/log).

4. **Path expansion.** `~` expansion uses the existing `internal/config` helper (Feature 001A). If that helper changes, this package's resolved path could shift. Mitigate by depending on the helper rather than reimplementing tilde expansion.

5. **Stale token on server reset.** If the user runs `cue server reset-auth`, the client's on-disk token becomes invalid. The next Bootstrap call still reads it from disk, sets it on the SDK, and returns nil — the failure surfaces on the *first authenticated request* during the session, as a `*client.APIError{Code: ErrCodeUnauthorized}`. Feature 107's error handling must catch this and surface a "token rejected — re-pair required" dialog. NOT 110's job, but worth flagging for 107's risk register.

---

## Estimate

- New code: ~100 LOC implementation + ~250 LOC tests.
- Behaviors: 15, each one full TDD micro-loop (3 commits).
- Total: ~45 commits + 1 docs commit. ≈ 1 working day.
