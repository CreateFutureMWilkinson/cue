# Feature 095: Embedding Model Benchmarking

**Phase:** Phase-8-Feature-095
**Status:** Done
**Packages:** `cmd/cue-bench/`
**Depends on:** Feature 093 (cue-bench tool), Feature 094 (few-shot calibration loop, `BuildPromptWithExamples`)

---

## Overview

Adds an `--embed-model` flag to cue-bench that replaces the tag-based few-shot example selection with vector-similarity selection using a real Ollama embedding model. This enables benchmarking inference model + embedding model combinations to find the best pairing for production routing accuracy.

## Motivation

Feature 093 established cue-bench for comparing inference models, but its example selection uses tag overlap — a crude heuristic that doesn't reflect production behavior. In production (Feature 094), Cue uses chromem-go with Ollama embeddings to find semantically similar rated messages for few-shot injection. The quality of that retrieval directly affects scoring accuracy: a poor embedding model retrieves irrelevant examples, which can confuse the inference model and degrade routing.

Without embedding benchmarking, switching embedding models (e.g. `nomic-embed-text` to `mxbai-embed-large` or `snowflake-arctic-embed`) is a blind change. This feature closes that gap by letting the user measure the downstream effect of different embedding models on routing accuracy within the same benchmark framework.

## Design

### CLI

One new flag added to the existing cue-bench command:

| Flag | Default | Description |
|---|---|---|
| `--embed-model` | (empty) | Ollama embedding model for vector-based example selection |

When `--embed-model` is omitted, behavior is identical to Feature 093 (tag-based selection). When set, the benchmark uses vector similarity instead.

**Usage:**

```bash
# Benchmark inference models with nomic-embed-text for example retrieval
cue-bench --models phi3:mini,qwen2.5:3b --embed-model nomic-embed-text

# Compare embedding models by running separate invocations
cue-bench --models phi3:mini --embed-model nomic-embed-text
cue-bench --models phi3:mini --embed-model mxbai-embed-large
cue-bench --models phi3:mini --embed-model snowflake-arctic-embed:xs

# Tag-based selection (existing behavior, no change)
cue-bench --models phi3:mini
```

One embedding model per run, multiple inference models benchmarked against that embedding model's retrieval. Compare runs with different `--embed-model` values to evaluate pairings.

### Execution Flow

```
1. Load corpus (unchanged)
   ├── scored entries (UserRating == nil) — test set
   └── pool entries (UserRating != nil) — example candidates

2. IF --embed-model is set:
   ├── Pre-embed phase: embed all pool + scored entries via Ollama /api/embed
   ├── Collect per-request embedding latencies
   └── Build in-memory EmbedIndex (entry → vector mapping)

3. Scoring loop (per model x example_count x scored_entry):
   ├── IF EmbedIndex exists:
   │   └── Select top-N pool entries by cosine similarity to scored entry
   ├── ELSE:
   │   └── Select by tag overlap (existing SelectExamples)
   ├── Build prompt with selected examples
   ├── POST to Ollama /api/generate (unchanged)
   └── Derive band, check correctness (unchanged)

4. Report: existing metrics + embedding latency (p50/p95) when embed model used
```

### Ollama /api/embed Integration

Call `/api/embed` directly via `net/http`, matching the pattern of `scoreEntry` for `/api/generate`. Do NOT use chromem-go — direct HTTP gives full latency control and avoids adding a runtime dependency to the benchmark tool.

**Request:**
```json
{"model": "nomic-embed-text", "input": "text to embed"}
```

**Response:**
```json
{"embeddings": [[0.123, -0.456, ...]]}
```

Embeddings are generated on-the-fly at benchmark start with no disk caching. The rated pool is typically ~20-25 entries plus ~30-45 scored entries — under 70 embedding calls total, completing in seconds.

### Vector-Based Example Selection

For each scored entry, compute cosine similarity between its embedding and every pool entry's embedding. Return the top-N most similar pool entries as few-shot examples. The `example_count` parameter (controlled by `--no-fewshot`) still determines N, matching the existing [0, 1, 3, 5] schedule.

Cosine similarity: `dot(a, b) / (|a| * |b|)`. Returns 0 for zero-magnitude vectors.

### Report Changes

**Table output** — new line after "Corpus:" when embed model is active:

```
Model Benchmark Results
=======================
Baseline: neural-chat
Corpus: 58 messages (35 scored, 23 rated examples), 1 run(s)
Embed Model: nomic-embed-text (p50: 12ms, p95: 28ms)

Base Scoring (0 examples):
...
```

**JSON output** — includes `embed_model`, `embed_p50_ms`, `embed_p95_ms` fields at the top level when embed model is set. Omitted when tag-based selection is used (backward compatible).

## Architecture

### New File: `cmd/cue-bench/embed.go`

All embedding logic isolated in one file:

```go
// EmbedResult pairs a corpus entry with its embedding vector.
type EmbedResult struct {
    Entry     CorpusEntry
    Embedding []float32
}

// EmbedIndex holds pre-computed embeddings for vector-based selection.
type EmbedIndex struct {
    Pool   []EmbedResult
    Scored map[string][]float32 // keyed by entry ID
}

// embedText calls Ollama /api/embed for a single text input.
// Returns the embedding vector and request latency in milliseconds.
func embedText(ctx context.Context, model, text, host string, httpClient *http.Client) ([]float32, int64, error)

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float64

// SelectExamplesByEmbedding selects up to n pool entries most similar
// to the scored entry's embedding.
func SelectExamplesByEmbedding(entryID string, index EmbedIndex, n int) []CorpusEntry

// BuildEmbedIndex embeds all pool and scored entries, returning the
// index and per-request latencies for metrics reporting.
func BuildEmbedIndex(ctx context.Context, model, host string, pool, scored []CorpusEntry, httpClient *http.Client, progressWriter io.Writer) (EmbedIndex, []int64, error)
```

