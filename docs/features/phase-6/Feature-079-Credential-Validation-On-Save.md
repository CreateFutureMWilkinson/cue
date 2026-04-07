# Feature 079 — Credential and Calendar Validation on Save

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Enhancement |
| Severity | Medium |
| Status | Planned |
| Depends on | 065, 067, 068 |
| UI Tests | Yes |

## Problem

Adding a Calendar, Email, or Slack account saves credentials without validating them. Users can save invalid credentials (wrong password, expired token, unreachable server) and only discover the problem later when polling silently fails. The user should get immediate feedback on whether their credentials work.

## Required Changes

### Validation on Save

For each service type, attempt a lightweight connection test before persisting:

- **Slack**: Call `auth.test` API endpoint with the provided token. Verify the response includes a valid user/team.
- **Email (IMAP)**: Open a TLS connection to the IMAP server, attempt LOGIN, then LOGOUT. Verify authentication succeeds.
- **Calendar (ICS-over-HTTP)**: Fetch the calendar URL with any provided credentials. Verify a valid iCalendar response is returned.

### UI Behavior

- On save, show a brief "Validating..." indicator
- If validation fails, display the error on the form and **do not leave the form screen**
- If validation succeeds, save to DB and return to the account list
- Optionally add a "Validate" button that tests credentials without saving, if this simplifies the UX

### Error Display

- Show the actual error message from the service (e.g., "invalid_auth", "connection refused", "404 Not Found")
- Keep form fields populated so the user can correct and retry

## Acceptance Criteria

- Invalid Slack token: form stays open with error message, no DB record created
- Invalid IMAP credentials: form stays open with error message
- Invalid calendar URL: form stays open with error message
- Valid credentials: saved to DB, returns to account list
- Validation errors are human-readable

## UI Test Coverage

- UI acceptance test: attempt to save with invalid credentials, verify error shown and form remains
- UI acceptance test: save with valid (mocked) credentials, verify success and return to list
