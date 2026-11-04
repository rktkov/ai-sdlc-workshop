# Spike S-01 — local model comparison

## Status

Planned. Awaiting approval to install `llama.cpp` and download the model set.

## Question

Which LFM2.5 size and quant clears the product bar on the target laptop? The bar is schema
validity, rubric agreement, first-card latency and memory.

## Target machine

| Property | Value |
|----------|-------|
| Model | MacBook Pro `MacBookPro16,1` |
| CPU | Intel Core i9-9880H, 8 cores, 2.3 GHz |
| Memory | 32 GB |
| Free disk | About 216 GB |
| Runtime now | No `llama.cpp`, Ollama or LM Studio installed |
| GPU note | The CPU path is the baseline. A Metal path can be a follow-up. |

The hardware is CPU-first and older than an Apple Silicon laptop. The result must be
reported as the target-laptop result, not as a general LFM2.5 result.

## Models to test

| ID | Model | Format | Purpose | Approx. size |
|----|-------|--------|---------|--------------|
| `M1` | LFM2.5-350M | QAD Q4_0 | Speed floor and memory floor | ~0.2 GB |
| `M2` | LFM2.5-1.2B-Instruct | QAD Q4_0 | Small candidate | ~0.7 GB |
| `M3` | LFM2.5-2.6B | QAD Q4_0 | Default candidate from the PRD | ~1.5 GB |
| `M4` | LFM2.5-2.6B | Q8_0 | Quality ceiling for the 2.6B family | ~2.8 GB |
| `M5` | LFM2.5-8B-A1B | QAD Q4_0 | MoE stretch case, 1.5B active | ~4.5 GB |
| `M6` | Llama-3.2-3B-Instruct | Q4_K_M | Dense baseline (`R-27`) | ~2.0 GB |

Total download is about 12 GB. `M5` is optional. If time or memory is tight, drop `M5`
first and keep `M1`…`M4` and `M6`.

The LFM2.5 QAD Q4_0 files are the vendor's quantization-aware checkpoints. They are the
right default for a CPU laptop. `M4` is the quality reference, not the default candidate.

## Runtime

- Install `llama.cpp` with Homebrew (`brew install llama.cpp`).
- Run `llama-server` per model on `127.0.0.1:8081`.
- Use the chat template from the GGUF.
- Use `-c 8192`, `-np 1`, `-t 8` and temperature `0.1`.
- The product adapter needs an OpenAI-compatible `/v1/chat/completions` endpoint. The
  spike uses the same shape as `ADR-0004`.

Example command shape:

```sh
llama-server -m model.gguf --host 127.0.0.1 --port 8081 -c 8192 -np 1 -t 8 --jinja
```

If `llama-server` does not accept `response_format` with a JSON schema, the spike falls
back to `json_object` plus a schema in the prompt. That gap is recorded as an adapter
finding.

## Fixtures

The spike uses synthetic content, not owner data.

| Fixture | Content | Purpose |
|---------|---------|---------|
| `F1`…`F5` | Five synthetic transcripts, 250–400 words each | Core quality and schema set |
| `F6` | One long transcript, about 8k tokens | Context and memory |
| `F7` | One empty or unreadable transcript | `no_transcript` path |
| `F8` | One adversarial transcript with instructions in the text | Prompt-injection resistance |

The rubric is a **spike rubric** with four axes:

1. `depth` — substance beyond the title.
2. `novelty` — new information or a new angle.
3. `signal-to-noise` — useful content per minute.
4. `practical-value` — something the owner can use.

This is not rubric v1. `Q3` is still open. The spike rubric is a fixed input so models can
be compared. It is not promoted to product content.

The five core fixtures get a human reference label for the four axes and a binary
`worth watching` label. The labeler is the owner or the agent with owner review.

## Test cases

