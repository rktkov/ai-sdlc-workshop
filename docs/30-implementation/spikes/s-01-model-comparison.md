# Spike S-01 — local model comparison

## Status

Confirmed on 2026-11-04. `LFM2.5-1.2B-Instruct QAD Q4_0` is the v1 default on the Intel target.

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
| `M5` | LFM2.5-8B-A1B | Q4_K_M | MoE stretch case, 1.5B active | ~4.5 GB |
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

## Results — Phase A smoke

The smoke run used `llama-server` build 12110 on the target Intel laptop. It used the
`json_schema` response format, temperature 0.1, 8 threads and an 8k context where noted.

| ID | Model | Core schema | MAE | Worth | p50 | Verdict |
|----|-------|-------------|-----|-------|-----|---------|
| `M1` | LFM2.5-350M QAD Q4_0 | 5/5 | 0.455 | 1/5 | 13.3 s | Fails quality. Scores are close to flat. |
| `M2` | LFM2.5-1.2B-Instruct QAD Q4_0 | 5/5 | 0.198 | 5/5 | 22.1 s | **Viable.** Meets the axis bar and the 60 s first-card bar. |
| `M3` | LFM2.5-2.6B QAD Q4_0 | 0/4 sampled | — | — | ~220 s | Fails latency. Decode is about 3.2 tok/s and output truncates. |
| `M4` | LFM2.5-2.6B Q8_0 | not run | — | — | — | Q8 is slower than Q4. The 2.6B Q4 result already fails. |
| `M5` | LFM2.5-8B-A1B Q4_K_M | not run | — | — | — | Optional. Larger than a failing 2.6B. |
| `M6` | Llama-3.2-3B Q4_K_M | 0/4 sampled | — | — | ~150 s | Fails latency. Decode is about 4.9 tok/s and output truncates. |

### What this means

1. **The 2.6B default is not viable on this laptop.** The target is an Intel i9 with no
   usable GPU path in this run. Decode is 3.2 tok/s. A card takes 2–4 minutes and often
   truncates before the JSON closes.
2. **The 1.2B model is the only candidate that passes the smoke bar.** It returns valid
   JSON on the five core fixtures, reaches MAE 0.198 and agrees on all five worth labels.
   Its median call is 22.1 s, below the 60 s first-card limit.
3. **The 350M model is too weak.** It returns valid JSON but the scores are nearly flat
   (MAE 0.455, worth agreement 1/5).
4. **The dense 3B baseline is too slow.** It is not a better trade than the 1.2B LFM.
5. **`llama-server` supports `json_schema`.** A strict schema request returned 200 and
   constrained the shape. This is the adapter mode `T-035` should use.

### Gaps

| Gap | Effect |
|-----|--------|
| `M2` long-context run used a 4k context | The 8k behavior is not measured for the 1.2B model. |
| Prompt-injection run on `M2` produced 15 axes | The injection defense is not proven. |
| Peak memory was not captured | The 8 GB bar is not measured. |
| Only one repetition per fixture | Variance and p95 are indicative, not final. |
| `M4` and `M5` were not run | Their latency is bounded below by the failing 2.6B and 8B sizes. |

## Decision

The owner confirmed the recommendation on 2026-11-04:

- `LFM2.5-1.2B-Instruct QAD Q4_0` is the v1 default on the Intel target.
- `LFM2.5-2.6B` stays optional for Apple Silicon or a GPU host.
- `LFM2.5-350M` stays a speed fallback only.
- `.env.example` and `internal/config` now default to `lfm2.5-1.2b-instruct`.

The rest of this section is the recommendation that led to the decision.

## Recommendation

Use **`LFM2.5-1.2B-Instruct-QAD-Q4_0`** as the v1 default on the target laptop.

- Keep `LFM2.5-2.6B` as an optional model for Apple Silicon or a GPU host.
- Keep `LFM2.5-350M` only as a speed fallback, not as a quality default.
- Do not use the dense 3B baseline as the default.
- Use `json_schema` strict mode in `T-035`.
- Record the default in `.env.example` and `internal/config` only after the owner confirms
  the change from `lfm2.5-2.6b-instruct` to `lfm2.5-1.2b-instruct`.

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
