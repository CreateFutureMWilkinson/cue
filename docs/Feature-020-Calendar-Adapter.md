# Feature 020: Calendar Adapter

**Phase:** Phase-2-Feature-020
**Status:** Planned
**Packages:** `internal/service/calendar/`

---

## Overview

Provider-agnostic calendar integration for reading external calendar events. The MVP fetches events from Google Calendar secret (share) links via ICS-over-HTTP. The adapter interface is designed for extensibility — future implementations can support CalDAV, Microsoft Graph, or direct Google Calendar API. Read-only; Cue never writes back to the calendar. Events feed into the day planner (Feature 021) as fixed time blocks that the Pomodoro schedule wraps around.

## Design Decisions

- **Provider interface with adapter pattern** — `CalendarProvider` is a narrow interface (`FetchEvents`). Each calendar source implements it independently. The planner depends only on the interface, not any concrete provider.
- **ICS-over-HTTP as MVP implementation** — Google Calendar's "secret address in iCal format" is the simplest integration requiring no OAuth, no API keys, and no account linking. Just a URL in config.
- **Pure Go ICS parsing** — use a Go ICS library (e.g., `github.com/arran4/golang-ical`) to parse the iCalendar feed. No CGO.
- **Date-filtered fetch** — `FetchEvents` accepts a target date and returns only events for that day. The ICS feed may contain weeks/months of data; filtering happens client-side after parse.
- **All-day events included** — all-day events are returned with start/end spanning the full workday window. The planner decides how to handle them (likely treated as blocking the entire day or ignored based on title).
- **Caching with TTL** — calendar data is fetched at most once per planning session. No background polling for MVP. The planner triggers a fetch when the user starts the planning wizard.
- **Config section `[calendar]`** — URL, enabled flag, timeout. No poll interval (on-demand only for MVP).

## Data Model

### CalendarEvent

```go
type CalendarEvent struct {
    ID      string     // source-native event ID (UID from ICS)
    Title   string
    Start   time.Time
    End     time.Time
    AllDay  bool
}
```

## API

### CalendarProvider Interface

```go
type CalendarProvider interface {
    FetchEvents(ctx context.Context, date time.Time) ([]CalendarEvent, error)
}
```

### ICS Adapter Constructor

```go
func NewICSProvider(url string, httpClient HTTPClient, timeout time.Duration) (*ICSProvider, error)
```

### HTTPClient Interface (for testability)

```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

## Config

```toml
[calendar]
enabled = false
ics_url = ""              # Google Calendar secret link
timeout_seconds = 30
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Calendar disabled in config | Provider not created; planner skips calendar integration |
| ICS URL empty or invalid | Validation error at config load time |
| HTTP fetch fails (network) | Return wrapped error; planner proceeds without calendar (empty event list, log warning) |
| HTTP timeout | Return wrapped context deadline error; same fallback as network failure |
| ICS parse error (malformed) | Return wrapped parse error; planner proceeds without calendar |
| No events for target date | Return empty slice (not an error) |
| All-day event detection | Events with `VALUE=DATE` (no time component) flagged as `AllDay=true` |

## Integration Points

- **Day Planner (Feature 021):** Consumes `[]CalendarEvent` as fixed time blocks. Meetings are non-negotiable in the schedule; Pomodoros fill remaining gaps.
- **Config (Feature 001):** New `[calendar]` section with validation rules.
- **Planner UI (Feature 022):** Calendar events displayed in schedule view as meeting blocks.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `calendar` | `ICSProviderSuite` | Fetch and parse valid ICS, filter by date, all-day events, empty feed, HTTP error, parse error, timeout, constructor validation |
| `calendar` | `CalendarEventSuite` | Event field population, all-day detection, multi-day event handling |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| ICS Provider | RED | Test Designer | — | — | — |
| ICS Provider | GREEN | Implementer | — | — | — |
| ICS Provider | REFACTOR | Refactorer | — | — | — |
