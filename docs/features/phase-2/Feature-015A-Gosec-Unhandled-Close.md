# Feature 015 Hotfix A: Gosec Unhandled db.Close()

**Phase:** Phase-2-Feature-015-Hotfix-A (security)
**Status:** Done
**Packages:** `internal/repository/implementation/sqlite/`

---

## Overview

Addresses 6 gosec G104 (CWE-703) findings in Feature 015 (Todo List) SQLite repository constructors. The `db.Close()` calls in error-handling paths did not explicitly handle the returned error, triggering gosec's "Errors unhandled" rule.

## Findings & Fixes

### gosec G104 — Errors Unhandled (CWE-703)

**Locations (6 total):**
- `todo_impl.go` lines 64, 69, 74 — constructor error paths for WAL, foreign keys, table creation
- `category_impl.go` lines 47, 52, 57 — constructor error paths for WAL, foreign keys, table creation

**Issue:** Bare `db.Close()` calls with return value silently discarded. gosec flags these as unhandled errors.

**Fix:** Changed `db.Close()` to `_ = db.Close()` in all 6 locations, explicitly acknowledging the discarded error. This matches the existing pattern in `message_impl.go`.

**Rationale:** In these error paths, a higher-priority error is already being returned (the PRAGMA or CREATE TABLE failure). The `db.Close()` error is secondary — logging or wrapping it would obscure the root cause. Explicit `_ =` assignment satisfies gosec while documenting the intent.

### Consistency improvement

**Location:** `message_impl.go` constructor

**Issue:** `message_impl.go` did not enable `PRAGMA foreign_keys = ON`, unlike `todo_impl.go` and `category_impl.go`.

**Fix:** Added `PRAGMA foreign_keys = ON` with proper `_ = db.Close()` error path to align all three SQLite constructors.

## Design Decisions

- **Explicit discard over error wrapping** — Wrapping `db.Close()` errors alongside the primary error would add complexity without value. The primary error (PRAGMA failure, table creation failure) is what the caller needs to act on.
- **AST-based regression tests** — Rather than runtime tests (which can't detect static analysis issues), the compliance tests use `go/parser` and `go/ast` to scan source files for bare `db.Close()` calls. This catches the exact pattern gosec flags.
- **Regression guard on message_impl.go** — The test suite includes `message_impl.go` (which was already compliant) to prevent future regressions.

## Test Coverage

- `gosec_compliance_test.go` — 3 test cases (one per impl file) using AST parsing to verify all `db.Close()` calls have explicit return value handling.
- All existing tests remain green.
- gosec now reports 0 issues.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 42s | 23,739 | b7c2918 |
| GREEN | Implementer | 28s | 19,315 | 2ddafa1 |
| REFACTOR | Refactorer | 66s | 37,291 | a93a27f |
