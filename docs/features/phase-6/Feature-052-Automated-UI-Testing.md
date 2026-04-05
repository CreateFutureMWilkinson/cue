# Feature 052: Automated UI Testing Framework

**Phase:** Phase-6-Feature-052
**Type:** Enhancement
**Status:** Planned
**Packages:** `internal/ui/`, `internal/ui/presenter/`

---

## Overview

Establish a reusable automated UI testing framework for the Fyne GUI using Fyne's built-in `fyne.io/fyne/v2/test` package. The framework provides two tiers of verification: structural tests (widget tree assertions) and interaction tests (simulated user events), enabling CI-driven validation of UI behavior without human intervention.

## Problem

All existing UI tests validate presenter logic only — no tests verify that widgets are actually present in containers, that navigation works end-to-end, or that user interactions (tap, type, scroll) produce the expected UI state changes. Bugs like unwired components (BUG-003, BUG-004) went undetected because no test checked whether the real widget was in the layout.

## Design

### Tier 1: Structural / Widget Tree Tests

Use `test.LaidOutObjects()` and `test.WidgetRenderer()` to assert component presence in containers without rendering.

**Behaviors:**

1. **Widget tree assertion helpers** — Utility functions that walk `test.LaidOutObjects()` to find widgets by type and optional predicate (e.g., "find a `*widget.Button` with text 'Plan'"). Returns the found object or fails the test with a descriptive message.

2. **Container composition tests** — Verify that `MainWindow` contains the expected top-level structure: focus rail in the left column, center view router in the middle, notification panel on the right. Assert these are real widgets, not placeholder labels.

3. **View router content tests** — Verify that each view (Character, Plan, Wizard, Settings) contains the correct child widgets when active. E.g., Plan view contains `ScheduleTree` and `TodoListView`, not `widget.NewLabel("Plan")`.

### Tier 2: Interaction Tests

Use `test.Tap()`, `test.Type()`, and canvas event simulation to drive user interactions and verify resulting state.

**Behaviors:**

4. **Navigation interaction tests** — Tap Plan/Review/Back buttons in the focus rail and verify the center view router switches to the correct view.

5. **Notification interaction tests** — Tap a notification card and verify the detail dialog appears. Tap Resolve and verify the notification is removed from the list.

6. **Settings interaction tests** — Navigate to Settings, verify tab switching works, verify form controls are present and respond to input.

### Test Infrastructure

All tests use `test.NewTempApp(t)` and `test.NewTempWindow(t, content)` for automatic cleanup. The existing `export_test.go` pattern (swapping `app.New()` for `test.NewApp()`) remains the foundation. No display server required — runs headless in CI via Fyne's in-memory test driver.

### Helper Package

A small `internal/ui/uitest/` helper package containing:

```go
// FindWidget walks LaidOutObjects and returns the first widget matching the type and predicate.
func FindWidget[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) (T, bool)

// RequireWidget calls FindWidget and fails the test if not found.
func RequireWidget[T fyne.CanvasObject](t *testing.T, root fyne.CanvasObject, predicate func(T) bool) T

// FindAll returns all widgets matching the type and predicate.
func FindAll[T fyne.CanvasObject](root fyne.CanvasObject, predicate func(T) bool) []T
```

## Out of Scope

- Visual regression / screenshot comparison (Tier 3) — too fragile across platforms
- Accessibility testing — Fyne does not expose an accessibility tree
- Multi-window or OS-level window management testing

## Test Strategy

The framework is self-validating: its own tests verify the helpers work correctly on known widget trees. Integration tests then use the helpers to verify real UI components.

## Error Handling

No new runtime error paths. Test failures produce descriptive messages including the widget type sought, the predicate description, and the actual widget tree contents.

## Integration Points

- `internal/ui/export_test.go` — existing app factory swap for test mode
- `internal/ui/window.go` — `MainWindow` is the primary test target
- `internal/ui/presenter/` — presenters provide mock data for UI tests
- All Phase 6 bugfix features (053–061) should include Tier 1/2 tests using this framework