| ID | Test | Happy or unhappy | Pass condition |
|----|------|------------------|----------------|
| `S01-T01` | Server starts and lists the model | Happy | `/v1/models` or equivalent responds |
| `S01-T02` | One valid request | Happy | OpenAI-shaped response with content |
| `S01-T03` | JSON schema response on `F1`…`F5` | Happy | Valid JSON and all required fields |
| `S01-T04` | Repeat `F1` three times | Happy | Score variance is recorded |
| `S01-T05` | Long transcript `F6` | Happy | No crash, no truncation surprise |
| `S01-T06` | Missing transcript `F7` | Unhappy | Explicit failure, not a fabricated card |
| `S01-T07` | Prompt injection `F8` | Unhappy | The model stays in the schema and ignores the instruction |
| `S01-T08` | Server down or timeout | Unhappy | The client returns an explicit failure |
| `S01-T09` | Invalid JSON response | Unhappy | The validator rejects it and records the failure |
| `S01-T10` | Batch of 20 cards, concurrency 2 | Happy | First analyzed card within 60 s, progress visible |

## Metrics

| Metric | How it is measured | Bar |
|--------|--------------------|-----|
| Schema validity, first pass | Valid JSON with all fields / calls | ≥ 95% |
| Schema validity after one retry | Final valid / calls | 100% |
| Required field completeness | Missing fields / calls | 0 |
| Axis agreement | Mean absolute error against the five labels | ≤ 0.20 |
| Worth-watching agreement | Correct binary labels / 5 | ≥ 4 of 5 |
| Topic match | Normalized topic match / 5 | ≥ 4 of 5 |
| First analyzed card | p95 over three batch runs | ≤ 60 s (`Performance` NFR) |
| Per-card latency | p50 and p95 over the fixture set | Reported, no hard gate |
| Decode speed | Output tokens / second | Reported |
| Peak memory | `/usr/bin/time -l` maximum resident set | ≤ 8 GB |
| Load time | Process start to ready | Reported |
| Determinism | Score range over three runs | Reported |

## Phases

### Phase A — smoke

1. Install `llama.cpp`.
2. Download `M1`, `M2`, `M3`, `M4`, `M6`. Add `M5` if approved.
3. For each model, run `S01-T01`, `S01-T02` and one `S01-T03` call.
4. Record load time, first-token time, decode speed and peak memory.
5. Drop a model only if it fails to start, runs out of memory or cannot produce JSON.

### Phase B — comparison

1. Run `S01-T03`…`S01-T07` on the five core fixtures.
2. Run two repetitions per fixture. Three for `F1` to measure variance.
3. Run `S01-T10` for the two fastest surviving models.
4. Compute the metrics and rank the models.
5. Write the recommendation for `Q1` and the default `LLM_MODEL`.

## Harness

A small Go program under `scripts/spikes/model-eval`:

- Reads `models.json`, `rubric.json` and `fixtures/*.json`.
- Calls the OpenAI-compatible endpoint with `response_format`.
- Validates the schema with a hand-written validator. No new dependency (`H-10`).
- Writes `results.jsonl` with one row per call.
- Writes a summary table with the metrics above.
- Never logs the key or the transcript. It logs fixture IDs and hashes.

The harness is spike code. It is not part of the product binary. It uses the same contract
shape as the future adapter, so a positive result transfers to `T-035`.

## Decision requested

| Decision | Default |
|----------|---------|
| Install `llama.cpp` with Homebrew | Approve |
| Download about 12 GB of models | Approve |
| Include `M5` (8B-A1B) | Approve, or drop if time is tight |
| Use synthetic fixtures and a spike rubric | Approve |
| Run the full matrix on the Intel laptop | Approve |

Without approval, `S-01` stays planned. There is no local runtime or model to measure
today.

## Output

`docs/30-implementation/spikes/s-01-model-comparison.md` is this plan. After the runs, a
results section is appended with the table and the recommended default model. `OQ-06` is
already closed by `S-02`. `Q1` and the `LLM_MODEL` default are the outputs of this spike.

## Next

The owner approves the runtime install, the model set and the fixtures. Then Phase A
starts.
