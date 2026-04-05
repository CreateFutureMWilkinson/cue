# Feature 052: Automated UI Testing Framework

**Phase:** Phase-6-Feature-052
**Type:** Enhancement
**Status:** Done
**Packages:** `internal/ui/uitest/`, `internal/ui/`

---

## Overview

Reusable automated UI testing framework for the Fyne GUI using Fyne's built-in `fyne.io/fyne/v2/test` package. Two tiers of verification: structural tests (widget tree assertions) and interaction tests (simulated user events), enabling CI-driven validation of UI behavior without human intervention.

## Problem

Existing UI tests validated presenter logic only — no tests verified that widgets were actually present in containers, that navigation worked end-to-end, or that user interactions produced expected UI state changes. Bugs like unwired components (BUG-003, BUG-004) went undetected because no test checked whether the real widget was in the layout.

## Design

### Helper Package (`internal/ui/uitest/`)

Generic tree-walking helpers with Go type parameter inference:

```go
func FindWidget[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) (T, bool)
func RequireWidget[T fyne.CanvasObject](t *testing.T, root fyne.CanvasObject, predicate func(T) bool) T
func FindAll[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) []T
```

Implementation walks `*fyne.Container.Objects` recursively depth-first. Also supports any type with an `Objects() []fyne.CanvasObject` method. Handles nil roots gracefully.

### Tier 1: Structural / Widget Tree Tests

1. **Widget tree assertion helpers** — 7 self-validating tests on known widget trees (flat containers, nested containers, nil roots).

2. **Container composition tests** (`composition_test.go`) — 4 tests verify `MainWindow.Content()` exposes the widget tree: outer HSplit, inner HSplit, center stack container, notification area. New `Content()` method on `MainWindow` delegates to `fyne.Window.Content()`.

3. **View router content tests** (`view_content_test.go`) — 5 tests verify each view (Character, Plan, Wizard, Settings) returns correct child widget type when active. Settings with real presenters verified to contain `*container.AppTabs` with 4 named tabs.

### Tier 2: Interaction Tests

4. **Navigation interaction tests** (`navigation_interaction_test.go`) — 5 tests use `test.Tap()` on FocusRail buttons (Plan, Back, Settings) and verify `CenterViewRouter.CurrentView()` switches correctly. Full round-trip tested.

5. **Notification interaction tests** (`notification_interaction_test.go`) — 3 tests verify NotificationPanel widget tree contains header label, list widget, and correct item count via `uitest` helpers.

6. **Settings interaction tests** (`settings_interaction_test.go`) — 4 tests verify SettingsView widget tree: AppTabs presence, default tab selection, tab content labels, non-nil content for all tabs.

## API Changes

### New Package: `internal/ui/uitest/`

- `FindWidget[T]` — first matching widget by type + predicate
- `RequireWidget[T]` — same but fails test on miss (`t.Fatalf`)
- `FindAll[T]` — all matching widgets

### New Method: `MainWindow.Content() fyne.CanvasObject`

Exposes the window's widget tree for structural testing. Delegates to `fyne.Window.Content()`.

## Out of Scope

- Visual regression / screenshot comparison (Tier 3)
- Accessibility testing
- Multi-window or OS-level window management testing

## Test Coverage

| File | Tests | Tier |
|---|---|---|
| `uitest_test.go` | 7 | Self-validation |
| `composition_test.go` | 4 | Tier 1 |
| `view_content_test.go` | 5 | Tier 1 |
| `navigation_interaction_test.go` | 5 | Tier 2 |
| `notification_interaction_test.go` | 3 | Tier 2 |
| `settings_interaction_test.go` | 4 | Tier 2 |
| **Total** | **28** | |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~69s | ~28,000 | 042262a |
| GREEN | Implementer | ~44s | ~24,000 | d9fbe80 |
| REFACTOR | Refactorer | ~78s | ~25,000 | fc68535 |
| RED | Test Designer | ~698s | ~29,000 | 26ac1da |
| GREEN | orchestrator | — | — | edce20b |
| — | Test Designer | ~86s | ~37,000 | 34fc8c0 |
| — | Test Designer | ~54s | ~33,000 | b0c3c21 |
| — | Test Designer | ~45s | ~26,000 | fe80e07 |
| — | Test Designer | ~693s | ~27,000 | ae2afad |

## Integration Points

- `internal/ui/window.go` — `MainWindow` primary test target, new `Content()` method
- `internal/ui/presenter/` — presenters provide mock data for UI tests
- All Phase 6 bugfix features (053–062) should include Tier 1/2 tests using this framework
