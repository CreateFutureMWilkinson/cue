# Feature-115A: Service Config Field Round-Trip (hotfix on Feature-115)

## Overview

Hotfix on Feature-115 that completes the Edit-buttons promise. The Edit
button prefills the form from the account returned by
`ssp.ListXAccounts(ctx)`, but for an Email account the user observed the
form showing `WebURL` blank and `PollIntervalSeconds` as `0`. Saving
without re-entering those fields silently overwrote the previously
configured values with empty/zero — a data-loss bug introduced because
Feature-115 turned previously hidden API gaps into a user-visible
regression.

The DB schema and `repository.XAccount` structs already carry these
fields. The gap was at the wire boundary: `slackAccountItem`,
`emailAccountItem`, `calendarAccountItem`, the matching `create*Request`
DTOs, and the SDK mirrors omitted them. The adapter dropped them on
`slackDTOToRepo`/`emailDTOToRepo`/`calendarDTOToRepo`. This hotfix
extends the API to round-trip the missing fields end-to-end across all
three account types.

## Fields Added on the Wire

| Account  | Added                                            | Already on the wire                                      |
| -------- | ------------------------------------------------ | -------------------------------------------------------- |
| Slack    | `web_url`, `poll_interval_seconds`, `username`   | `name`, `workspace_id`, `enabled`, `bot_token` (write)   |
| Email    | `web_url`, `poll_interval_seconds`               | `name`, `imap_host/port`, `username`, `encryption`, `enabled`, `password` (write) |
| Calendar | `poll_interval_seconds`                          | `name`, `ics_url`, `enabled`                             |

Slack `username` is the user's `@handle` used by the Settings list
label `Slack: %s (@%s)`. Not a secret; exposed on both request and
response.

## Design Decisions

- **Symmetric extension.** Same six DTOs (3 response items, 3 request
  bodies), same three conversion helpers, same three adapter
  upsert-builders, same SDK mirror. Keeps all three account types
  parallel — no special-casing.
- **Secrets remain write-only.** `bot_token` and `password` are still
  accepted on Create/Update requests and never returned on responses.
  The "leave blank to keep existing" UX from Feature-115 continues to
  apply unchanged.
- **No DB or repository changes.** The schema already persists these
  columns (verified by existing repository round-trip tests at
  `internal/repository/implementation/sqlite/service_config_impl_test.go`).
- **Adapter comment refresh.** The previous comment at
  `cmd/cue/adapters/service_config.go:14-22` documented the gap as
  intentional; updated to state that wire DTOs now carry the full set
  of non-secret fields and only secrets remain write-only.
- **JSON additivity is non-breaking.** Adding optional fields to
  request/response JSON is backwards-compatible: existing clients that
  don't know about the new fields ignore them on read and send zero
  values on write (which preserves prior behaviour where those fields
  were already zero on the wire).

## API Changes

### Slack

`GET /api/v1/services/slack` and `GET /api/v1/services/slack/{id}` now
return `username`, `web_url`, and `poll_interval_seconds` on each
account.

`POST /api/v1/services/slack` and `PUT /api/v1/services/slack/{id}`
now accept the same three fields on the request body and persist them
to `repository.SlackAccount`.

### Email

`GET /api/v1/services/email` and `GET /api/v1/services/email/{id}` now
return `web_url` and `poll_interval_seconds`.

`POST /api/v1/services/email` and `PUT /api/v1/services/email/{id}`
accept the same.

### Calendar

`GET /api/v1/services/calendar` and `GET /api/v1/services/calendar/{id}`
now return `poll_interval_seconds`.

`POST /api/v1/services/calendar` and `PUT /api/v1/services/calendar/{id}`
accept the same.

The OpenAPI spec at `docs/api/openapi.yaml` was regenerated via
`just api-gen` and validated via `just api-lint`.

## Integration Points

- `internal/server/handler/service.go` — three response items, three
  request items, three `XToItem` helpers, six handler bodies (3 create,
  3 update) extended.
