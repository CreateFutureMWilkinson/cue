# Feature 093: Model Benchmark Tool

**Phase:** Phase-8-Feature-093
**Status:** Planned
**Packages:** `cmd/cue-bench/`, `internal/service/decisionengine/` (shared types only)
**Depends on:** Feature 092 (uses the optimized prompt), Feature 094 (uses `BuildPromptWithExamples`, `FewShotExample`, `ScoreWithContext`; config uses `calibration_*` field prefix)

---

## Overview

A Go CLI tool (`cmd/cue-bench/`) that benchmarks Ollama models for message importance scoring. It runs a curated corpus of messages through multiple models using the same prompt template and scoring types as Cue's production code, then compares inference time, score quality, JSON compliance, and few-shot calibration effectiveness across models.

## Motivation

Cue currently defaults to `neural-chat` (7B) for scoring. With the trickle queue (Feature 086) reducing throughput pressure, there's an opportunity to evaluate smaller models (phi3:mini 3.8B, gemma2:2b, qwen2.5:3b, llama3.2:3b) that consume less GPU/CPU per call. But switching models blindly risks degrading classification quality — especially for the few-shot calibration loop (Feature 094), which depends on the model's ability to reason about user-rated examples.

A benchmark tool that shares Cue's actual prompt and scoring code ensures that test results accurately predict production behavior.

## Why Go (Not Python)

The benchmark tool is written in Go to guarantee **prompt parity** with production:

- Imports `internal/service/decisionengine` types directly (`PromptTemplate`, `ScorerResponse`, `OllamaRequest`)
- Uses the same `BuildPrompt()` function (Feature 092) and `BuildPromptWithExamples()` function (Feature 094) and JSON parsing logic
- Eliminates drift between "what the benchmark tested" and "what Cue runs in production"
- Compiles as a single binary alongside the main Cue binary

If the prompt changes in Feature 092 or any future update, the benchmark tool automatically uses the new prompt without manual synchronization.

## Architecture

```
cmd/cue-bench/
    main.go           # CLI entry point, orchestration
    corpus.go          # Test message corpus (embedded)
    reporter.go        # Results formatting + output
```

The tool is a standalone binary that shares types from `internal/service/decisionengine/` but has no runtime dependency on the rest of Cue (no SQLite, no UI, no vector store).

## Test Corpus

The corpus is a set of representative messages embedded in the binary via `//go:embed`. Each message has:

```go
type CorpusEntry struct {
    ID              string            `json:"id"`
    Source          string            `json:"source"`
    Sender          string            `json:"sender"`
    Channel         string            `json:"channel"`
    Content         string            `json:"content"`
    ExpectedBand    string            `json:"expected_band"`    // "notified", "buffered", "ignored"
    Tags            []string          `json:"tags"`
    UserRating      *int              `json:"user_rating"`      // If set, this entry can serve as a few-shot example
}
```

The `user_rating` field serves double duty: entries with ratings act as the "feedback corpus" for few-shot benchmark runs. Entries without ratings are the messages being scored.

### Corpus Categories

The corpus must cover the full spectrum of message types Cue encounters:

| Category | Examples | Expected Band | Count |
|---|---|---|---|
| **Critical** | Server down, production outage, security incident | notified | 8-10 |
| **Deadline** | "Due by EOD", "reminder: review by Friday" | notified | 5-7 |
| **@mention** | Direct mention in thread (not caught by deterministic rule) | notified | 3-5 |
| **Routine** | standup updates, PR review requests, meeting notes | ignored | 10-15 |
| **Noise** | bot messages, automated reports, emoji reactions | ignored | 10-15 |
| **Ambiguous** | messages that reasonable people disagree on | buffered | 8-10 |
| **Edge cases** | empty body, very long content, non-English, code snippets | varies | 5-8 |

**Target corpus size: 50-70 messages.** Of these, ~20-25 should have `user_rating` set to act as the few-shot example pool. The remaining ~30-45 are scored.

Large enough for statistical significance, small enough to run in <10 minutes per model.

### Corpus File Format

Messages are stored in `cmd/cue-bench/corpus.json` and embedded at compile time:

```json
[
  {
    "id": "critical-outage-01",
    "source": "slack",
    "sender": "ops-bot",
    "channel": "#incidents",
    "content": "🔴 CRITICAL: API gateway returning 503 for all regions. P0 incident declared.",
    "expected_band": "notified",
    "tags": ["outage", "critical"],
    "user_rating": null
  },
  {
    "id": "rated-noise-01",
    "source": "slack",
    "sender": "deploy-bot",
    "channel": "#deployments",
    "content": "Deployed service-foo v2.3.1 to staging. All health checks passing.",
    "expected_band": "ignored",
    "tags": ["noise", "bot"],
    "user_rating": 1
  }
]
```

