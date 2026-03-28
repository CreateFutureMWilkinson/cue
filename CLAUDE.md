# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Cue is a local-first ADHD-friendly productivity assistant in Go. It monitors Slack and Email for high-stakes messages, scores them via local Ollama inference (importance 0–10, confidence 0.0–1.0), and routes them as NOTIFIED / BUFFERED / IGNORED. No data leaves the machine. Full product spec lives in `.claude/CLAUDE.md`.

Module path: `github.com/CreateFutureMWilkinson/cue` · Go 1.25+

## Build & Test Commands

```bash
just test                # run all tests (short output)
just test-verbose        # run all tests (verbose)
just test-coverage       # tests + HTML coverage report in _build/coverage.html
just fmt                 # go fmt ./...
just lint                # gofmt check + go vet
just tidy                # go mod tidy && go mod verify
just build               # compile to _build/cue
just security            # gosec ./...
just vulncheck           # govulncheck ./...
```

Run a single test suite:
```bash
go test -count=1 -v -run TestRouter ./internal/service/decisionengine/
```

Run a single test method within a suite:
```bash
go test -count=1 -v -run TestRouter/TestDeterministicChannelJoin ./internal/service/decisionengine/
```

Validation sequence before marking work complete: `just fmt && just lint && just tidy && just test`

## Architecture

```
internal/
  config/              Config loading from ~/.cue/config.toml (TOML, BurntSushi/toml)
  repository/          MessageRepository interface
    implementation/
      sqlite/          Pure-Go SQLite impl (modernc.org/sqlite, WAL mode, FIFO eviction)
  service/
    decisionengine/    Router (deterministic rules + threshold routing) + OllamaClient
    orchestrator/      Batch polling loop — per-source goroutines, activity events
    watcher/           SlackWatcher + EmailWatcher (poll interfaces, not real API clients yet)
    buffer/            Feedback buffer — review, rate 0–10, optional vector embedding
    vector/            In-memory vector store with cosine similarity
  alert/               Configurable audio alerts (file playback + beeep fallback)
  ui/                  GUI placeholder (not implemented)
```

**Dependency flow (acyclic):**
orchestrator → watcher/{slack,email} + decisionengine/router + repository/sqlite + buffer
router → decisionengine/ollama_client (via Scorer interface)
buffer → vector (via VectorEmbedder interface)

All cross-package dependencies use interfaces. Constructors validate all injected deps.

## Key Routing Rules

1. `message_type == "channel_join"` → IS=9, CS=1.0, NOTIFIED
2. `@username` in content → IS=8, CS=1.0, NOTIFIED
3. All else → Ollama-scored, then: IS≥7 AND CS≥0.8 → NOTIFIED; IS≥7 AND CS<0.8 → BUFFERED; else IGNORED
4. Ollama failure fallback → IS=7, CS=0.0 → BUFFERED

Importance is NEVER determined by sender identity.

## Hard Constraints

- **SQLite driver:** `modernc.org/sqlite` only. Never `mattn/go-sqlite3` (CGO).
- **LLM:** Local Ollama only. No hosted/cloud LLM providers ever.
- **Config:** TOML only (`~/.cue/config.toml`). No hardcoded values or CLI feature flags.
- **Testing:** All tests use testify `suite.Suite` in `_test` package suffix. TDD (red-green-refactor) required.
- **Formatting:** All code must pass `gofmt` before commit.

## Testing Conventions

- Test packages use `_test` suffix (e.g., `package router_test`)
- All test files use testify suite pattern:
  ```go
  type RouterSuite struct { suite.Suite }
  func TestRouter(t *testing.T) { suite.Run(t, new(RouterSuite)) }
  func (s *RouterSuite) TestSomething() { s.Equal(expected, actual) }
  ```
- Use `s.T().TempDir()` for filesystem/DB test artifacts
- Mock all external deps (Slack, Email, Ollama) — no live service dependencies
- Coverage target: ≥80%

## TDD Workflow

**All features and bug fixes MUST use agent teams** (test-designer → implementer → refactorer) with context isolation. No exceptions — even simple changes go through the full pipeline. See `.claude/agents/` for role definitions.

One commit per phase. **Run `just fmt` as the last step before every commit** (red, green, and refactor):
- `test(scope): failing test for ...` — failing tests only
- `feat(scope): implement ... [tests pass]` — minimal code to pass
- `refactor(scope): improve ...` — cleanup, tests stay green

### Post-Feature Docs Commit (Required)

After the refactor commit, before marking a feature complete, create a single `docs(scope): ...` commit that includes:

1. **Per-feature design doc** at `docs/Feature-N-Name.md` — overview, design decisions, API, error handling, integration points, test coverage summary, and TDD agent stats table.
2. **Agent stats log** in `docs/agent-log.md` — table with columns: Implementation Phase, TDD Phase, Agent, Duration, Tokens, Commit. Log all three TDD phases.
3. **CHANGELOG.md** update — Keep a Changelog format. Use `### Breaking` for backwards-incompatible changes.
4. **README.md** update — Keep project overview, setup, and feature status current.

## Implementation Status

**Done (Phase 1):** config, SQLite repository, router, Ollama client, Slack/Email watchers, orchestrator, feedback buffer, vector store, audio alerts, Fyne GUI, `cmd/cue/main.go` entry point — all with tests.

**Not yet implemented:** real Slack/IMAP API clients (currently using placeholder implementations).
