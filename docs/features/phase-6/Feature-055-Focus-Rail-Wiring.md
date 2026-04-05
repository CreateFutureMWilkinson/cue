# Feature 055: Focus Rail Wiring

**Phase:** Phase-6-Feature-055
**Type:** Bugfix
**Severity:** High
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 017 (Focus Rail), Feature 016 (Three-Column Layout)

---

## Bug Description

The focus rail in the left column of the three-column layout is a placeholder `widget.NewLabel("Focus")` instead of the real `FocusRail` widget. The `FocusRail` struct exists in `focus_rail.go` with timer ring, navigation buttons, and task display — but `window.go` never instantiates it.

## Expected Behavior

The left column should contain the `FocusRail` widget with: countdown timer ring, current task name, Done/Back/Plan/Review buttons — as specified in UiSpec.md and implemented in `focus_rail.go`.

## Actual Behavior

`window.go:63` creates `widget.NewLabel("Focus")` and uses that as the left column content. The real `FocusRail` type is dead code.

## Root Cause

The `FocusRail` widget was built (Feature 017) but never wired into the `MainWindow` constructor. The placeholder label was left from initial scaffolding.

## Proposed Fix

Replace the placeholder label in `window.go` with a proper `NewFocusRail(viewRouter)` instantiation. Pass required dependencies (view router, timer presenter if available) through the `MainWindow` constructor.

## Test Strategy

- RED: Structural test (using Feature 052 framework) asserting the left column contains a `FocusRail` container, not a `widget.Label`
- RED: Interaction test — tap Plan button in focus rail, verify view router navigates to Plan view
- GREEN: Wire `FocusRail` into `window.go`
- REFACTOR: Remove any dead placeholder code
