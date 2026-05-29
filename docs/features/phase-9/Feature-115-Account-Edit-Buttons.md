# Feature-115: Account Edit Buttons in Settings

## Overview

Each Slack, Email, and Calendar account row in the Settings panel now renders
an **Edit** button alongside the existing **Delete** button. Tapping Edit
swaps the tab content into the same form used for adding an account, but
prefilled with the existing account's values; saving routes through the
already-existing presenter `EditXAccount` methods (preserving the account ID)
so the record is updated in place instead of duplicated.

Prior to this feature, account editing required deleting the account and
re-adding it from scratch — losing the credentials in the process and forcing
the user to re-enter them.

## Design Decisions

- **Reuse the form, not a separate edit dialog.** The Add and Edit forms have
  identical field layouts. `createXAccountForm` was generalised to accept an
  optional `existing *repository.XAccount`: nil means add, non-nil means edit.
  When non-nil, fields are prefilled and Save routes through the presenter's
  `EditXAccount` method (preserving `existing.ID`); otherwise it routes
  through `SaveXAccount` (a fresh insert).
- **Per-type list helpers.** The previous `listAccountWidgets(ssp, "slack")`
  string-switched helper was replaced with three typed helpers
  (`listSlackAccountWidgets`, `listEmailAccountWidgets`,
  `listCalendarAccountWidgets`), each accepting an `onEdit
  func(*repository.XAccount)` callback. This keeps the per-row Edit closure
  type-safe and avoids the awkward shape of one helper returning rows whose
  callback signatures differ by account type.
- **Forward-declared tab + builder.** The `onEdit` callback needs to mutate
  the tab content, but the tab is created after the closure is constructed
  (the builder produces the initial content). The composition uses
  forward-declared `var slackTab *container.TabItem` and
  `var buildSlackListContent func() ...` so the closure captures the
  variables before assignment, exactly mirroring the existing Add Account
  pattern. No new mutable state.
- **Presenter Edit methods were already present.** `EditSlackAccount`,
  `EditEmailAccount`, and `EditCalendarAccount` existed on
  `ServiceSettingsPresenter` (each calls `Upsert*Account` on the repo and
  re-asserts the watcher state via the toggler). This feature is a pure UI
  wiring of methods that previously had no caller from the Fyne client.

## Public API Surface

No public API changes. Internal helper signatures changed:

- `createSlackAccountForm(ssp, existing *repository.SlackAccount, onSaved func()) *fyne.Container`
- `createEmailAccountForm(ssp, existing *repository.EmailAccount, onSaved func()) *fyne.Container`
- `createCalendarAccountForm(ssp, existing *repository.CalendarAccount, onSaved func()) *fyne.Container`
- `listAccountWidgets(ssp, accountType string)` removed; replaced with three typed helpers.

## Error Handling

Save errors from Edit (validator failures, transport errors) are surfaced
through the same `errorLabel` already used by the Add flow: the form stays
open and the error message is displayed. Validators are no-op when not
configured (production wires real Slack/Email/Calendar validators in
`cmd/cue/main.go` via `presenter.WithXValidator` options).

## Integration Points

- `internal/ui/settings_view.go` — form generalisation, per-type list
  helpers, edit-mode wiring.
- `internal/ui/presenter/service_settings_presenter.go` — already-existing
  `EditXAccount` methods now have a UI caller for the first time.
- `tests/ui/helpers_test.go` — `mockServiceConfigRepo.UpsertXAccount` updated
  to perform real upsert-by-ID semantics (replace if ID matches an existing
  entry, otherwise append) so tests can distinguish edit from add.

## Test Coverage

UI acceptance tests in `tests/ui/settings_acceptance_test.go`:

- `TestSlackAccountRowHasEditButton`,
  `TestEmailAccountRowHasEditButton`,
  `TestCalendarAccountRowHasEditButton` — assert the Edit button is rendered
  per row.
- `TestSlackEditPrefillsFormAndUpdatesInPlace`,
  `TestEmailEditPrefillsFormAndUpdatesInPlace`,
  `TestCalendarEditPrefillsFormAndUpdatesInPlace` — assert that tapping Edit
  prefills every field (including the email encryption Select), saving the
  edited form replaces the existing repo entry (count stays 1, ID
  preserved), and the updated label appears in the list view.

Existing settings tests (Add/Save flows, validation failures, empty-state
labels, etc.) continue to pass against the upgraded mock repo.

## TDD Agent Stats

| Phase             | TDD Phase | Agent              | Notes                                                                                          |
| ----------------- | --------- | ------------------ | ---------------------------------------------------------------------------------------------- |
| Phase-9-Feature-115 | UI Tests  | Inline             | Six new UI acceptance tests written directly; mock repo upgraded to upsert-by-ID semantics.    |
| Phase-9-Feature-115 | Green     | Inline             | Form generalisation + typed list helpers; no new presenter methods (Edit\*Account pre-existed). |
| Phase-9-Feature-115 | Refactor  | —                  | No refactor commit; implementation arrived clean (three symmetric tab blocks, forward-declared vars). |
