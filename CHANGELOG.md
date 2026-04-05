# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
with the addition of a **Breaking** section for backwards-incompatible changes
that would otherwise appear under **Changed**. Entries under **Breaking** trigger
a major version bump in automated release recommendation logic.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Config loading and validation** — TOML-based configuration at `~/.cue/config.toml` with safe defaults, auto-creation on first run, tilde expansion, and table-driven validation (Phase-1-Feature-1)
- **SQLite message repository** — Pure Go SQLite storage (`modernc.org/sqlite`) with WAL mode, FIFO eviction (100 messages per source), upsert by MessageID, and full CRUD operations (Phase-1-Feature-2)
- **Deterministic routing rules** — Decision engine router with channel_join (IS=9) and @mention (IS=8) deterministic rules, Scorer interface for LLM evaluation, configurable threshold-based routing (NOTIFIED/BUFFERED/IGNORED), and safe fallback on scorer failure (Phase-1-Feature-3)
- **Ollama client scoring** — HTTP client implementing the Scorer interface for local Ollama LLM inference, with JSON prompt construction, markdown code block extraction, configurable timeout, and graceful error handling (Phase-1-Feature-4)
- **Slack watcher polling** — Polls Slack channels for new messages, detects channel joins (IS=9), includes thread context for replies, tracks per-channel timestamps to avoid reprocessing (Phase-1-Feature-5)
- **Email watcher polling** — Polls IMAP inbox for new messages, extracts sender/subject/folder/body, detects @mentions in To/CC/BCC (case-insensitive), tracks last-seen UID to avoid reprocessing (Phase-1-Feature-6)
- `MessageType` field on `Message` struct for distinguishing message event types
- Agent team TDD workflow with test-designer, implementer, and refactorer agents
- Agent log tracking duration and token usage per TDD phase
