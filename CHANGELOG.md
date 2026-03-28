# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
with the addition of a **Breaking** section for backwards-incompatible changes
that would otherwise appear under **Changed**. Entries under **Breaking** trigger
a major version bump in automated release recommendation logic.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking

### Added

- **Config loading and validation** — TOML-based configuration at `~/.cue/config.toml` with safe defaults, auto-creation on first run, tilde expansion, and table-driven validation (Phase-1-Feature-1)
- **SQLite message repository** — Pure Go SQLite storage (`modernc.org/sqlite`) with WAL mode, FIFO eviction (100 messages per source), upsert by MessageID, and full CRUD operations (Phase-1-Feature-2)
- **Deterministic routing rules** — Decision engine router with channel_join (IS=9) and @mention (IS=8) deterministic rules, Scorer interface for LLM evaluation, configurable threshold-based routing (NOTIFIED/BUFFERED/IGNORED), and safe fallback on scorer failure (Phase-1-Feature-3)
- **Ollama client scoring** — HTTP client implementing the Scorer interface for local Ollama LLM inference, with JSON prompt construction, markdown code block extraction, configurable timeout, and graceful error handling (Phase-1-Feature-4)
- **Slack watcher polling** — Polls Slack channels for new messages, detects channel joins (IS=9), includes thread context for replies, tracks per-channel timestamps to avoid reprocessing (Phase-1-Feature-5)
- **Email watcher polling** — Polls IMAP inbox for new messages, extracts sender/subject/folder/body, detects @mentions in To/CC/BCC (case-insensitive), tracks last-seen UID to avoid reprocessing (Phase-1-Feature-6)
- **Router orchestration** — Coordinates watchers, router, and repository in batch polling loops with per-source goroutines, immediate first poll, configurable intervals, activity event emission, graceful error handling (individual store errors don't abort batch), and idempotent shutdown (Phase-1-Feature-7)
- **Vector store with cosine similarity** — In-memory vector storage with pluggable embedding function, cosine similarity search (topN), message ID association for feedback linking, and zero-denominator handling (Phase-1-Feature-8)
- **Feedback buffer service** — Review workflow for buffered messages (IS >= 7, CS < 0.8) with oldest-first retrieval, user rating (0-10) with optional notes, message deletion, optional vector embedding on save for learning loop, and graceful embedding failure handling (Phase-1-Feature-9)
- **Audio alerts** — Cross-platform audio notifications using `beeep` with sharp ping on NOTIFIED messages (1000 Hz), startup chime (600 Hz), shutdown tone (400 Hz), 2-second cooldown to prevent spam, configurable on/off via config, and non-fatal error handling (Phase-1-Feature-10)
- **Fyne GUI** — Desktop GUI with presenter/view architecture: notification queue (NOTIFIED messages newest-first with 15-char truncation), real-time activity log (ring buffer, error highlighting), feedback buffer review (0-10 rating buttons, skip, delete, counter), app lifecycle management (startup/shutdown alerts), and `cmd/cue/main.go` composition root wiring all Phase 1 components (Phase-1-Feature-11)
- `MessageType` field on `Message` struct for distinguishing message event types
- Agent team TDD workflow with test-designer, implementer, and refactorer agents
- Agent log tracking duration and token usage per TDD phase
- Validation pipelines

### Changed

- **GUIConfig** — Replaced web-oriented `Host`/`Port` fields with Fyne-relevant `WindowWidth`/`WindowHeight` (defaults: 1200x800) (Phase-1-Feature-11)

### Removed

### Fixed
