# Feature 058: Vector Score Advisor Wiring

**Phase:** Phase-6-Feature-058
**Type:** Bugfix
**Severity:** Medium
**Status:** Planned
**Packages:** `cmd/cue/`
**Related:** Feature 042 (Vector-Assisted Routing), Feature 043 (chromem-go Vector Database)

---

## Bug Description

The router is initialized with `nil` for the vector score advisor parameter in `main.go`, even though:
- The `VectorScoreAdvisor` interface and implementation exist (Feature 042)
- The `vectorStore` is created and passed to the buffer service
- Config fields `vector_enabled`, `vector_similarity_threshold`, `vector_top_n`, `vector_damping_factor` exist and are validated

The vector scoring feature is completely disabled in the production setup despite being fully implemented.

## Expected Behavior

When `cfg.Orchestrator.Router.VectorEnabled` is true, the vector score advisor should be instantiated using the chromem-go vector store and passed to the router constructor. User feedback should influence future routing decisions.

## Actual Behavior

`cmd/cue/main.go:142-151` — Router constructor receives `nil` for the advisor parameter. Vector-assisted routing never runs.

## Root Cause

The composition root in `main.go` was not updated to wire the vector advisor when Feature 042 was completed — or the wiring was removed/missed during a later refactor.

## Proposed Fix

In `main.go`, conditionally create a `VectorScoreAdvisor` when `cfg.Orchestrator.Router.VectorEnabled` is true. Pass the chromem-go vector store (already instantiated) and the config parameters to the advisor constructor. Pass the advisor to the router instead of `nil`.

## Test Strategy

- RED: Test that when vector is enabled in config, the router receives a non-nil advisor
- GREEN: Wire the advisor in `main.go`
- REFACTOR: Clean up conditional construction