### Modified Files

| File | Changes |
|---|---|
| `main.go` | Add `EmbedModel string` to `BenchConfig`; add `--embed-model` flag; call `BuildEmbedIndex` when set; pass `*EmbedIndex` to `RunBenchmark`; compute embed percentiles |
| `benchmark.go` | Add `*EmbedIndex` parameter to `RunBenchmark`; conditional example selection (vector vs tag-based) in scoring loop |
| `reporter.go` | Add `EmbedModel`, `EmbedP50Ms`, `EmbedP95Ms` to `BenchReport`; render embed stats in table header and JSON |
| `benchmark_test.go` | Update 2 existing `RunBenchmark` call sites to pass `nil` for new parameter |
| `main_test.go` | Add `EmbedModel` to compile-time `BenchConfig` literal; add flag parsing test |

### New Test File: `cmd/cue-bench/embed_test.go`

`EmbedSuite` with testify suite convention, covering:

- `CosineSimilarity` — identical, orthogonal, zero-magnitude vectors
- `embedText` — success with mock server, HTTP error, invalid JSON
- `SelectExamplesByEmbedding` — correct ranking, N capping, missing entry ID
- `BuildEmbedIndex` — pool + scored entries indexed, latencies collected

## Metrics

Embedding latency is a **report-level** metric (not per-model), since embedding happens once for all inference models in a run.

| Metric | Level | Description |
|---|---|---|
| `embed_p50_ms` | Report | Median embedding latency across all embed calls |
| `embed_p95_ms` | Report | 95th percentile embedding latency |

All existing per-model metrics (band accuracy, FP/FN rate, JSON compliance, inference p50/p95, calibration lift) remain unchanged. The downstream effect of different embedding models is captured through these existing metrics — better embeddings lead to better example selection, which leads to higher band accuracy and calibration lift.

## Error Handling

| Error | Action |
|---|---|
| Ollama /api/embed returns HTTP error | Return error from `BuildEmbedIndex`, abort benchmark |
| Ollama /api/embed returns invalid JSON | Return error from `BuildEmbedIndex`, abort benchmark |
| `--embed-model` specifies unavailable model | Ollama returns error on first embed call, caught above |
| Entry ID not found in EmbedIndex | `SelectExamplesByEmbedding` returns nil (no examples for that entry) |

Embedding errors are fatal to the benchmark run (unlike inference errors which fall back to IS=7, CS=0.0). Rationale: if the embedding model is broken, all example selection is meaningless — fail fast rather than produce misleading results.

## TDD Micro-Loops

| Loop | Behavior | New/Modified Files |
|---|---|---|
| 1 | `CosineSimilarity` pure math (identical, orthogonal, zero vectors) | embed.go, embed_test.go |
| 2 | `embedText` HTTP call + error handling (mock Ollama server) | embed.go, embed_test.go |
| 3 | `SelectExamplesByEmbedding` (ranking, N cap, missing ID) | embed.go, embed_test.go |
| 4 | `BuildEmbedIndex` (indexes pool + scored, collects latencies) | embed.go, embed_test.go |
| 5 | CLI flag + `RunBenchmark` integration (conditional selection) | main.go, benchmark.go, main_test.go, benchmark_test.go |
| 6 | Reporter (embed stats in table header + JSON metadata) | reporter.go, embed_test.go or reporter_test.go |

Dependencies: Loop 3 depends on 1. Loop 4 depends on 2. Loop 5 depends on 3 + 4. Loop 6 depends on 5. Loops 1 and 2 can run in parallel.

## Future Extensions

- **Disk caching**: Cache embeddings keyed by model name + corpus hash for faster repeated runs
- **Batch embedding**: Call `/api/embed` with multiple inputs per request if Ollama adds batch support
- **Retrieval quality metrics**: Measure tag overlap of vector-selected vs tag-selected examples to quantify retrieval improvement
- **Multiple embed models per run**: `--embed-models` flag to iterate the full benchmark once per embedding model, producing comparative tables

## Files

| File | Action |
|---|---|
| `cmd/cue-bench/embed.go` | **New** — `EmbedResult`, `EmbedIndex`, `embedText`, `CosineSimilarity`, `SelectExamplesByEmbedding`, `BuildEmbedIndex` |
| `cmd/cue-bench/embed_test.go` | **New** — `EmbedSuite` with ~11 test methods |
| `cmd/cue-bench/main.go` | **Modify** — `BenchConfig.EmbedModel`, `--embed-model` flag, `BuildEmbedIndex` call, embed percentile computation |
| `cmd/cue-bench/benchmark.go` | **Modify** — `RunBenchmark` signature (`*EmbedIndex` param), conditional example selection |
| `cmd/cue-bench/reporter.go` | **Modify** — `BenchReport` embed fields, table/JSON rendering |
| `cmd/cue-bench/benchmark_test.go` | **Modify** — update `RunBenchmark` call sites (pass `nil`) |
| `cmd/cue-bench/main_test.go` | **Modify** — `BenchConfig` literal, `--embed-model` flag test |
