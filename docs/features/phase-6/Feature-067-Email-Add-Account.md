# Feature 067 — Email Settings Add Account Callback Is Noop

**Phase:** Phase-6-Feature-067
**Type:** Bugfix
**Severity:** Medium
**Status:** Complete
**Depends on:** Feature 060

## Problem

In the Email settings tab, the "Add Account" button does nothing. Clicking it has no visible effect — no form appears, no account is created.

## Root Cause

`settings_view.go:42`: `newAccountTab("Email", func() {})` — the `onAdd` callback is an empty function literal. Feature 060 implemented the tab structure but left the Add Account callbacks as noops.

## Fix

1. The Email tab is now constructed manually (not via `newAccountTab`) so the button callback can capture and replace the tab content.
2. Tapping "Add Account" replaces the Email tab content with a form containing Entry widgets for: IMAP Host, IMAP Port, Username, Password, Poll Interval.
3. Tapping "Save" with empty fields shows a validation error label ("All fields are required").
4. Tapping "Save" with valid data calls `ssp.SaveEmailAccount()` with a new `repository.EmailAccount`, then restores the account list view.
5. Port and poll interval are validated as numeric values.

## Design Decisions

- **Inline form replacement**: Rather than a modal/dialog, the form replaces the tab content directly. This keeps the UI simple and avoids Fyne modal complexity.
- **`createEmailAccountForm` helper**: Extracted form creation into a standalone function accepting `ssp` and `onSaved` callback, keeping `NewSettingsView` readable.
- **`buildEmailListContent` closure**: Captures `emailAddBtn` and `emailAccountList` to rebuild the account list view after save or cancel.

## Files Changed

- `internal/ui/settings_view.go` — Email tab construction, `createEmailAccountForm()` helper

## Error Handling

- Empty required fields → validation error label shown
- Non-numeric port/poll interval → specific error message
- Presenter `SaveEmailAccount` failure → error displayed in form

## Test Coverage

- `TestEmailAddAccountShowsFormFields` — verifies form appears with >= 5 Entry widgets
- `TestEmailAddAccountValidationShowsError` — verifies error label on empty submission
- `TestEmailAddAccountSaveWithValidDataReplacesForm` — verifies form replaced after valid save
- UI acceptance test `TestBug067/TestEmailAddAccountCallbackIsWired` — end-to-end verification

## TDD Agent Stats

| Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Phase-6-Feature-067 | RED | Test Designer | ~35s | ~27,000 | 0b7bcfd |
| Phase-6-Feature-067 | GREEN | Implementer | ~61s | ~31,000 | b33a80f |
| Phase-6-Feature-067 | REFACTOR | Refactorer | ~48s | ~24,000 | 1095d1f |
| Phase-6-Feature-067 | RED | Test Designer | ~41s | ~28,000 | 52b31e9 |
| Phase-6-Feature-067 | GREEN | orchestrator | manual | — | b459a4b |
| Phase-6-Feature-067 | REFACTOR | orchestrator | skipped | — | — |
| Phase-6-Feature-067 | RED | Test Designer | ~39s | ~28,000 | 5c72e85 |
| Phase-6-Feature-067 | GREEN | orchestrator | manual | — | 81283b1 |
| Phase-6-Feature-067 | REFACTOR | Refactorer | ~557s | ~33,000 | 363dad7 |
