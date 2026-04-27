# Feature 101: Day Planner API

**Phase:** Phase-9-Feature-101
**Status:** Planning
**Package:** `internal/server/handler/`

---

## Overview

Expose the day planner workflow over REST. This is the most complex API surface because the planner is a multi-step wizard with state transitions. The API must support: listing available tasks, starting a planning session, estimating pomodoros, generating schedule options, confirming a plan, and tracking progress through the active schedule.

## The State Machine Problem

The GUI planner uses `PlannerPresenter` which manages a state machine:

```
StepIdle → StepTaskSelect → StepEstimates → StepPriority → StepSchedule → StepActive
```

Each step depends on the previous step's output. The GUI presenter holds this state in memory.

**Question: How should the API handle stateful wizard flow?**

### Option A: Session-Based State (Server-Side)

Server creates a planning session with an ID. Each step mutates server-side state.

```
POST /api/v1/planner/sessions → {"session_id": "abc"}
POST /api/v1/planner/sessions/abc/select-tasks {task_ids: [...]}
POST /api/v1/planner/sessions/abc/next → returns estimates
POST /api/v1/planner/sessions/abc/override-estimate {task_id, pomos}
POST /api/v1/planner/sessions/abc/next → returns schedule previews
POST /api/v1/planner/sessions/abc/confirm {strategy: "focus-maximized"}
```

- Pro: Mirrors the GUI flow exactly. Easy to understand.
- Con: Server holds session state. Need session timeout/cleanup. What if client disconnects mid-wizard?

### Option B: Stateless Steps (Client-Side State)

Each endpoint is independent. Client passes accumulated state with each request.

```
GET  /api/v1/planner/tasks → available tasks
POST /api/v1/planner/estimate {task_ids: [...]} → estimates for selected tasks
POST /api/v1/planner/generate {tasks: [{id, pomos}...], date: "2026-04-10"} → two schedule options
POST /api/v1/planner/confirm {schedule: {strategy, blocks: [...]}} → saves plan
```

- Pro: Truly stateless. No session management. Each call is self-contained.
- Con: Client must manage and re-send state. Generate endpoint receives potentially large payload. No server-side validation that the wizard was followed in order.

### Option C: Hybrid — Persistent Draft

```
POST /api/v1/planner/draft → creates draft plan (persisted in DB)
PATCH /api/v1/planner/draft/tasks {task_ids: [...]}
POST /api/v1/planner/draft/estimate → runs Ollama estimation, stores results in draft
PATCH /api/v1/planner/draft/estimates/{task_id} {override_pomos: 3}
POST /api/v1/planner/draft/generate → generates schedule options, stores in draft
POST /api/v1/planner/draft/confirm {strategy: "focus-maximized"} → promotes draft to active plan
DELETE /api/v1/planner/draft → abandon
```

- Pro: State is persisted (survives crashes). Each step is a mutation on a known entity. Client can resume a partially-completed wizard.
- Con: Most complex to implement. Needs draft table or reuse of schedule table with "draft" status.

**Recommendation:** Option A (session-based) for v1. The planning wizard is inherently stateful — fighting that with Option B creates a worse API. Session timeout of 30 minutes handles abandoned wizards. Only one session at a time per server (single-user app). Option C is better long-term but over-engineered for v1.

## Endpoints

### Active Plan

```
GET /api/v1/planner/active
```

Returns the current active schedule if one exists, including current block and progress.

**Response:**
```json
{
  "active": true,
  "schedule": {
    "id": "uuid",
    "date": "2026-04-10",
    "strategy": "focus-maximized",
    "blocks": [
      {
        "start": "09:00",
        "end": "09:45",
        "type": "focus",
        "task_id": "uuid",
        "task_name": "Review PR #42",
        "status": "completed"
      },
      {
        "start": "09:45",
        "end": "09:55",
        "type": "short_break",
        "status": "current"
      }
    ],
    "current_block_index": 1,
    "total_focus_time_minutes": 180,
    "elapsed_focus_time_minutes": 45
  }
}
```

If no active plan: `{"active": false}`.

### Complete Current Task

```
POST /api/v1/planner/active/complete-task
```

Marks the current focus block as complete and advances to the next block. Returns updated schedule state.

### Abandon Plan

```
DELETE /api/v1/planner/active
```

Abandons the active plan. Returns 200. Idempotent — returns 200 even if no active plan.

### Available Tasks

```
GET /api/v1/planner/tasks
```

Returns incomplete todos available for planning. Includes categories.

**Response:**
```json
{
  "tasks": [
    {
      "id": "uuid",
      "title": "Review PR #42",
      "priority": 2,
      "due_date": "2026-04-11",
      "categories": ["code-review", "team"]
    }
  ]
}
```

