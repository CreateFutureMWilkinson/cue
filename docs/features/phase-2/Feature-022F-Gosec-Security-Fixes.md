# Feature 022F — Gosec Security Fixes

## Overview

Resolved two remaining gosec findings in the Planner UI code: a G404 (weak RNG) false positive with incorrect suppression syntax and a G104 (unhandled error) in a UI callback.

## Findings

### G404 — Weak random number generator (`planner_view.go`)

`math/rand.Intn` was used to select a random placeholder message. This is safe for non-security-sensitive UI text, but gosec flagged it. The existing `//nolint:gosec` directive was a golangci-lint annotation that gosec does not recognize. Replaced with the correct gosec-native `// #nosec G404` inline annotation.

### G104 — Unhandled error (`app_binder.go`)

`CompleteCurrentTask()` returns an error that was silently discarded in the focus rail's Done callback. Added explicit `_ =` assignment with a comment explaining the intentional discard — UI callbacks should not panic on task completion failure.

## Design Decisions

- **`#nosec` over `crypto/rand`**: For selecting from 7 placeholder strings, `math/rand` is perfectly appropriate. Using `crypto/rand` would be over-engineering with no security benefit.
- **`_ =` over logging**: The UI package has no logger dependency. Adding one just to log a rare error in a button callback would violate the minimal-change principle. The explicit discard documents intent.

## Files Changed

| File | Change |
|---|---|
| `internal/ui/planner_view.go` | Replace `//nolint:gosec` with `// #nosec G404` |
| `internal/ui/app_binder.go` | Add `_ =` for `CompleteCurrentTask` error |
| `internal/ui/app_binder_test.go` | Add `TestBindDoneCallbackDoesNotPanicOnError` regression test |

## Test Coverage

Existing tests cover all affected code paths. New regression test verifies the Done callback does not panic when `CompleteCurrentTask` returns an error.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 36s | 24,882 | 4e5a58a |
| GREEN | Implementer | 28s | 23,413 | 8c6922c |
| REFACTOR | Refactorer | 42s | 28,103 | 8c6922c |
