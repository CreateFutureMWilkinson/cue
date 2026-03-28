# CLAUDE.md — Cue Project Instructions

This file is the authoritative instruction set for AI coding agents (Claude Code) when working in this repository. It supersedes any conflicting inline comments or ad-hoc instructions given during a session.

---

## 1. Project Overview

**Cue** is a local-first, privacy-centric ADHD-friendly productivity assistant written in Go. It monitors Slack and Email for high-stakes messages (deadlines, outages, server incidents, channel joins, @mentions), evaluates them using local Ollama inference, and routes them based on importance (0–10) and confidence (0.0–1.0) scores. A web-based GUI displays notifications and a feedback buffer. No data leaves the machine; no hosted LLM provider is ever used.

**User Profile:** ADHD-sufferer juggling dozens of Slack channels and hundreds of emails. Needs to catch critical messages (missed deadlines, outages, new channel assignments, direct mentions) without drowning in noise. Acceptable false positive rate: 20–30%.

Module path: `github.com/CreateFutureMWilkinson/cue`
Go version: `1.26.1`

---

## 2. Technology Stack (Hard Constraints)

| Concern | Choice | Notes |
|---|---|---|
| Language | Go 1.26+ | Native, cross-platform binaries |
| Config | TOML | `~/.cue/config.toml` with safe defaults |
| Relational DB | `modernc.org/sqlite` | Pure Go. **Never** use CGO drivers (`mattn/go-sqlite3`) |
| Vector DB | `github.com/rengensheng/chromem-go` | Flat-file, pure Go, BERT embeddings |
| LLM Inference | Ollama API (local only) | No hosted LLM substitutions ever |
| GUI | Fyne | Web-based layout; bare-bones Phase 1 |
| Audio | `github.com/gen2brain/beeep` | Cross-platform OS alerts |
| CLI | `github.com/urfave/cli/v3` | Config/auth commands |
| Testing | `github.com/stretchr/testify` + stdlib | Suite-based TDD required |

---

## 3. Repository Layout

```
cmd/cue/
  main.go                           # Composition root, entry point

internal/
  config/                           # Configuration loading, validation
    config.go
    config_test.go
  
  repository/                       # Repository interfaces and adapters
    message.go (interface)
    implementation/sqlite/
      message_impl.go
      message_impl_test.go
  
  service/
    decisionengine/                 # Routing, scoring, orchestration
      router.go
      router_test.go
      ollama_client.go
      ollama_client_test.go
    
    buffer/                         # Feedback buffer management
      buffer.go
      buffer_test.go
    
    vector/                         # Vector store integration
      vector.go
      vector_test.go
    
    watcher/                        # Slack + Email polling
      slack.go
      slack_test.go
      email.go
      email_test.go

  ui/                               # Fyne GUI
    window.go
    notification_pane.go
    activity_log.go
    feedback_review.go

  alert/                            # Audio notifications
    audio.go
    audio_test.go

docs/                               # Architecture, design docs
  DESIGN.md
  DATA_MODEL.md

tests/                              # Integration tests
  integration_test.go

_build/                             # Build artifacts (not committed)
scripts/                            # Build scripts
.claude/
  agents/
    test-designer.md
    implementer.md
    refactorer.md
```

---

## 4. Core Routing Logic

### Deterministic Rules (Applied First)

| Rule | Importance Score | Condition |
|---|---|---|
| User added to new channel | 9 | Message type = channel_join |
| Direct @mention of user | 8 | Message content contains @username |
| All other messages | LLM-scored | No deterministic match |

**Key:** Importance is NEVER determined by sender identity. Remove any rules based on "who it is from."

### Ollama Scoring

Ollama evaluates message content, sender, channel, and thread context. Returns:
- `importance_score` (0–10 float)
- `confidence_score` (0.0–1.0 float)
- `reasoning` (string explanation)

### Routing Decision

```
if importance_score >= 7 AND confidence_score >= 0.8:
    status = NOTIFIED
    action: audio alert, display in GUI notification pane

else if importance_score >= 7 AND confidence_score < 0.8:
    status = BUFFERED
    action: silent queue, user reviews in feedback buffer

else:
    status = IGNORED
    action: log to database, available for manual review
```

**Thresholds are configurable in config.toml.**

### Fallback Behavior

