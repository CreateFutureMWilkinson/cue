# Feature 101: Day Planner API

**Phase:** Phase-9-Feature-101
**Status:** Planning
**Package:** `internal/server/handler/`
**Depends on:** 097, 101A

---

## Overview

Expose day planner schedule generation and management over REST. The planner is purely about **time management** — it reads the user's calendar, generates two schedule options of focus/break blocks around meetings, and lets the user save one. Tasks are managed independently via Feature 101A and are not referenced by the planner.

Schedules are stored by date. Only today's schedule is considered "active" (eligible for timer progression). Future schedules can be created and stored but are inert until their date arrives.

## Design Decisions

1. **No state machine** — the old `PlannerPresenter` wizard (6-step state machine) is replaced by two stateless operations: generate and save. No sessions, no server-side wizard state.
2. **No tasks in schedules** — blocks are typed as `focus`, `short_break`, `long_break`, or `meeting`. What the user does during a focus block is a UI concern.
3. **Date-keyed storage** — schedules are addressed by ISO date (`/api/v1/planner/{date}`). `/api/v1/planner/active` is an alias for today's date.
4. **PUT upsert** — saving a schedule overwrites any existing plan for that date. No 409 conflict — PUT is idempotent.
5. **Future schedules** — can be stored but not progressed. Only today's plan drives timers.
6. **Stateless generation** — generate reads calendar and config, returns two options. Nothing stored until the client PUTs one.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/planner/generate` | Generate two schedule options |
| GET | `/api/v1/planner/{date}` | Get schedule for a date |
| PUT | `/api/v1/planner/{date}` | Save/overwrite schedule for a date |
| DELETE | `/api/v1/planner/{date}` | Delete schedule for a date |
| GET | `/api/v1/planner/active` | Alias for GET `/api/v1/planner/{today}` |
| DELETE | `/api/v1/planner/active` | Alias for DELETE `/api/v1/planner/{today}` |

### Generate Schedules

```
POST /api/v1/planner/generate
Content-Type: application/json

{"date": "2026-04-20"}
```

**Request:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `date` | string (ISO date) | No | Target date. Defaults to today or next working day (via `Planner.TargetDate()` logic). |

Server fetches calendar events for the target date, then generates two schedule options using the existing `Planner.GenerateSchedules()` with empty task list.

**Response:**
```json
{
  "date": "2026-04-20",
  "options": [
    {
      "strategy": "focus-maximized",
      "total_focus_minutes": 225,
      "break_count": 5,
      "blocks": [
        {
          "start": "09:00",
          "end": "09:25",
          "type": "focus"
        },
        {
          "start": "09:25",
          "end": "09:30",
          "type": "short_break"
        },
        {
          "start": "10:00",
          "end": "11:00",
          "type": "meeting"
        }
      ]
    },
    {
      "strategy": "recovery-balanced",
      "total_focus_minutes": 180,
      "break_count": 8,
      "blocks": [...]
    }
  ]
}
```

### Get Schedule

```
GET /api/v1/planner/2026-04-20
GET /api/v1/planner/active
```

Returns the saved schedule for the given date. `/active` resolves to today's date.

**Response:**
```json
{
  "date": "2026-04-20",
  "strategy": "focus-maximized",
  "blocks": [
    {
      "start": "09:00",
      "end": "09:25",
      "type": "focus"
    }
  ],
  "created_at": "2026-04-20T08:30:00Z"
}
```

**404** if no schedule exists for the date.

### Save Schedule

```
PUT /api/v1/planner/2026-04-20
Content-Type: application/json

{
  "strategy": "focus-maximized",
  "blocks": [
    {
      "start": "09:00",
      "end": "09:25",
      "type": "focus"
    },
    {
      "start": "09:25",
      "end": "09:30",
      "type": "short_break"
    }
  ]
}
```

Upserts the schedule for the given date. Overwrites any existing schedule. The client sends the full schedule payload (typically one of the two options from generate, but the API does not enforce this).

**Response:** 200 OK with the saved schedule (same shape as GET response). 400 if blocks are malformed.

### Delete Schedule

```
DELETE /api/v1/planner/2026-04-20
DELETE /api/v1/planner/active
```

Deletes the schedule for the given date. Returns 204 No Content. 404 if no schedule exists.

## Block Types

| Type | Description |
|------|-------------|
| `focus` | Focus/work time block |
| `short_break` | Short break (default 5 min) |
| `long_break` | Long break (default 20 min) |
| `meeting` | Calendar event (from CalendarProvider) |

Blocks carry no task references. The UI decides what to display in focus blocks.

## Active Plan Semantics

- **Active** means today's date has a saved schedule.
- `/active` is a routing alias — it resolves to today's date and delegates to the same handler.
- Timer progression, current block tracking, etc. are separate concerns (Feature 104: Timer API).
- Future schedules (date > today) are stored but have no active behavior.

## Changes to Existing Code

### Schedule Generator

The existing `Planner.GenerateSchedules()` requires a `[]TaskEstimate` parameter for assigning tasks to focus blocks. Since the planner no longer assigns tasks, this will be called with an empty task list. Focus blocks will have no `TaskID` or `TaskName` — they are anonymous time slots.

This may require adjusting the generator to handle empty task lists gracefully (generate focus blocks without task assignment).

### Schedule Repository

The existing `ScheduleRepository` uses a unique constraint on date, which aligns with the PUT upsert semantics. The `Save` method may need to become an upsert (INSERT OR REPLACE) if it isn't already.

### PlannerPresenter

The GUI presenter's state machine is not modified by this feature. It will be revisited in Feature 107 (Fyne Client Re-wire) when the UI is rebuilt against the API.

## Behaviors to Implement

1. **Generate handler** — parse optional date, fetch calendar, generate two schedules, return options
2. **Get schedule handler** — load by date, 404 if missing
3. **Put schedule handler** — validate blocks, upsert by date
4. **Delete schedule handler** — delete by date, 404 if missing
5. **Active alias routing** — resolve `/active` to today's date, delegate to date handlers
6. **Generator adaptation** — handle empty task list in schedule generation

## Testing Considerations

- Generate: with and without date param, default date logic, calendar event inclusion
- GET: existing schedule, missing schedule (404), active alias resolves to today
- PUT: new schedule, overwrite existing, malformed blocks (400)
- DELETE: existing schedule, missing schedule (404), active alias
- Active alias: verify `/active` and `/planner/{today}` return identical results
- Generator: empty task list produces valid focus/break blocks without task references
