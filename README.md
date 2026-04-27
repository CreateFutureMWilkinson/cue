[![CI](https://github.com/CreateFutureMWilkinson/cue/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/CreateFutureMWilkinson/cue/actions/workflows/ci.yml)
[![Coverage](https://s3.hrafn.xyz/aether-workflow-report-artefacts/cue/branch/main/coverage-badge.svg)](https://s3.hrafn.xyz/aether-workflow-report-artefacts/cue/branch/main/coverage.html)

# Cue

A local-first, privacy-centric ADHD-friendly productivity assistant. Cue monitors Slack and Email for high-stakes messages (deadlines, outages, @mentions, channel joins), evaluates them using local Ollama inference, and routes them based on importance and confidence scores. No data leaves your machine.

## Status

Phases 1–8 complete. Phase 9 in progress: server infrastructure (097), message & notification REST API (098), activity event stream (099), server orchestrator wiring (099A), feedback buffer API (100), and todo CRUD API (101A) done — `cue-server` is now a fully functional headless runtime. It runs the orchestrator, queue processor, watchers (registered from enabled Slack/Email accounts in the DB), Ollama scoring, vector store, and HTTP/WebSocket surface in one composition root (`server.NewComposition`). Activity events flow orchestrator → hub → WebSocket clients in real time; alerts broadcast as a new `type: "alert"` envelope so clients render audio/visuals themselves. Sixteen REST endpoints for notifications, messages, feedback buffer review (list/get/rate/delete/stats), and todo CRUD (list/create/get/update/delete with async LLM time estimation); a 500-entry replay ring at `/api/v1/events?since=<seq>`, 16-connection cap, same-origin policy, and ordered shutdown (orchestrator → queue processor → close event channel → HTTP → DB) with `slog` progress logging. `cmd/cue` is unchanged and still runs its own orchestrator until Feature 107. See [docs/Roadmap.md](docs/Roadmap.md) for full implementation status.

## Supported Platforms

| Platform | Architecture | Display Server |
|---|---|---|
| Linux | amd64 | X11, Wayland |
| Linux | arm64 | X11, Wayland |
| macOS | arm64 (Apple Silicon) | Cocoa (native) |

## Requirements

- Go 1.26+ with CGO enabled
- [Ollama](https://ollama.ai) running locally with `neural-chat` and `nomic-embed-text` models
- Platform-specific build dependencies — see [docs/guides/Building.md](docs/guides/Building.md)

## Quick Start

```bash
# Install build dependencies (see docs/guides/Building.md for your platform)
just deps

# Build
just build

# Run (creates ~/.cue/config.toml with defaults on first run)
just run

# Test
just test
```

> **macOS users:** Release binaries are unsigned. On first launch, remove the quarantine flag with `xattr -d com.apple.quarantine ./cue` or allow via System Settings → Privacy & Security. See [docs/guides/Building.md](docs/guides/Building.md#macos-gatekeeper) for details.

## Configuration

Cue uses TOML configuration at `~/.cue/config.toml`. A default config is created on first run. Generate an annotated example with `cue config example`. See [CLAUDE.md](.claude/CLAUDE.md) Section 6 for the full schema.

## Architecture

- **Config** (`internal/config/`) — TOML loading, validation, defaults
- **Repository** (`internal/repository/`) — Message persistence with SQLite (pure Go, no CGO)
- **Decision Engine** (`internal/service/decisionengine/`) — Deterministic rules + scorer-based routing into three destinations:
  - **Notified** (importance >= 7, confidence >= 0.8) — audio alert + GUI notification queue
  - **Buffered** (importance >= 7, confidence < 0.8) — silent queue for manual review in feedback buffer
  - **Ignored** (importance < 7) — logged to database, available for manual review
- **Orchestrator** (`internal/service/orchestrator/`) — Coordinates watchers, router, and repository in batch polling loops (poll → route → store) with per-source goroutines and activity event emission
- **Calendar** (`internal/service/calendar/`) — Provider-agnostic calendar integration; ICS-over-HTTP adapter for Google Calendar secret links
- **Watchers** (`internal/service/watcher/`) — Slack and Email polling
- **Alert** (`internal/alert/`) — Configurable audio alerts with real file playback via gopxl/beep (MP3/WAV/OGG), beeep fallback, configurable cooldown and volume
- **UI** (`internal/ui/`) — Fyne desktop GUI with presenter/view architecture (notification queue, activity log, feedback review, character animation)
- **WASM Character Plugins** (`internal/ui/character/wasmhost/`) — wazero-based plugin host for loading third-party character `.wasm` files at runtime with sandboxed rendering API
- **Entry Point** (`cmd/cue/`) — Composition root wiring all components
- **Character UAT** — `cue uat` subcommand launches full UI with UAT control panel for live character testing (`just run-uat`)

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