- `pkg/client/service_config.go` — `SlackAccount`, `EmailAccount`,
  `CalendarAccount` and the matching `Create*Request` types extended;
  `Update*Request` aliases inherit the new fields automatically.
- `cmd/cue/adapters/service_config.go` — `slackDTOToRepo`,
  `emailDTOToRepo`, `calendarDTOToRepo`, and three `Upsert*Account`
  request builders extended; the doc comment refreshed.
- `docs/api/openapi.yaml` — regenerated.

The settings UI form (`internal/ui/settings_view.go`) needs no
change: it already reads `WebURL` and `PollIntervalSeconds` from the
prefilled account record. The bug resolves itself once the API
round-trips.

## Error Handling

No new error paths. JSON decode of a request that omits the new fields
yields zero values (preserving prior behaviour); decode of a response
that omits them likewise yields zero values. The existing 400 (bad
JSON) and 404 (not found) paths continue to apply unchanged.

## Test Coverage

- Server handler tests extended at `internal/server/handler/service_test.go`
  for List/Create/Update Slack, List/Create Email, List/Create
  Calendar — assert the new fields are forwarded on request and
  emitted on response.
- SDK tests extended at `pkg/client/service_config_test.go` —
  `httptest`-backed assertions that the SDK marshals new fields on
  Create and unmarshals them on List/Create for all three account types.
- Adapter tests extended at `cmd/cue/adapters/service_config_test.go` —
  Slack and Email round-trip suites assert new fields survive
  Upsert→List, plus a new `TestCalendarRoundTripsPollInterval` covering
  the Calendar Create + List path that previously had no round-trip
  coverage.
- Repository tests at `internal/repository/implementation/sqlite/service_config_impl_test.go`
  already prove the DB layer persists `WebURL`, `PollIntervalSeconds`,
  and `Username` — no repo work needed and no test changes there.
- UI acceptance tests at `tests/ui/settings_acceptance_test.go` already
  assert prefill of `WebURL` and `PollIntervalSeconds` for the Email
  Edit case (Feature-115). They pass against the in-memory mock today;
  with this hotfix they pass end-to-end against a real `cue-server`.

## Verification

Manual smoke test against a running pair of `cue server` + `cue ui`:

1. Add an Email account with `WebURL=https://mail.example.com` and
   `Poll=600`.
2. Restart both processes against the same DB.
3. Open Settings → Email → Edit on that row.
4. Form must show `WebURL=https://mail.example.com` and `Poll=600`,
   not blanks. Saving without changing the password preserves the
   password (already covered by Feature-115).
5. Repeat for Slack (verify `Username`, `WebURL`, `Poll`) and Calendar
   (verify `Poll`).

## TDD Agent Stats

| Phase                | TDD Phase | Agent  | Notes                                                                                                              |
| -------------------- | --------- | ------ | ------------------------------------------------------------------------------------------------------------------ |
| Phase-9-Feature-115A | Red       | Inline | Failing handler tests for new field round-trip on Slack/Email/Calendar Create/Update/List.                          |
| Phase-9-Feature-115A | Green     | Inline | Extended response/request DTOs and conversion helpers; create/update handlers copy new fields into repository struct. |
| Phase-9-Feature-115A | Red       | Inline | Failing SDK tests; client.SlackAccount/EmailAccount/CalendarAccount and Create*Request didn't compile.              |
| Phase-9-Feature-115A | Green     | Inline | Mirrored new fields on SDK DTOs; Update*Request aliases inherit automatically.                                      |
| Phase-9-Feature-115A | Red       | Inline | Failing adapter round-trip tests covering Slack/Email/Calendar; new TestCalendarRoundTripsPollInterval suite added. |
| Phase-9-Feature-115A | Green     | Inline | Adapter helpers and Upsert builders forward the new fields; doc comment refreshed.                                  |
| Phase-9-Feature-115A | Refactor  | —      | No refactor needed; implementation arrived clean — three symmetric layers each replicating the same field set.       |
