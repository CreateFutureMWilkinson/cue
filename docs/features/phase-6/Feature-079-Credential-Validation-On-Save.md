# Feature 079 — Credential and Calendar Validation on Save

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Enhancement |
| Severity | Medium |
| Status | Done |
| Depends on | 065, 067, 068 |
| UI Tests | Yes |

## Problem

Adding a Calendar, Email, or Slack account saves credentials without validating them. Users can save invalid credentials (wrong password, expired token, unreachable server) and only discover the problem later when polling silently fails. The user should get immediate feedback on whether their credentials work.

## Solution

### Validator Interfaces (presenter package)

Three minimal interfaces defined in `internal/ui/presenter/service_settings_presenter.go`:

```go
type SlackValidator interface {
    ValidateSlack(ctx context.Context, token string) error
}

type EmailValidator interface {
    ValidateEmail(ctx context.Context, host string, port int, username, password, encryption string) error
}

type CalendarValidator interface {
    ValidateCalendar(ctx context.Context, url string) error
}
```

### Functional Options Pattern

Validators are injected into `ServiceSettingsPresenter` via `ServiceSettingsOption` functional options, preserving backward compatibility with existing call sites:

```go
type ServiceSettingsOption func(*ServiceSettingsPresenter)

func WithSlackValidator(v SlackValidator) ServiceSettingsOption
func WithEmailValidator(v EmailValidator) ServiceSettingsOption
func WithCalendarValidator(v CalendarValidator) ServiceSettingsOption
```

When a validator is nil (not injected), the presenter skips validation and saves directly, maintaining the pre-Feature-079 behavior.

### Concrete Implementations (`internal/service/validation/`)

| Validator | Method | Protocol | Library |
|---|---|---|---|
| `SlackAPIValidator` | `auth.test` API | HTTPS POST | `net/http` |
| `IMAPValidator` | connect + login + logout | IMAP (SSL/TLS, STARTTLS, plain) | `emersion/go-imap/v2` |
| `ICSValidator` | fetch + parse | HTTP GET | `arran4/golang-ical` |

**SlackAPIValidator** sends a `POST` to `{baseURL}/auth.test` with a `Bearer` token header. Parses the JSON response and checks `ok` field. Configurable base URL via `WithSlackBaseURL` option for testing.

**IMAPValidator** is stateless and safe for concurrent use. Connects with the appropriate encryption mode (ssl_tls, starttls, or none), performs `LOGIN` + `LOGOUT`, and returns any error as a human-readable message. Input validation rejects empty host, username, or password before attempting connection. 10-second timeout on the dialer.

**ICSValidator** fetches the URL, checks for HTTP 200, and parses the body as iCalendar with a 1 MB `LimitReader` to prevent memory exhaustion. Configurable `http.Client` via `WithHTTPClient` option for testing.

### Presenter Integration

`SaveSlackAccount`, `SaveEmailAccount`, and `SaveCalendarAccount` now validate credentials before persisting:

1. Field validation (existing)
2. Credential validation (new) -- calls the injected validator if non-nil
3. Database insert (existing)
4. Watcher creation (existing)

Validation errors are wrapped with context (e.g., `"slack credential validation failed: invalid_auth"`).

### UI Indicator

Before calling the presenter's save method, each account form in `settings_view.go` sets the error label to `"Validating..."` and shows it. On success the label is hidden and the form navigates to the account list. On failure the error label is updated with the error message and the form stays open with all fields populated.

### Composition Root Wiring

In `cmd/cue/main.go`, the three validators are instantiated and injected:

```go
serviceSettingsPresenter := presenter.NewServiceSettingsPresenter(
    serviceConfigRepo, orch, watcherFactory,
    presenter.WithSlackValidator(validation.NewSlackAPIValidator()),
    presenter.WithEmailValidator(validation.NewIMAPValidator()),
    presenter.WithCalendarValidator(validation.NewICSValidator()),
)
```

## Design Decisions

