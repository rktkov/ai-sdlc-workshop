# Model comparison harness

Spike `S-01` needs one comparable call shape for every model. This program sends the same
prompt, rubric and schema to an OpenAI-compatible `/v1/chat/completions` endpoint.

## Usage

```sh
llama-server -m ../../.spike-models/LFM2.5-2.6B-QAD-Q4_0.gguf \
  --host 127.0.0.1 --port 8081 -c 8192 -np 1 -t 8 --jinja

go run . \
  -base-url http://127.0.0.1:8081/v1 \
  -model lfm2.5-2.6b-qad-q4_0 \
  -fixtures fixtures.json \
  -rubric rubric.json \
  -out results.jsonl \
  -repeat 2
```

The program appends one JSON object per call. It prints a summary at the end.

## What it measures

- JSON validity and required fields.
- Axis mean absolute error against the five core labels.
- Worth-watching agreement.
- p50 and p95 latency.
- JSON-mode support. The first call asks for `json_object`. If the server rejects it, the
  program retries without the field and records `json_mode=false`.

## Models

| ID | File |
|----|------|
| `M1` | `LFM2.5-350M-QAD-Q4_0.gguf` |
| `M2` | `LFM2.5-1.2B-Instruct-QAD-Q4_0.gguf` |
| `M3` | `LFM2.5-2.6B-QAD-Q4_0.gguf` |
| `M4` | `LFM2.5-2.6B-Q8_0.gguf` |
| `M5` | `LFM2.5-8B-A1B-Q4_K_M.gguf` |
| `M6` | `Llama-3.2-3B-Instruct-Q4_K_M.gguf` |

The files live in `execution/.spike-models/` and are not committed.

## Limitations

- Empty transcripts are skipped in this first harness pass. The explicit
  `no_transcript` path is covered by `S01-T06` in the spike plan and will be checked
  separately.
- Topic match is recorded but not used in the summary yet.
- The harness is spike code. It is not part of the product binary.
