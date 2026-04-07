# Feature 074 — Failing UI Acceptance Tests for Bugs 065-073

**Phase:** Phase-6-Feature-074
**Type:** Enhancement (Precursor)
**Severity:** —
**Depends on:** 052

## Purpose

Per the UI Feature Workflow (CLAUDE.md section 13), UI acceptance tests must be committed **before** TDD micro-loops begin. This feature adds failing UI acceptance tests for all UI-affecting bugs in the 065-073 range. These tests define the expected behavior and serve as the outer verification gate for each bugfix.

## Bugs Covered

| Bug | Test Coverage |
|---|---|
| 065 — Calendar Settings Tab | Settings has 5 tabs; Calendar tab exists with Add Account button |
| 066 — PlannerView Content Rendering | Placeholder text visible in widget tree; Plan My Day button navigates to wizard |
| 067 — Email Add Account | Tapping Add Account calls presenter (callback is not empty) |
| 068 — Slack Add Account | Tapping Add Account calls presenter (callback is not empty) |
| 069 — Timer Volume Slider | Audio tab has two sliders; Timer Volume label exists and updates |
| 070 — Activity Log Overlay | Overlay uses Stack (not VSplit); semi-transparent background present |
| 072 — Wizard Reorder Buttons | Up/Down buttons call ReorderTask on the view model |
| 073 — PlannerView Button Wiring | Next/Back/CompleteTask/Abandon buttons invoke non-empty callbacks |

Bug 071 (planner subsystem wiring in main.go) is a composition-root integration issue, not UI-testable at the acceptance level. Its effects are covered indirectly by 066 and 073 tests.

## Files

- `tests/ui/bugfix_acceptance_test.go` — all failing tests for bugs 065-073

## Acceptance Criteria

- [ ] All tests compile under `ui_acceptance` build tag
- [ ] All tests FAIL (red) against the current codebase
- [ ] Each test clearly documents which bug and UiSpec AC it covers
- [ ] Tests use existing `uitest` helpers and `helpers_test.go` mocks
