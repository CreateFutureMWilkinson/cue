# Feature 080 — Default Poll Intervals Per Service Type

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | Low |
| Status | Planned |
| Depends on | — |
| UI Tests | No |

## Problem

The system has a single global poll interval default of 600 seconds. There are no per-service-type defaults. When adding accounts via the UI, users must manually enter a poll interval every time. The form should pre-populate with sensible defaults that differ by service type.

## Current State

- Global orchestrator default: 600s (config.go:119)
- Per-account `PollIntervalSeconds` field exists in DB records but has no default
- UI forms require manual entry with no pre-populated value

## Required Defaults

| Service | Default Poll Interval | Rationale |
|---|---|---|
| Calendar | 600 seconds (10 min) | Calendar events change infrequently |
| Email | 600 seconds (10 min) | Matches current global default |
| Slack | 60 seconds (1 min) | Slack messages are time-sensitive; faster polling needed |

## Required Changes

1. Pre-populate the poll interval field in each "Add Account" form with the service-specific default
2. If the user clears the field and saves, fall back to the default for that service type
3. Document the defaults in the example config TOML comments

## Acceptance Criteria

- Slack "Add Account" form shows 60 in the poll interval field by default
- Email "Add Account" form shows 600 in the poll interval field by default
- Calendar "Add Account" form shows 600 in the poll interval field by default
- Saving without modifying the interval uses the pre-populated default
