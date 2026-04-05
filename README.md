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
| 12 | Configurable audio alerts (amendment) | Done |
| 13 | gopxl/beep audio player (amendment) | Done |

**Phase 3** — Animations

| # | Component | Status |
|---|---|---|
| 14 | Character animation system | Done |

**Hotfixes**

| ID | Scope | Status |
|---|---|---|
| 14-A | Security hardening (gosec G304/G404, x/image, x/net CVEs) | Done |

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
- **Alert** (`internal/alert/`) — Configurable audio alerts with real file playback via gopxl/beep (MP3/WAV/OGG), beeep fallback, configurable cooldown and volume
- **UI** (`internal/ui/`) — Fyne desktop GUI with presenter/view architecture (notification queue, activity log, feedback review, character animation)
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

## Why Cue Instead of OpenClaw?

[OpenClaw](https://openclaw.ai/) is a popular general-purpose AI agent that automates tasks across messaging platforms via a skills system. It's powerful and extensible, but it's a different tool for a different problem. Here's how they compare:

| | Cue | OpenClaw |
|---|---|---|
| **Purpose** | Single-purpose ADHD notification triage | General-purpose AI agent |
| **LLM** | Local Ollama only — no data leaves your machine | Cloud LLMs (Claude, GPT, DeepSeek) |
| **Privacy** | Strict local-first by design | Connects to cloud services |
| **Interface** | Desktop GUI (Fyne) | Chat-based (Signal, Telegram, Discord) |
| **Extensibility** | Fixed pipeline: fetch → score → route | Plugin/skills system for arbitrary workflows |
| **ADHD support** | Core design goal — noise filtering, importance scoring, batch processing, feedback loop | Available via community skills, not core |
| **Feedback loop** | Built-in: rate messages 0–10, vector embeddings for learning | Not built-in |

OpenClaw *could* approximate Cue's behavior with the right skills, but wouldn't have the tight scoring/routing engine, the built-in feedback loop, or the guarantee that nothing ever leaves your machine. Cue is purpose-built for one job: making sure ADHD users catch the messages that matter without drowning in noise.

## License

TBD
