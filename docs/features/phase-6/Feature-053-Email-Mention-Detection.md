# Feature 053: Email Mention Detection Fix

**Phase:** Phase-6-Feature-053
**Type:** Bugfix
**Severity:** Critical
**Status:** Planned
**Packages:** `internal/service/decisionengine/`
**Related:** Feature 003 (Deterministic Routing), Feature 006 (Email Watcher)

---

## Bug Description

Email mention detection never triggers the IS=8 deterministic boost. The email watcher correctly detects when the user is in To/CC/BCC and sets `MessageType: "mention"`, but the router's `@mention` rule only searches for `@username` patterns in message **content**. Emails don't contain `@username` strings, so the deterministic rule never fires for email — all email mentions fall through to LLM scoring instead of receiving the guaranteed NOTIFIED path.

## Expected Behavior

When an email has the user in To/CC/BCC, the router should assign IS=8, CS=1.0, status=NOTIFIED — identical to a Slack `@mention`.

## Actual Behavior

Email mentions are scored by Ollama like any other message. The `MessageType: "mention"` field set by the email watcher is ignored by the router.

## Root Cause

`router.go:135-145` — The `@mention` deterministic rule checks `strings.Contains(msg.RawContent, "@"+username)` only. It does not check `msg.MessageType == "mention"`, which is how the email watcher signals a direct mention.

## Proposed Fix

Extend the `@mention` deterministic rule to also match when `msg.MessageType == "mention"`. This makes the rule source-agnostic — both Slack `@username` in content and email To/CC/BCC matches trigger the same IS=8 path.

```go
// Before: only checks content
if strings.Contains(msg.RawContent, "@"+username) { ... }

// After: checks content OR message type
if strings.Contains(msg.RawContent, "@"+username) || msg.MessageType == "mention" { ... }
```

## Test Strategy

- RED: Test that a message with `MessageType: "mention"` and no `@username` in content receives IS=8, CS=1.0, NOTIFIED
- GREEN: Extend the deterministic rule check
- REFACTOR: Extract mention detection into a helper if warranted
