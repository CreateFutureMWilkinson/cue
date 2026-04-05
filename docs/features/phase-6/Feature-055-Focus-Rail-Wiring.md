# Feature 055: Focus Rail Wiring

**Phase:** Phase-6-Feature-055
**Type:** Bugfix
**Severity:** High
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 017 (Focus Rail), Feature 016 (Three-Column Layout), Feature 052 (Automated UI Testing)

---

## Bug Description

The focus rail in the left column of the three-column layout was a placeholder `widget.NewLabel("Focus")` instead of the real `FocusRail` widget. The `FocusRail` struct existed in `focus_rail.go` with timer ring, navigation buttons, and task display — but `window.go` never instantiated it.

## Expected Behavior

The left column should contain the `FocusRail` widget with: countdown timer ring, current task name, Done/Back/Plan/Review/Settings buttons — as specified in UiSpec.md and implemented in `focus_rail.go`.

## Actual Behavior

`window.go:58` created `widget.NewLabel("Focus")` and used that as the left column content. The real `FocusRail` type was dead code.

## Root Cause

The `FocusRail` widget was built (Feature 017) but never wired into the `MainWindow` constructor. The placeholder label was left from initial scaffolding.

## Fix

Replaced the placeholder label in `window.go` with `NewFocusRail(viewRouter).Container()`. When `viewRouter` is nil (test edge case), falls back to the label placeholder. The `FocusRail` uses `SetOnViewChange` (single callback slot) while `MainWindow` uses `AddOnViewChange` (listener list) — both are compatible and fire on navigation.

## Test Coverage

| Test | File | Type |
|---|---|---|
| `TestMainWindowLeftColumnIsFocusRailContainer` | `composition_test.go` | Tier 1 — structural assertion that left column is `*fyne.Container`, not `*widget.Label` |
| `TestWiredFocusRailPlanButtonNavigates` | `navigation_interaction_test.go` | Tier 2 — interaction test: tap Plan button in wired FocusRail, verify router navigates to `ViewPlan` |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (structural) | Test Designer | ~551s | ~28,000 | cbcf332 |
| GREEN | Implementer | ~35s | ~22,000 | 8d5c94a |
| REFACTOR | Refactorer | ~23s | ~22,000 | (no changes) |
| RED (interaction) | Test Designer | ~86s | ~29,000 | adbff27 |
