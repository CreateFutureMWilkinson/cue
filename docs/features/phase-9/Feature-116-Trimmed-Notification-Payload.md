# Feature-116: Trimmed Notification Payload + Account Deep Links

## Overview

The notification panel was receiving the entire `RawContent` of every message
across the wire — for email this meant the full body, including HTML
attachments quoted into the IMAP fetch. Users with ADHD-relevant traffic
volumes reported "far too much content" rendering. There was also no way to
jump from a notification card back to the source message: the SDK and the UI
both lacked a permalink, and the per-account `WebURL` already stored in
`ServiceConfigRepository` never reached the watcher → message → wire path.

This feature reshapes the notification list payload and threads the
account-level `WebURL` end-to-end so clicking a notification opens the
workspace or inbox in the user's default browser.

## Behaviour on the Wire

`notificationItem` (the `/api/v1/notifications` list shape) gains two fields
and trims `content` source-specifically:

| Source | `subject` | `content` | `web_url` |
| ------ | --------- | --------- | --------- |
| email  | populated | empty     | account WebURL |
| slack  | empty     | capped at 280 chars | account WebURL |

The detail endpoint (`GET /api/v1/notifications/{id}`) still returns the full
`raw_content` and the new `subject` + `web_url` fields, so the expanded /
detail view stays lossless. The decision is "list = preview, detail = full",
not "the body is gone".

## Data Model

`repository.Message` gains `Subject string` and `WebURL string`. SQLite
persists both via two idempotent `ALTER TABLE … ADD COLUMN … DEFAULT ''`
migrations. The shared `messageColumnsStr` and the `scanMessage` row scanner
were extended in lock-step.

For email, `Subject` is populated by the watcher from `email.Subject`; the
existing `RawContent = email.Subject + "\n" + email.Body` concatenation
remains so the LLM scorer, regex rules engine, embedding store, and vector
advisor (which all read `RawContent`) keep their full context unchanged.
Splitting subject from body in `RawContent` would have silently removed the
subject from routing decisions, a regression we deliberately avoided.

## Watcher → Message Wiring

`SlackWatcherConfig` and `EmailWatcherConfig` each gained a `WebURL` field.
The watcher stamps that value onto every `repository.Message` it emits
(channel-join, plain message, threaded message — all paths). `composition.go`
threads `acct.WebURL` from `ServiceConfigRepository` through both watcher
factories: the startup `registerWatchersFromDB` loop and the runtime
`createWatcherFactory` closure used by the service manager.

Per-message permalinks (Slack `archives/<channel>/p<ts>`, Gmail
`#inbox/<id>`) were considered and explicitly deferred — they require
provider-specific URL construction. The account-level URL is universal and
ships now; per-message permalinks can layer on top later by populating a
non-empty `msg.WebURL` from a source-specific helper instead of the account
default.

## UI

`NotificationCard.WebURL` is populated by `BuildNotificationCards` from
`msg.WebURL`. The presenter's `MessagePreview` field now sources from
`msg.Subject` when the source is email (falling back to truncated
`RawContent` when subject is empty), and from truncated `RawContent` for
slack and other sources.

`MainWindow` wires `panel.SetOnNotificationClick` to a `openWebURL(app)`
helper that parses the URL and invokes `fyne.App.OpenURL`. Empty URLs are
treated as a no-op (logged at `Warn` only on parse / dispatch failure), so
notifications without a configured account URL silently swallow the click
rather than erroring. Resolve and Dismiss buttons remain on the expanded
card; the previous click → detail-dialog path is replaced by click →
deep-link.

## SDK

`client.Message` and `client.NotificationSummary` both gained `Subject` and
`WebURL` with the matching `json:"subject"` / `json:"web_url"` tags. The
adapter at `cmd/cue/adapters/messages.go` copies both fields onto
`repository.Message` in `messageDTOToRepo`. The `MessageDetail` shape was not
touched (detail endpoint still serves the full record; subject and WebURL
will round-trip through the existing `Content` and detail fetch when needed
— added incrementally if a future view needs them).

## Tests

Layered coverage:

- **UI acceptance** (`tests/ui/notification_acceptance_test.go`):
  `TestEmailCardPreviewShowsSubjectOnly` and
  `TestClickOnNotificationPassesWebURL`.
- **Presenter unit** (`internal/ui/presenter/notification_card_test.go`):
  `TestEmailMessageUsesSubjectForPreview`,
  `TestCardWebURLPopulatedFromMessage`.
- **Email watcher** (`internal/service/watcher/email_test.go`):
  `TestPoll_SubjectFieldPopulated`.
- **SQLite repository** (`internal/repository/implementation/sqlite/`):
  `TestInsertPersistsSubjectAndWebURL`.
- **Server handler** (`internal/server/handler/notification_test.go`):
  `TestListNotificationsTrimsContentAndAddsSubjectAndWebURL`.
- **SDK** (`pkg/client/messages_test.go`):
  `TestListNotificationsDecodesSubjectAndWebURL`.
- **Adapter** (`cmd/cue/adapters/messages_test.go`):
  `TestQueryByStatusCarriesSubjectAndWebURL`.

## Design Decisions

- **Trim at the wire, not in storage.** `RawContent` keeps the full message
  for scoring/embedding; the server handler is the only layer that knows
  the wire is bandwidth-sensitive.
- **Subject as a first-class field.** Avoided overloading `RawContent`
  with newline-delimited "subject\nbody" parsing on every read.
- **Account-level URL, not per-message permalink.** Universal across
  providers; per-message links can be added incrementally without
  re-shaping the data model.
- **Click → URL replaces click → detail dialog.** The expanded card still
  exposes Resolve / Dismiss; the dialog was a redundant overlay.

## Validation

- `just test` — passes (pre-existing flaky `TestWebSocketHandler_HappyPath`
  ignored; passes in isolation).
- `just test-ui` — green.
- `just lint`, `just tidy` — clean.
- `just security`, `just vulncheck` — only pre-existing findings in
  `cmd/cue-fake/` and stdlib vulnerabilities unrelated to this work.