## Benchmark Modes

### Mode 1: Base Scoring (No Examples)

Scores all unrated corpus entries using the base prompt (no few-shot examples). This measures the model's raw classification ability and establishes the baseline.

### Mode 2: Few-Shot Scoring

For each unrated corpus entry, selects the most relevant rated entries (by simple text similarity or tag overlap) and includes them as few-shot examples in the prompt. This measures the model's ability to use calibration context.

Runs are performed at multiple example counts to measure the impact:

| Run | Examples | Purpose |
|---|---|---|
| Base | 0 | Raw model capability |
| Light | 1 | Minimal context — does even one example help? |
| Medium | 3 | Moderate context |
| Full | 5 | Production-equivalent context (Feature 094 default) |

This directly answers: **"Can this smaller model effectively use few-shot examples for calibration?"** A model that scores well at 0 examples but degrades at 5 is a poor candidate for Cue, even if it's fast.

### Example Selection for Benchmarks

The benchmark tool does not have a vector store. Instead, it uses a simple heuristic for selecting few-shot examples from the rated corpus entries:

1. Tag overlap: prefer rated entries whose tags overlap with the scored entry's tags
2. Source match: prefer same-source examples (Slack→Slack, email→email)
3. Random tiebreak: when multiple entries qualify equally, select randomly (seeded for reproducibility)

This is intentionally cruder than production's cosine similarity — the benchmark measures the model's ability to reason about examples, not the quality of example selection.

## Metrics Collected

For each model × message × example-count combination:

| Metric | Description |
|---|---|
| `inference_ms` | Wall-clock time from request sent to response received |
| `importance_score` | The model's 0-10 importance score |
| `confidence_score` | The model's 0.0-1.0 confidence score |
| `reasoning` | The model's explanation (for qualitative review) |
| `json_valid` | Whether the response was valid JSON without fixup |
| `band` | Derived routing band (notified/buffered/ignored) using default thresholds |
| `band_correct` | Whether `band` matches `expected_band` |

### Aggregate Metrics Per Model Per Example Count

| Metric | Description |
|---|---|
| `band_accuracy` | % of messages routed to the correct band |
| `false_positive_rate` | % of ignored messages incorrectly notified |
| `false_negative_rate` | % of notified messages incorrectly ignored |
| `json_compliance` | % of responses that were valid JSON |
| `p50_latency` | Median inference time |
| `p95_latency` | 95th percentile inference time |
| `calibration_lift` | Band accuracy at 5 examples minus band accuracy at 0 examples |

### Calibration Lift

The most important metric for evaluating a model's suitability for Cue. `calibration_lift` measures how much the model's accuracy improves when given few-shot examples:

- **Positive lift (>5%):** Model effectively uses calibration context — good candidate
- **Near zero (±2%):** Model ignores examples — calibration won't help, but won't hurt
- **Negative lift (<-2%):** Model is confused by examples — poor candidate, examples degrade performance

A model with lower base accuracy but high calibration lift may be preferable to one with high base accuracy but zero lift, because calibration lift compounds over time as the user rates more messages.

## CLI Interface

```bash
# Run benchmark with all locally available models
cue-bench --baseline neural-chat --models phi3:mini,gemma2:2b,qwen2.5:3b

# Run with specific Ollama host
cue-bench --ollama-host http://localhost:11434 --baseline neural-chat --models phi3:mini

# Run with custom corpus file
cue-bench --corpus ./my-corpus.json --baseline neural-chat --models phi3:mini

# Output as JSON for further analysis
cue-bench --baseline neural-chat --models phi3:mini --format json

# Dry run: validate corpus and check models are available
cue-bench --dry-run --models phi3:mini,gemma2:2b

# Skip few-shot runs (base scoring only)
cue-bench --baseline neural-chat --models phi3:mini --no-fewshot
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--baseline` | `neural-chat` | Baseline model to compare against |
| `--models` | (required) | Comma-separated list of models to benchmark |
| `--ollama-host` | `http://localhost:11434` | Ollama API base URL |
| `--timeout` | `30s` | Per-request timeout |
| `--corpus` | (embedded) | Path to custom corpus JSON file |
| `--format` | `table` | Output format: `table` or `json` |
| `--runs` | `1` | Number of runs per model (for latency averaging) |
| `--dry-run` | `false` | Validate corpus and model availability without running |
| `--cooldown` | `2s` | Pause between inference calls (thermal management) |
| `--no-fewshot` | `false` | Skip few-shot benchmark runs |
| `--seed` | `42` | Random seed for reproducible example selection |