### Start Planning Session

```
POST /api/v1/planner/sessions
```

Creates a new planning session. Fails with 409 if a session already exists or an active plan is running.

**Response:**
```json
{
  "session_id": "uuid",
  "step": "task_select",
  "tasks": [...same as GET /tasks...]
}
```

### Select Tasks

```
POST /api/v1/planner/sessions/{id}/select-tasks
```

**Request:**
```json
{
  "task_ids": ["uuid1", "uuid2", "uuid3"]
}
```

### Get Estimates

```
POST /api/v1/planner/sessions/{id}/estimate
```

Triggers Ollama estimation for selected tasks. This may take several seconds.

**Question: Should estimation be synchronous or async?**

- **Synchronous**: Client waits for response. Simple but could take 10-30s for many tasks.
- **Async**: Server returns 202 Accepted, client polls for completion.

**Recommendation:** Synchronous for v1. The planner currently estimates synchronously in the GUI. Set a generous HTTP timeout (60s). If this proves too slow for many tasks, add async later.

**Response:**
```json
{
  "session_id": "uuid",
  "step": "estimates",
  "estimates": [
    {
      "task_id": "uuid1",
      "title": "Review PR #42",
      "estimated_pomos": 2,
      "user_override": null,
      "effective_pomos": 2
    }
  ],
  "summary": {
    "total_pomos": 8,
    "available_blocks": 10,
    "overloaded": false
  }
}
```

### Override Estimate

```
PATCH /api/v1/planner/sessions/{id}/estimates/{task_id}
```

**Request:**
```json
{"override_pomos": 3}
```

### Generate Schedules

```
POST /api/v1/planner/sessions/{id}/generate
```

Generates two schedule options (focus-maximized and recovery-balanced). Includes calendar events if configured.

**Response:**
```json
{
  "session_id": "uuid",
  "step": "schedule",
  "options": [
    {
      "strategy": "focus-maximized",
      "total_focus_minutes": 225,
      "break_count": 5,
      "blocks": [...]
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

### Confirm Schedule

```
POST /api/v1/planner/sessions/{id}/confirm
```

**Request:**
```json
{"strategy": "focus-maximized"}
```

Saves the schedule, destroys the session, transitions to active plan.

### Abandon Session

```
DELETE /api/v1/planner/sessions/{id}
```

Discards the planning session without saving.

## Design Decisions to Make

### Todo CRUD

**Question: Should the planner API include todo management, or is that a separate feature?**

The planner reads from `TodoRepository` but doesn't create/edit/delete todos — the GUI has a separate todo management interface. Options:
- Include basic todo CRUD here (since planning needs it)
- Separate Feature 106: Todo API

**Recommendation:** Separate feature. The planner API is already the most complex one. Keep it focused on the planning workflow.

### Calendar Event Inclusion

The planner uses `CalendarProvider.FetchEvents()` to include calendar blocks in schedule generation. This is transparent to the API — events are included in the generated schedule blocks.

**Question: Should there be an endpoint to preview calendar events for the day?** This would help UIs show "here's what's already on your calendar" before the user commits to a plan.

## Behaviors to Implement

1. **Get active plan handler** — Return current schedule state or empty.
2. **Complete task handler** — Advance the active schedule.
3. **Abandon plan handler** — Clear active schedule.
4. **List available tasks handler** — Query incomplete todos.
5. **Create session handler** — Initialize planning session, return task list.
6. **Select tasks handler** — Store task selection in session.
7. **Estimate handler** — Run Ollama estimation, return estimates.
8. **Override estimate handler** — Update user override for a task.
9. **Generate schedules handler** — Run schedule generation, return options.
10. **Confirm handler** — Save schedule, destroy session.
11. **Abandon session handler** — Clean up session.
12. **Session timeout** — Background goroutine that cleans up expired sessions.

## Testing Considerations

- Full wizard flow: Create session → select tasks → estimate → generate → confirm → verify active plan.
- Session expiry: Create session, wait, verify it's cleaned up.
- Concurrent session rejection: Two clients try to create sessions simultaneously.
- Estimation timeout: Mock slow Ollama, verify client gets a response (or timeout error).
- Active plan operations without an active plan: Verify appropriate error responses.

## Questions Summary

1. Session-based, stateless, or persistent draft for the wizard flow?
2. Synchronous or async estimation?
3. Should todo CRUD be part of this feature or separate?
4. Calendar event preview endpoint?
5. Session timeout duration?
6. What happens if the server restarts mid-session? (Sessions lost — acceptable for v1?)
7. Should the API support resuming a partially-completed wizard?
