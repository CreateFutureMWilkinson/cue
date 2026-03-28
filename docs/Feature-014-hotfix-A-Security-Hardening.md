# Feature 014 Hotfix A: Security Hardening

**Phase:** Hotfix-A (security)
**Status:** Done
**Packages:** `internal/alert/`, `cmd/cue/`

---

## Overview

Addresses all findings from `gosec` and `govulncheck` scans. Two static analysis findings (G404 weak RNG, G304 path traversal) and one symbol-level vulnerability plus four transitive dependency CVEs.

## Findings & Fixes

### gosec G404 — Weak RNG (CWE-338)

**Location:** `internal/alert/alert.go:173`
**Issue:** `math/rand/v2.IntN` used to select random audio file.
**Fix:** Replaced with `crypto/rand.Int(rand.Reader, ...)` for CSPRNG-based index selection.

### gosec G304 — Path Traversal (CWE-22)

**Location:** `internal/alert/beep_player.go:105`
**Issue:** `os.Open(path)` with user-configurable directory + directory-listed filenames, no bounds checking.
**Fix (two layers):**
1. `filterSupportedAudioFiles` in `alert.go` now rejects filenames where `filepath.Clean(filepath.Join(dir, name))` escapes the configured `AudioDir`.
2. `BeepPlayer.decodeAudioFile` uses `os.OpenRoot` (Go 1.24+) to scope all file access under the configured audio directory at the OS level.

### GO-2026-4815 — x/image TIFF OOM

**Module:** `golang.org/x/image` v0.24.0 → v0.38.0
**Risk:** Symbol-level — reachable via Fyne UI image decoding pipeline.

### GO-2026-4441, GO-2026-4440, GO-2025-3595, GO-2025-3503 — x/net vulnerabilities

**Module:** `golang.org/x/net` v0.35.0 → v0.45.0
**Risk:** Package/module level — infinite parse loops, quadratic parsing, XSS, proxy bypass.

## Design Decisions

- **os.Root over manual validation** — Go 1.24's `os.OpenRoot` provides kernel-level path scoping, stronger than `filepath.Clean` prefix checks alone. Both are applied for defense in depth.
- **Variadic constructor** — `NewBeepPlayer(audioDir ...string)` maintains backward compatibility with existing unit tests that don't need scoped access.
- **nosec on fallback** — `openDirect` (test-only path) carries `#nosec G304` annotation since production always uses `os.Root`.

## Error Handling

- `crypto/rand` failure: returns error (no silent fallback to weak RNG).
- Path traversal filenames: silently filtered out. If all files rejected, falls back to beeep notification.
- `os.OpenRoot` failure: returns wrapped error with "root directory" context.

## Integration Points

- `cmd/cue/main.go` updated to pass `cfg.Notification.AudioDir` to `NewBeepPlayer`.
- No interface changes to `AudioPlayer` or `AlertService`.

## Test Coverage

| Test | Purpose |
|---|---|
| `TestSelectAudioFileRejectsPathTraversal` | All-traversal filenames → error, beeper fallback |
| `TestSelectAudioFilRejectsMixedTraversalAndValid` | Mixed filenames → only safe files selected |
| `TestSelectAudioFileAcceptsNormalPaths` | Normal filenames unaffected |
| `TestBeepPlayerRejectsPathOutsideRoot` | os.Root rejects absolute/traversal paths outside audioDir |

## TDD Agent Stats

| Impl Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Feature-014-hotfix-A (alert) | RED | Test Designer | 53s | 28,453 | 62a513b |
| Feature-014-hotfix-A (alert) | GREEN | Implementer | 44s | 30,561 | 64fdca1 |
| Feature-014-hotfix-A (alert) | REFACTOR | Refactorer | 97s | 37,520 | 9bdc35f |
| Feature-014-hotfix-A (beep) | RED | Test Designer | 45s | 29,222 | db981ea |
| Feature-014-hotfix-A (beep) | GREEN | Implementer | 85s | 43,746 | 47506be |
| Feature-014-hotfix-A (beep) | REFACTOR | Refactorer | 108s | 45,715 | 2e19ed0 |