If Ollama times out, returns invalid JSON, or fails:
```
importance_score = 7
confidence_score = 0.0
→ BUFFERED (safe default: user reviews manually)
```

---

## 5. Data Model

### Message Event (SQLite)

```go
type Message struct {
    ID               uuid.UUID     // Generated at evaluation time
    Source           string        // "email" | "slack"
    SourceAccount    string        // Email account ID | Slack workspace ID
    Channel          string        // Email folder / Slack channel name
    Sender           string        // Email address / Slack user ID
    MessageID        string        // Source-native message ID for linking
    RawContent       string        // Full message body
    ImportanceScore  float64       // 0–10, set by router
    ConfidenceScore  float64       // 0.0–1.0, set by router
    Status           string        // "Pending", "Notified", "Buffered", "Ignored", "Resolved"
    Reasoning        string        // LLM explanation
    UserRating       *int          // 0–10, set during feedback review [nullable]
    UserFeedback     *string       // Free-text notes [nullable]
    VectorID         *uuid.UUID    // Reference to BERT embedding [nullable]
    CreatedAt        time.Time
    UpdatedAt        time.Time
    ResolvedAt       *time.Time
}
```

### Todo Item (Phase 2)

```go
type Todo struct {
    ID        uuid.UUID
    Title     string
    Priority  int
    CreatedAt time.Time
    CompletedAt *time.Time
}
```

---

## 6. Configuration (config.toml)

Location: `~/.cue/config.toml` (auto-created with defaults on first run)

```toml
[database]
path = "~/.cue/messages.db"

[slack]
enabled = true
bot_token = "xoxb-..."
workspace_id = "T..."
poll_interval_seconds = 600

[email]
enabled = true
imap_host = "imap.gmail.com"
imap_port = 993
username = "user@gmail.com"
password_env = "CUE_EMAIL_PASSWORD"
poll_interval_seconds = 600

[orchestrator.router]
importance_threshold = 7
confidence_threshold = 0.8
buffer_size_per_source = 100

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_enabled = true
batch_process = true

[gui]
host = "localhost"
port = 8080

[logging]
log_level = "info"
log_dir = ""
```

**ONLY TOML controls configuration.** No hardcoded values, no CLI flags for feature toggles.

---

## 7. Notification Queue UI (Phase 1)

### Main Window (Fyne)

**Layout: Three panes**

1. **Notification Queue** (left, scrollable)
   - Columns: [Source (15ch)] | [Sender (15ch)] | [Channel (15ch)] | [Message preview]
   - Rows: one per NOTIFIED message (newest first)
   - Each row: sender, channel, source are truncated independently to 15 chars
   - Click row to expand: show full message, IS, CS, timestamp
   - Action: "Resolve" button (marks as Resolved, optionally rate)

2. **Activity Log** (right, real-time)
   - Live feed of system events
   - "Slack: fetched 12 messages"
   - "Routed 8 NOTIFIED, 3 BUFFERED, 1 IGNORED"
   - "Ollama: inference took 250ms"
   - "Email: connection error, retrying..."
   - Errors in red; info in neutral color

3. **Feedback Buffer Review** (bottom, button-triggered)
   - Modal/tab to review buffered messages (oldest → newest)
   - Display: sender, channel, source, message content, current IS, CS
   - User rating: **0–10 buttons** (not slider)
   - Optional notes textarea
   - Actions: "Save Rating", "Skip", "Delete"
   - Counter: "3 of 47 buffered messages reviewed"

### Menu

- **Settings:** Edit config.toml, reconnect sources, change thresholds
- **About:** Version, links
- **Quit:** Graceful shutdown

### No Visual Decorations

Phase 1: Functional, plain, no animations or character. Focus on feedback loop accuracy.

---

## 8. Audio Alerts

Use `beeep` for notifications:

- **Notification triggered (NOTIFIED status):** Sharp ping + 2-second mute to prevent spam
- **System startup:** Subtle chime
- **Errors:** Low-priority background errors do NOT alert; only show in activity log
- **Shutdown:** Fade-out tone

Configurable on/off in `config.toml`.

---

## 9. Slack Integration

**Configuration:** Bot token + workspace ID