## Output Format

### Table Output (Default)

```
Model Benchmark Results
=======================
Baseline: neural-chat (7B)
Corpus: 58 messages (35 scored, 23 rated examples), 1 run(s)

Base Scoring (0 examples):

Model           | Band Acc | FP Rate | FN Rate | JSON % | p50 ms | p95 ms
----------------|----------|---------|---------|--------|--------|-------
neural-chat     | 82.9%    | 15.5%   | 5.2%    | 96.6%  |   1842 |   3201
phi3:mini       | 79.3%    | 19.0%   | 6.9%    | 98.3%  |    923 |   1647
gemma2:2b       | 74.1%    | 24.1%   | 8.6%    | 94.8%  |    612 |   1102
qwen2.5:3b      | 77.6%    | 20.7%   | 7.0%    | 100.0% |    784 |   1389

Few-Shot Calibration (5 examples):

Model           | Band Acc | FP Rate | FN Rate | JSON % | p50 ms | p95 ms | Cal Lift
----------------|----------|---------|---------|--------|--------|--------|--------
neural-chat     | 88.6%    | 10.2%   | 4.1%    | 97.1%  |   2103 |   3842 |  +5.7%
phi3:mini       | 86.1%    | 12.4%   | 5.0%    | 98.6%  |   1134 |   1921 |  +6.8%
gemma2:2b       | 75.3%    | 23.0%   | 7.9%    | 95.4%  |    798 |   1347 |  +1.2%
qwen2.5:3b      | 84.2%    | 14.1%   | 5.5%    | 100.0% |    987 |   1612 |  +6.6%

✓ phi3:mini: best calibration lift (+6.8%) with 50% latency reduction — recommended
⚠ gemma2:2b: minimal calibration lift (+1.2%) — few-shot examples have little effect
```

### JSON Output

Full per-message results for deeper analysis, including each model's score, reasoning, latency, and example count for every corpus entry.

## Shared Code Boundary

The benchmark tool imports from `internal/service/decisionengine/`:

| Symbol | Used For | Created by |
|---|---|---|
| `BuildPrompt` | Base prompt construction (no examples) | Feature 092 (exported) |
| `BuildPromptWithExamples` | Few-shot prompt construction | Feature 094 |
| `ScorerResponse` | Parsing model output identically | Existing (exported) |
| `OllamaRequest` | Sending requests identically | Existing (exported) |
| `OllamaResponse` | Parsing Ollama API responses identically | Existing (exported) |
| `FewShotExample` | Same struct used in production few-shot path | Feature 094 |

A parity test should be added that asserts the benchmark uses the same prompt construction path as production.

### Export Considerations

Currently `buildPrompt`, `scorerResponse`, `ollamaRequest`, and `ollamaResponse` are unexported. These types are stable, well-defined API contracts with Ollama. The `internal/` path already prevents external consumption. Export them as part of Feature 092 (which modifies `ollama_client.go` anyway).

Note: `extractJSON` is removed by Feature 092 — do not export it.

## Build Integration

```justfile
bench:
    go build -o _build/cue-bench ./cmd/cue-bench/
```

The benchmark binary is built separately from the main Cue binary but from the same module.

## Future Extensions

- **Regression testing**: store baseline results and detect score drift on prompt changes
- **Corpus expansion**: CLI flag to append real messages from the Cue database (with user consent) to the embedded corpus
- **Embedding model benchmarking**: extend to compare embedding models for the calibration loop (separate from inference models)

## Files

| File | Action |
|---|---|
| `cmd/cue-bench/main.go` | **New** — CLI entry point, model orchestration |
| `cmd/cue-bench/corpus.go` | **New** — corpus loading + embedded default corpus |
| `cmd/cue-bench/corpus.json` | **New** — default test message corpus |
| `cmd/cue-bench/reporter.go` | **New** — table + JSON output formatting |
| `internal/service/decisionengine/ollama_client.go` | **Modify** — export shared types/functions for benchmark access |

## Test Coverage

- Corpus JSON parsing (valid, malformed, empty, entries with/without user_rating)
- Example selection by tag overlap + source match
- Metric calculations (band accuracy, calibration lift, false positive/negative rates)
- Table + JSON output formatting
- CLI flag validation (missing required flags, invalid model names)
- Dry-run mode validates corpus and model availability
- Parity test: assert `buildPromptWithExamples()` output matches between benchmark and production code paths
- Few-shot runs at 0, 1, 3, 5 example counts produce distinct results
