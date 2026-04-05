# Feature 031 Hotfix B: Key File Path Traversal Fix (G304)

**Phase:** Phase-4-Feature-031-Hotfix-B (security)
**Status:** Done
**Packages:** `internal/secret/`

---

## Overview

Addresses gosec G304 (CWE-22) path traversal finding in `NewKeyFileEncryptor` where `os.ReadFile(keyPath)` used a variable-controlled path. Replaces all direct `os.Stat`/`os.ReadFile`/`os.WriteFile` calls with `os.OpenRoot`-scoped equivalents, matching the pattern established in Feature 014A for audio file access.

## Finding

### gosec G304 -- Path Traversal (CWE-22)

**Location:** `internal/secret/secret.go:38`
**Issue:** `os.ReadFile(keyPath)` accepts a variable path, which gosec flags as a potential file inclusion vulnerability. A path containing `../` components could traverse outside the intended directory.

## Design Decisions

### Two-Layer Defense

1. **Input validation** -- Reject paths containing `..` components before any file I/O. Uses `strings.SplitSeq` + `filepath.ToSlash` to detect traversal in the raw (uncleaned) path.
2. **Kernel-level scoping** -- `os.OpenRoot(dir)` (Go 1.24+) scopes all subsequent `Open`/`OpenFile`/`Stat` calls to the key file's parent directory. Even if validation is bypassed, the OS prevents escaping the root.

### Path Handling

- `filepath.Clean(keyPath)` normalizes the path
- `filepath.Dir` / `filepath.Base` split into directory and filename
- All file operations go through the `os.Root` handle

## API

No API changes. `NewKeyFileEncryptor(keyPath string)` signature unchanged. New error returned for paths containing `..` traversal components.

## Error Handling

| Condition | Error message |
|---|---|
| Path contains `..` | `key path must not contain path traversal (..): <path>` |
| Directory doesn't exist | `opening key directory: <os error>` |

## Test Coverage

| Test | Description |
|---|---|
| `TestPathTraversalPrevention` | Constructs `allowedDir/../secretDir/key` path; asserts rejected |
| All existing tests | Unchanged behavior for legitimate paths |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~55s | ~23,600 | 9cc6199 |
| GREEN | orchestrator + manual | ~60s | ~23,600 | df5056b |
| REFACTOR | orchestrator | ~30s | — | 519cbc0 |
