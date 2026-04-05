[![CI](https://github.com/CreateFutureMWilkinson/cue/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/CreateFutureMWilkinson/cue/actions/workflows/ci.yml)
[![Coverage](https://s3.hrafn.xyz/aether-workflow-report-artefacts/cue/branch/main/coverage-badge.svg)](https://s3.hrafn.xyz/aether-workflow-report-artefacts/cue/branch/main/coverage.html)

# Cue

A local-first, privacy-centric ADHD-friendly productivity assistant. Cue monitors Slack and Email for high-stakes messages (deadlines, outages, @mentions, channel joins), evaluates them using local Ollama inference, and routes them based on importance and confidence scores. No data leaves your machine.

## Status

**Phase 1** — Smart Routing + Feedback Buffer + Bare-Bones UI

| # | Component | Status |
|---|---|---|
| 1 | Config loading + validation | Done |
| 2 | Message data model (SQLite) | Done |
| 3 | Deterministic routing rules | Done |
| 4 | Ollama client + scoring | Done |
| 5 | Slack watcher | Done |
| 6 | Email watcher | Done |
| 7 | Router orchestration | Done |
| 8 | Vector integration (chromem-go) | Done |
| 9 | Feedback buffer | Done |
| 10 | Audio alerts | Done |
| 11 | Fyne GUI | Done |

## Requirements

- Go 1.26+
- [Ollama](https://ollama.ai) running locally with `neural-chat` and `nomic-embed-text` models

## Quick Start

```bash
# Build
just build

# Run (creates ~/.cue/config.toml with defaults on first run)
just run

# Test
just test
```

## Configuration

Cue uses TOML configuration at `~/.cue/config.toml`. A default config is created on first run. See [CLAUDE.md](.claude/CLAUDE.md) Section 6 for the full schema.

## Architecture

- **Config** (`internal/config/`) — TOML loading, validation, defaults
- **Repository** (`internal/repository/`) — Message persistence with SQLite (pure Go, no CGO)
- **Decision Engine** (`internal/service/decisionengine/`) — Deterministic rules + scorer-based routing into three destinations:
  - **Notified** (importance >= 7, confidence >= 0.8) — audio alert + GUI notification queue
  - **Buffered** (importance >= 7, confidence < 0.8) — silent queue for manual review in feedback buffer
  - **Ignored** (importance < 7) — logged to database, available for manual review
- **Orchestrator** (`internal/service/orchestrator/`) — Coordinates watchers, router, and repository in batch polling loops (poll → route → store) with per-source goroutines and activity event emission
- **Watchers** (`internal/service/watcher/`) — Slack and Email polling
- **Alert** (`internal/alert/`) — Cross-platform audio alerts via `beeep` with cooldown
- **UI** (`internal/ui/`) — Fyne desktop GUI with presenter/view architecture (notification queue, activity log, feedback review)
- **Entry Point** (`cmd/cue/`) — Composition root wiring all components

## Development

```bash
just fmt          # Format code
just lint         # Format check + go vet
just test         # Run tests
just test-coverage # Coverage report with gates (target: ≥80%)
just tidy         # Module hygiene
```

TDD (Red-Green-Refactor) is required for all feature work. See [CLAUDE.md](.claude/CLAUDE.md) Section 13.

## License

TBD