**Behavior:**
- Poll channels list every 10 minutes (batch process)
- Detect new channels user has joined (emit IS=9)
- For each message: capture sender, channel name, thread context, raw text
- No message sending (read-only)
- Graceful error handling: log, backoff, retry

**Thread Context:**
- If message is a reply in thread, include thread parent message for context

---

## 10. Email Integration

**Configuration:** IMAP host, port, username, password (via environment variable)

**Behavior:**
- Poll inbox every 10 minutes (batch process)
- Detect new messages
- Extract: sender, subject, folder, body (text only)
- Detect @mentions: scan for user's email address in To/CC/BCC
- No sending (read-only)
- Graceful error handling: log, backoff, retry

---

## 11. Batch Processing (Critical)

Message fetching and routing happen in **batches every 10 minutes:**

```
Slack: fetch 10 messages → [Batch] → Route all → Store all
Email: fetch 10 messages → [Batch] → Route all → Store all
```

- Single thread per source (Slack, Email)
- Batch as a unit: fetch all, then route all, then store all
- Any slow message (Ollama timeout) counted in p95 latency for entire batch
- <500ms p95 for routing decision (message received → IS/CS/status)

---

## 12. Error Handling Strategy

| Error | Severity | Action |
|---|---|---|
| Ollama timeout (>10s) | Medium | IS=7, CS=0.0 (BUFFERED), log, retry next batch |
| Slack API rate limit | Low | Backoff 60s, retry silently |
| Email IMAP connection lost | High | Log, reconnect, show in activity log, do NOT crash |
| Invalid message format | Low | Log, skip message, continue processing |
| chromem-go embedding failure | Medium | Store message without vector, log, continue |
| Config file missing | Critical | Create with defaults on startup |
| GUI port already bound | Critical | Error message, exit gracefully |
| Database locked (WAL) | Low | Retry with backoff, log |

**Philosophy:** User only sees errors that require action. Background errors log and retry silently.

---

## 13. Engineering Process: Red-Green-Refactor TDD

Red-Green-Refactor is **required** for all feature and bug-fix work.

### Agent Teams (Required for ALL Changes)

**All features and bug fixes MUST use Agent Teams** to enforce context isolation. No exceptions — even simple changes (config, utility functions) go through the full pipeline.

```
Feature Requirement (from docs/DESIGN.md)
        ↓
   Test Designer Agent (RED)
   - Reads requirement, writes failing test
   - Does NOT access implementation files
   - Confirms test fails: go test ./...
        ↓
   Implementer Agent (GREEN)
   - Reads failing test as specification
   - No knowledge of test writer's assumptions
   - Writes minimal code to pass tests
   - Confirms all tests green: go test ./...
        ↓
   Refactorer Agent (REFACTOR)
   - Reads passing tests + implementation
   - Improves code quality
   - Keeps all tests GREEN
        ↓
   Commit checkpoints (one per phase)
```

### Invoking Agent Teams

Enable in `.claude/settings.json`:
```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  }
}
```

Then use natural language:
```
Create an agent team to implement [feature] with TDD:

1. Test Designer: Write failing tests for [requirement]
   - [specific test case]
   - [specific test case]
   - Confirm: go test -run TestName -v ./path/to/pkg

2. Implementer: Write minimal code to pass those tests
   - Only implement what tests require
   - Confirm: go test ./...

3. Refactorer: Improve code while keeping tests green
   - Confirm: go test ./...
```

### Commit Checkpoints

Create one commit per phase. **Run `just fmt` as the last step before every commit** (red, green, and refactor):

| Phase | Scope | Message |
|---|---|---|
| Red | Failing test(s) only | `test(scope): failing test for ...` |
| Green | Minimal implementation | `feat(scope): implement ... [tests pass]` |
| Refactor | Cleanup; tests remain green | `refactor(scope): improve ...` |

Keep each commit **tightly scoped to one phase**. Do not mix phases.

---

## 14. Go Conventions

- **Constructors** — use dependency injection; validate all dependencies
- **Interfaces** — keep minimal and consumer-focused
- **Globals/singletons** — avoid entirely
- **Context** — pass `context.Context` as first argument to all blocking/external operations
- **Error wrapping** — `fmt.Errorf("context: %w", err)`. Do not log and return same error at same layer.
- **Package dependencies** — must remain acyclic
- **Concurrency** — goroutines for Slack/Email watchers. Channels between components. Keep main UI thread responsive.
- **File creation** — always write complete `.go` files with `package` declaration
- **Formatting** — all code must pass `gofmt` before commit

