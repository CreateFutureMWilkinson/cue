# Feature 065A: Calendar Add Account Form

**Phase:** Phase-6-Feature-065A
**Type:** Bugfix (Hotfix)
**Severity:** Medium
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 065 (Calendar Settings Tab), Feature 067 (Email Add Account), Feature 068 (Slack Add Account)

---

## Problem

Feature 065 added the Calendar tab to SettingsView but left the "Add Account" button wired to a noop callback (`func() {}`). The `ServiceSettingsPresenter` exposes full Calendar CRUD (`SaveCalendarAccount`, `ListCalendarAccounts`, etc.) and the SQLite backend stores calendar accounts with encrypted ICS URLs — but there is no UI form to invoke any of it. Tapping "Add Account" on the Calendar tab does nothing.

This was explicitly deferred in Feature 065's acceptance criteria:

> - [ ] Add Account opens a form for ICS URL, name, and poll interval (deferred — noop callback)
> - [ ] Submitting the form calls `ServiceSettingsPresenter.SaveCalendarAccount()` (deferred — noop callback)

## Root Cause

`settings_view.go:213`:
```go
calendarTab := newAccountTab("Calendar", func() {})
```

No `createCalendarAccountForm()` function exists. The Email (Feature 067) and Slack (Feature 068) tabs were given inline form builders but Calendar was skipped.

## Proposed Fix

Create a `createCalendarAccountForm()` function following the same pattern as `createEmailAccountForm()` and `createSlackAccountForm()`. Wire it into the Calendar tab with dynamic content switching (list view <-> add form), identical to how Email and Slack tabs work.

### Form Fields

Based on the `CalendarAccount` model (`repository/service_config.go:34-43`):

| Field | Widget | Placeholder | Validation |
|---|---|---|---|
| Name | `widget.NewEntry()` | "Account Name" | Required, non-empty |
| ICS URL | `widget.NewEntry()` | "ICS Calendar URL" | Required, non-empty |
| Poll Interval | `widget.NewEntry()` | "Poll Interval (seconds)" | Required, positive integer |

### Constructor Change

The Calendar tab in `NewSettingsView` needs to switch from the generic `newAccountTab` helper to the same dynamic content-switching pattern used by Slack and Email tabs:

```go
// Replace:
calendarTab := newAccountTab("Calendar", func() {})

// With:
calendarAccountList := container.NewVBox()
calendarAddBtn := widget.NewButton("Add Account", nil)

buildCalendarListContent := func() fyne.CanvasObject {
    return container.NewBorder(
        widget.NewLabel("Calendar Accounts"),
        calendarAddBtn,
        nil, nil,
        container.NewVScroll(calendarAccountList),
    )
}

calendarTab := container.NewTabItem("Calendar", buildCalendarListContent())

calendarAddBtn.OnTapped = func() {
    calendarTab.Content = createCalendarAccountForm(ssp, func() {
        calendarTab.Content = buildCalendarListContent()
    })
}
```

### New Function

```go
func createCalendarAccountForm(ssp *presenter.ServiceSettingsPresenter, onSaved func()) *fyne.Container {
    nameEntry := widget.NewEntry()
    nameEntry.SetPlaceHolder("Account Name")
    icsURLEntry := widget.NewEntry()
    icsURLEntry.SetPlaceHolder("ICS Calendar URL")
    pollEntry := widget.NewEntry()
    pollEntry.SetPlaceHolder("Poll Interval (seconds)")

    errorLabel := widget.NewLabel("")
    errorLabel.Hide()

    saveBtn := widget.NewButton("Save", nil)
    cancelBtn := widget.NewButton("Cancel", func() { onSaved() })

    saveBtn.OnTapped = func() {
        // validate required fields
        // validate poll interval is positive int
        // build CalendarAccount, call ssp.SaveCalendarAccount()
        // on success: onSaved()
    }

    return container.NewVBox(
        widget.NewLabel("Add Calendar Account"),
        nameEntry, icsURLEntry, pollEntry,
        errorLabel,
        container.NewHBox(saveBtn, cancelBtn),
    )
}
```

## Test Strategy

### Behaviors

1. **Calendar form display** — tapping "Add Account" replaces tab content with a form containing 3 entry fields (Name, ICS URL, Poll Interval), Save, and Cancel buttons.
2. **Calendar form validation** — empty fields show error; non-numeric poll interval shows error.
3. **Calendar form save** — valid input calls `ServiceSettingsPresenter.SaveCalendarAccount()` and returns to list view.
4. **Calendar form cancel** — tapping Cancel returns to list view without saving.

### TDD Micro-Loops

| # | Behavior | RED | GREEN | REFACTOR |
|---|---|---|---|---|
| 1 | Form display | Assert 3 entries + Save/Cancel buttons appear | Implement `createCalendarAccountForm` + wire tab | Clean up |
| 2 | Validation | Assert error label shown for empty/invalid input | Add validation logic | Clean up |
| 3 | Save | Assert presenter called with correct CalendarAccount | Wire save handler | Clean up |
| 4 | Cancel | Assert list view restored without save call | Wire cancel handler | Clean up |

## Files to Change

| File | Change |
|---|---|
| `internal/ui/settings_view.go` | Add `createCalendarAccountForm()`, replace `newAccountTab` with dynamic content switching |
| `internal/ui/settings_view_test.go` | New tests for calendar form |
| `internal/ui/settings_interaction_test.go` | New tests for calendar form interaction |
| `tests/ui/settings_acceptance_test.go` | Update calendar tab acceptance tests |

## Acceptance Criteria

- [x] Calendar "Add Account" button opens a form with Name, ICS URL, and Poll Interval fields
- [x] Form has Save and Cancel buttons
- [x] Empty fields show validation error
- [x] Non-numeric poll interval shows validation error
- [x] Valid submission calls `ServiceSettingsPresenter.SaveCalendarAccount()`
- [x] Save returns to calendar account list view
- [x] Cancel returns to calendar account list view without saving
- [x] All existing settings tests remain green
