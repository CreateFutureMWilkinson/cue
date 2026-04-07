# Feature 064: WizardView Missing Step Widget Rendering

**Phase:** Phase-6-Feature-064
**Type:** Bugfix
**Severity:** High
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 022D (Wizard Steps), Feature 056 (Plan/Wizard Wiring), Feature 052 (Automated UI Testing)

---

## Bug Description

`WizardView.Container()` returns a completely empty `container.NewVBox()`. The `buildState()` method populates struct fields (`taskCheckboxes`, `estimateRows`, `hasNextButton`, etc.) but **never creates Fyne widgets**. Accessor methods return data structs, not widgets. The acceptance tests use `uitest.FindWidget()` / `uitest.FindAll()` to traverse the widget tree and find nothing.

## Expected Behavior

Per UiSpec.md (lines 1123-1139):
- Step 1: checkboxes for task selection, inline creation entry, Next + Cancel buttons
- Step 2: entry fields for estimate overrides, summary text, Back + Next buttons
- Step 3: task list with up/down reorder buttons, Back + Next buttons
- Step 4: schedule selection buttons (one per card), Back button

## Actual Behavior

`v.container = container.NewVBox()` — empty. All state lives in struct fields:
- `taskCheckboxes []TaskCheckboxItem` (data, not `*widget.Check`)
- `estimateRows []presenter.TaskEstimateRow` (data, not `*widget.Entry`)
- `hasNextButton bool` (flag, not `*widget.Button`)
- `priorityList []string` (strings, not widgets)
- `scheduleCards int` (count, not buttons)

## Root Cause

Feature 022D implemented the wizard state machine and data accessors but stopped short of rendering step-specific widget trees. The view was designed as a "headless" state holder with the assumption that widget rendering would follow — it never did.

## Fix

Added `renderContainer()` method called after `buildState()` in both constructor and `Refresh()`. It clears `v.container.Objects` and dispatches to step-specific render methods based on `v.vm.CurrentStep()`.

### Step 1 — Task Selection (`renderStep1`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each `v.taskCheckboxes`: `widget.NewCheck(item.Title, ...)` bound to `v.vm.SelectTask`
- `widget.NewEntry()` — inline task creation with "New task" placeholder
- `widget.NewButton("Next", ...)` — calls `v.vm.NextStep()`
- `widget.NewButton("Cancel", ...)` — navigates to `ViewPlan`

### Step 2 — Estimates (`renderStep2`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each `v.estimateRows`: title label + entry pre-filled with effective pomos
- `widget.NewLabel(v.summaryText)` — summary
- `widget.NewButton("Back", ...)` + `widget.NewButton("Next", ...)`

### Step 3 — Priority Reorder (`renderStep3`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each `v.priorityList`: numbered label
- `widget.NewButton("Up", ...)` and `widget.NewButton("Down", ...)` when `v.hasUpDownButtons`
- `widget.NewButton("Back", ...)` + `widget.NewButton("Next", ...)`

### Step 4 — Schedule Selection (`renderStep4`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each schedule card: `widget.NewButton("Select "+strategy, ...)` calling `v.vm.SelectSchedule()`
- `widget.NewButton("Back", ...)`

### Container clearing

```go
v.container.Objects = nil
// ... add step widgets ...
v.container.Refresh()
```

## Files Modified

| File | Change |
|---|---|
| `internal/ui/wizard_view.go` | Added `renderContainer()` + `renderStep1()` through `renderStep4()` |
| `internal/ui/wizard_view_acceptance_test.go` | New file — 12 acceptance tests for widget presence per step |

## TDD Behaviors (Micro-Loops)

1. **Step 1 widget rendering** — checkboxes, entry, Next + Cancel buttons
2. **Step 2 widget rendering** — estimate entries, summary, Back + Next buttons
3. **Step 3 widget rendering** — task labels, Up/Down + Back/Next buttons
4. **Step 4 widget rendering** — schedule selection buttons, Back button

## Test Coverage

12 new acceptance tests (build tag `uitest`) verifying widget tree contents per step:

| Test | Step | Assertion |
|---|---|---|
| `TestWizardStep1ContainsCheckboxes` | 1 | >= 3 `*widget.Check` |
| `TestWizardStep1HasNavigationButtons` | 1 | "Next" + "Cancel" buttons |
| `TestWizardStep1HasInlineCreation` | 1 | >= 1 `*widget.Entry` |
| `TestWizardStep2ContainsEstimateEntries` | 2 | >= 2 `*widget.Entry` |
| `TestWizardStep2HasSummaryLabel` | 2 | Label containing "Pomodoros" |
| `TestWizardStep2HasBackAndNextButtons` | 2 | "Back" + "Next" buttons |
| `TestWizardStep3HasReorderControls` | 3 | "Up" + "Down" buttons |
| `TestWizardStep3HasBackAndNextButtons` | 3 | "Back" + "Next" buttons |
| `TestWizardStep3HasTaskLabels` | 3 | >= 2 `*widget.Label` |
| `TestWizardStep4HasScheduleSelectionButtons` | 4 | Buttons containing strategy names |
| `TestWizardStep4HasBackButton` | 4 | "Back" button |
| `TestWizardStep4NoNextButton` | 4 | No "Next" button (regression guard) |

Pre-existing integration tests in `tests/ui/` also now pass (previously 5 failures).

## TDD Agent Stats

| Behavior | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Step 1 | RED | Test Designer | ~100s | ~42,000 | 8e6881c |
| Step 1 | GREEN | Implementer | ~64s | ~29,000 | 023921e |
| Step 1 | REFACTOR | Refactorer | ~42s | ~25,000 | 8191e56 |
| Step 2 | RED | Test Designer | ~55s | ~29,000 | c34bb73 |
| Step 2 | GREEN | Implementer | ~67s | ~24,000 | 366a468 |
| Step 3 | RED | Test Designer | ~56s | ~26,000 | fb230cf |
| Step 3 | GREEN | Implementer | ~48s | ~26,000 | 73e27bd |
| Step 4 | RED | Test Designer | ~61s | ~27,000 | ea1232d |
| Step 4 | GREEN | Implementer | ~52s | ~28,000 | aca206d |
| All | REFACTOR | orchestrator | manual | — | 29ac978 |