---

## 15. Testing & Coverage

- Tests live in `*_test.go` files in a dedicated `_test` package (e.g., `package router_test`)
- All tests use **testify test suites** (`suite.Suite`):

  ```go
  type RouterSuite struct {
      suite.Suite
  }

  func (s *RouterSuite) TestImportanceScoring() {
      // ...
  }

  func TestRouter(t *testing.T) {
      suite.Run(t, new(RouterSuite))
  }
  ```

- After any code/test change, **always run `just test`** across whole project
- Use `s.T().TempDir()` for filesystem/database test artifacts
- Tests must be deterministic — no live external service dependencies (mock Slack/Email/Ollama)

### Coverage Gates

| Level | Threshold |
|---|---|
| Target | ≥ 80% |
| Warning | 65–79.99% |
| High risk | 50–64.99% |
| Fail | < 50% |

---

## 16. Just Commands

| Command | What it does |
|---|---|
| `just build` | Clean, compile to `_build/cue` |
| `just test` | Run tests with short output |
| `just test-verbose` | Run tests with verbose output |
| `just test-coverage` | Run tests, generate HTML report, check gates |
| `just watch` | Watch `.go` changes, re-run tests |
| `just run` | `go run ./cmd/cue` with example config |
| `just fmt` | `go fmt ./...` |
| `just lint` | `gofmt` check + `go vet` |
| `just tidy` | `go mod tidy && go mod verify` |
| `just security` | `gosec ./...` |
| `just vulncheck` | `govulncheck ./...` |
| `just clean` | Remove `_build/` |

---

## 17. Validation Sequence

Before marking work complete, run in order:

1. `just fmt` — format all Go code
2. `just lint` — formatting check + go vet
3. `just tidy` — module hygiene
4. `just test` — focused tests for changed packages
5. Broader tests as needed
6. `just security` — gosec static analysis
7. `just vulncheck` — vulnerability scan

---

## 18. Implementation Phases

### Phase 1: Smart Routing + Feedback Buffer + Bare-Bones UI

**Components:**
1. Config loading + validation
2. Message data model (SQLite)
3. Deterministic routing rules
4. Ollama client + scoring
5. Slack watcher (polling, batch)
6. Email watcher (polling, batch)
7. Router orchestration
8. chromem-go vector integration
9. Feedback buffer (storage + review)
10. Audio alerts
11. Fyne GUI (notification queue, activity log, feedback review)

**Success Criteria:**
- ✅ Slack/Email messages fetched and routed <500ms p95
- ✅ Deterministic rules (new channel IS=9, @mention IS=8) work correctly
- ✅ Ollama scoring consistent and meaningful
- ✅ 20–30% false positive rate acceptable
- ✅ Feedback buffer captures 100 messages per source
- ✅ User can rate messages 0–10 with optional notes
- ✅ Audio alerts cross-platform
- ✅ GUI functional (bare-bones, no animations)
- ✅ Graceful error handling
- ✅ All code documented, tested, passing coverage gates

### Phase 2: Day Planner + Timer (Future)

- Internal todo list
- Calendar integration (ICS/CalDAV)
- Pomodoro schedule generation
- Timer + task tracking

### Phase 3: Animations (Future)

- Fairy character states + animation

---

## 19. Agent Guardrails

- Do not revert unrelated local changes
- Keep changes minimal and in-scope
- Avoid speculative refactors
- When unsure about a requirement, consult this file first
- Respect Agent Teams context isolation: Test Designer reads only requirements; Implementer reads only failing tests; Refactorer sees final code only

---

## 20. Key Design Decisions Locked

- ✅ Batching: 10-minute polling cycles, batch process all messages
- ✅ Notification queue: [Source|Sender|Channel|Message], 15 char truncation each
- ✅ Rating: 0–10 buttons
- ✅ Rules: New channel (IS=9), @mention (IS=8), no "deadline/urgent" keywords, no sender-based importance
- ✅ Config: TOML only, controls db/Ollama/logging paths
- ✅ Latency: <500ms p95 routing decision
- ✅ False positives: 20–30% acceptable
- ✅ Feedback: manual review + vector embeddings for future learning