1. **Functional options over constructor parameters** -- Adding three optional validator parameters to the existing `NewServiceSettingsPresenter` constructor would break all existing call sites. Functional options let callers opt in.

2. **Interfaces in presenter, implementations in validation** -- The presenter owns the contract (what it needs), and the validation package owns the how. This keeps the presenter testable with mocks and avoids importing network-dependent packages.

3. **Nil-safe skip** -- When no validator is injected, save proceeds without validation. This keeps tests simple and allows graceful degradation if validators cannot be constructed.

4. **Synchronous validation** -- Validation runs synchronously on the UI thread with a "Validating..." indicator. Given the short timeouts (10s) and the infrequent nature of account setup, async validation adds complexity without meaningful UX benefit.

5. **1 MB response limit for ICS** -- Prevents memory exhaustion from malicious or oversized calendar feeds while being large enough for any reasonable calendar.

## Error Handling

| Error | Source | User sees |
|---|---|---|
| Invalid token (`invalid_auth`) | Slack API | "slack credential validation failed: Slack authentication failed: invalid_auth" |
| Connection refused | Any | "...connection refused" |
| Login failed | IMAP server | "IMAP login: ..." |
| Non-200 HTTP | ICS URL | "calendar fetch failed: HTTP 404" |
| Invalid iCal format | ICS parser | "invalid iCalendar response: ..." |
| Validator not injected | Presenter | (no error -- skips validation) |

All errors are wrapped with `fmt.Errorf` context at each layer. The UI displays the full error string on the form's error label.

## Integration Points

- `internal/ui/presenter/service_settings_presenter.go` -- interfaces, options, validation calls in save methods
- `internal/service/validation/` -- three concrete validator implementations
- `internal/ui/settings_view.go` -- "Validating..." indicator before save
- `cmd/cue/main.go` -- validator instantiation and injection

## Test Coverage

### UI Acceptance Tests (6 new)

- Slack save with invalid token shows error, form remains
- Slack save with valid token succeeds
- Email save with invalid credentials shows error, form remains
- Email save with valid credentials succeeds
- Calendar save with invalid URL shows error, form remains
- Calendar save with valid URL succeeds

### Presenter Unit Tests (6 new)

- `TestSaveSlackAccount_ValidationFails` -- invalid token prevents save
- `TestSaveSlackAccount_ValidationPasses` -- valid token allows save
- `TestSaveEmailAccount_ValidationFails` -- invalid credentials prevent save
- `TestSaveEmailAccount_ValidationPasses` -- valid credentials allow save
- `TestSaveCalendarAccount_ValidationFails` -- invalid URL prevents save
- `TestSaveCalendarAccount_ValidationPasses` -- valid URL allows save

### Concrete Validator Tests (17 new)

**SlackValidatorSuite** (5 tests): invalid token, valid token, connection refused, non-JSON response, context handling

**EmailValidatorSuite** (7 tests): empty host/username/password, valid credentials (mock), SSL/TLS mode, STARTTLS mode, connection refused

**CalendarValidatorSuite** (5 tests): valid ICS, HTTP error, invalid iCal body, connection error, context handling

## TDD Agent Stats

| Impl Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Phase-6-Feature-079 | UI TESTS | Test Designer | ~30s | ~5,000 | e33a5c7 |
| Phase-6-Feature-079 (Slack) | RED+GREEN | Test Designer + Implementer | ~45s | ~37,000 | e42367b |
| Phase-6-Feature-079 (Email+Calendar) | RED+GREEN | Test Designer + Implementer | ~85s | ~44,000 | fb61779 |
| Phase-6-Feature-079 (concrete validators) | GREEN | Implementer (x3 parallel) | ~325s | ~89,000 | 1c0c8bf |
| Phase-6-Feature-079 (UI indicator) | GREEN | Implementer | ~20s | ~3,000 | 7fc2867 |
| Phase-6-Feature-079 (wiring) | GREEN | Implementer | ~10s | ~2,000 | 201b609 |
