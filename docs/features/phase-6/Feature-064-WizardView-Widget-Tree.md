# Feature 064: WizardView Missing Step Widget Rendering

**Phase:** Phase-6-Feature-064
**Type:** Bugfix
**Severity:** High
**Status:** Planned
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

## Failing Acceptance Tests

| Test | Step | What It Expects |
|---|---|---|
| `TestWizardStep1ContainsCheckboxes` | 1 | `*widget.Check` objects in tree |
| `TestWizardStep1HasNavigationButtons` | 1 | "Next" and "Cancel" `*widget.Button` |
| `TestWizardStep1HasInlineCreation` | 1 | `*widget.Entry` for task creation |
| `TestWizardStep2ContainsEstimates` | 2 | `*widget.Entry` for estimate overrides |
| `TestWizardStep2HasBackAndNext` | 2 | "Back" and "Next" `*widget.Button` |
| `TestWizardStep3HasReorderControls` | 3 | At least 2 buttons (nav + reorder) |
| `TestWizardStep3HasBackButton` | 3 | "Back" `*widget.Button` |
| `TestWizardStep4HasScheduleSelectionButtons` | 4 | At least 1 button (schedule selection) |

## Fix

Add `renderContainer()` method called after `buildState()` in both constructor and `Refresh()`. It clears `v.container.Objects` and dispatches to step-specific render methods based on `v.vm.CurrentStep()`.

### Step 1 — Task Selection (`renderStep1`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each `v.taskCheckboxes`: `widget.NewCheck(item.Title, func(checked bool) { v.vm.SelectTask(item.ID, checked) })`
- `widget.NewEntry()` — inline task creation
- `widget.NewButton("Next", func() { v.vm.NextStep(context.Background()) })`
- `widget.NewButton("Cancel", func() { v.router.NavigateTo(ViewPlan) })`

### Step 2 — Estimates (`renderStep2`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each `v.estimateRows`: `widget.NewLabel(row.Title)` + `widget.NewEntry()` pre-filled with estimate
- `widget.NewLabel(v.summaryText)` — summary, optional overload warning
- `widget.NewButton("Back", func() { v.vm.PreviousStep() })`
- `widget.NewButton("Next", func() { v.vm.NextStep(context.Background()) })`

### Step 3 — Priority Reorder (`renderStep3`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each `v.priorityList`: numbered label
- `widget.NewButton("Up", ...)` and `widget.NewButton("Down", ...)` when `v.hasUpDownButtons`
- `widget.NewButton("Back", func() { v.vm.PreviousStep() })`
- `widget.NewButton("Next", func() { v.vm.NextStep(context.Background()) })`

### Step 4 — Schedule Selection (`renderStep4`)

- `widget.NewLabel(v.stepIndicator)` — step indicator
- For each schedule card: `widget.NewButton("Select "+strategy, func() { v.vm.SelectSchedule(ctx, strategy) })`
- `widget.NewButton("Back", func() { v.vm.PreviousStep() })`

### Container clearing

```go
v.container.Objects = v.container.Objects[:0]
// ... add step widgets ...
v.container.Refresh()
```

## Files to Modify

| File | Change |
|---|---|
| `internal/ui/wizard_view.go` | Add `renderContainer()` + `renderStep1()` through `renderStep4()` |
| `internal/ui/wizard_view_test.go` | Unit tests for widget presence per step |

## TDD Behaviors (Micro-Loops)

1. **Step 1 widget rendering** — checkboxes, entry, Next + Cancel buttons
2. **Step 2 widget rendering** — estimate entries, summary, Back + Next buttons
3. **Step 3 widget rendering** — task labels, Up/Down + Back/Next buttons
4. **Step 4 widget rendering** — schedule selection buttons, Back button

## Risk Areas

- **Container clearing on Refresh** — must clear `Objects` slice and call `container.Refresh()` to trigger Fyne re-render
- **Existing accessor-based unit tests** — must remain passing; accessors return struct field data, widgets are additive
- **Step dispatch isolation** — each step should only render its own widgets; no checkbox leaking into step 2, etc.

## TDD Agent Stats

_To be filled during implementation._
