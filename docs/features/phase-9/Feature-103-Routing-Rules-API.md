# Feature 103: Routing Rules API

**Phase:** Phase-9-Feature-103
**Status:** Planning
**Package:** `internal/server/handler/`

---

## Overview

Expose CRUD operations for routing rules — the deterministic rules that evaluate messages before Ollama scoring. Rules have priority ordering and regex-based pattern matching. This is a straightforward CRUD surface with one twist: rule ordering matters.

## Endpoints

### List Rules

```
GET /api/v1/rules
```

Returns all routing rules ordered by priority (ascending — lower number = higher priority).

**Response:**
```json
{
  "rules": [
    {
      "id": "uuid",
      "name": "Channel Join",
      "priority": 1,
      "source_pattern": ".*",
      "channel_pattern": ".*",
      "content_pattern": "",
      "message_type": "channel_join",
      "action": "notified",
      "importance_override": 9.0,
      "confidence_override": 1.0,
      "enabled": true,
      "created_at": "2026-04-01T00:00:00Z"
    },
    {
      "id": "uuid",
      "name": "Direct Mention",
      "priority": 2,
      "source_pattern": ".*",
      "channel_pattern": ".*",
      "content_pattern": "@username",
      "message_type": "",
      "action": "notified",
      "importance_override": 8.0,
      "confidence_override": 1.0,
      "enabled": true,
      "created_at": "2026-04-01T00:00:00Z"
    }
  ],
  "queue_depth": 15
}
```

`queue_depth` is included so UIs can display a warning if the Ollama queue is backing up (messages falling through to queue because no rule matched).

### Get Rule

```
GET /api/v1/rules/{id}
```

### Create Rule

```
POST /api/v1/rules
```

**Request:**
```json
{
  "name": "Deployment Alerts",
  "priority": 3,
  "source_pattern": "slack",
  "channel_pattern": "#deploy.*",
  "content_pattern": "(failed|error|rollback)",
  "message_type": "",
  "action": "notified",
  "importance_override": 8.5,
  "confidence_override": 1.0,
  "enabled": true
}
```

**Validation:**
- `name`: Required, non-empty
- `priority`: Required, positive integer
- `action`: Must be one of "notified", "ignored", "queue"
- `source_pattern`, `channel_pattern`, `content_pattern`: Must be valid Go regex if non-empty
- `importance_override`: 0.0-10.0
- `confidence_override`: 0.0-1.0

**Question: What happens when the new rule's priority conflicts with an existing rule?** Options:
- Reject with 409 (client must choose a unique priority)
- Auto-shift existing rules down to make room
- Allow duplicate priorities (evaluation order within same priority is undefined)

**Recommendation:** Allow duplicate priorities. The reorder endpoint handles explicit ordering. Auto-shifting is complex and surprising. Rejecting is unfriendly.

### Update Rule

```
PUT /api/v1/rules/{id}
```

Full replacement of rule fields. Same validation as create.

### Delete Rule

```
DELETE /api/v1/rules/{id}
```

**Question: Should deleting a built-in rule (channel_join, @mention) be allowed?** These are the core deterministic rules from the spec. Options:
- Prevent deletion (400 error for system rules)
- Allow deletion (user has full control)
- Allow disable but not delete

**Recommendation:** Allow deletion. The user should have full control over their routing rules. The defaults can be restored by recreating them. Document the default rules clearly so users can recreate them.

### Reorder Rule

```
POST /api/v1/rules/{id}/reorder
```

**Request:**
```json
{"priority": 5}
```

Updates the priority of a single rule. Does NOT shift other rules.

## Design Decisions to Make

### Regex Validation

**Question: How strictly should regex patterns be validated?**

- **Compile-time only**: Verify the pattern compiles with `regexp.Compile()`. Accept any valid Go regex.
- **Safety checks**: Also reject regexes that could cause catastrophic backtracking (nested quantifiers, etc.).
- **Dry-run**: Apply the regex against a sample message to verify it works as expected.

**Recommendation:** Compile-time validation only. Go's `regexp` package uses RE2, which guarantees linear-time matching — no catastrophic backtracking is possible. This is a non-issue.

### Rule Change Propagation

When a rule is created/updated/deleted, the in-memory `RulesEngine` needs to reload.

**Question: How does a rule change in the DB propagate to the running engine?**

- **Immediate reload**: Handler calls `rulesEngine.Reload()` after DB mutation.
- **Polling**: Engine polls DB every N seconds for changes. Simpler but delayed.
- **Event**: Handler sends event on a channel that the engine listens to.

**Recommendation:** Immediate reload. The `RulesPresenter` likely already triggers this. Same pattern for the API handler.

## Behaviors to Implement

1. **List rules handler** — Query all rules ordered by priority, include queue depth.
2. **Get rule handler** — By ID.
3. **Create rule handler** — Validate, save, trigger engine reload.
4. **Update rule handler** — Validate, save, trigger engine reload.
5. **Delete rule handler** — Delete, trigger engine reload.
6. **Reorder rule handler** — Update priority, trigger engine reload.
7. **Regex validation** — Shared validation function for pattern fields.

## Questions Summary

1. Priority conflict handling — reject, auto-shift, or allow duplicates?
2. Allow deletion of built-in/default rules?
3. How to propagate rule changes to the running engine?
4. Should there be a "test rule" endpoint that takes a sample message and shows if/how the rule would match?
5. Should rule history/audit be tracked (who changed what, when)?
