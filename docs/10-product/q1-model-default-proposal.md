# Proposed PRD amendment — Q1 model default

## Status

Proposed. Not applied. The owner must approve the exact diff.

## Why

Spike `S-01` measured the LFM2.5 family on the target Intel laptop. The 2.6B default
misses the 60 s first-card bar. The 1.2B model passes the schema, rubric and latency
smoke. The owner confirmed `LFM2.5-1.2B-Instruct QAD Q4_0` as the v1 default on
2026-11-04.

This amendment records that answer in the PRD. It changes the Q1 answer only. It does not
change a functional requirement.

## Exact diff

```diff
--- a/docs/10-product/prd-ytfeed.md
+++ b/docs/10-product/prd-ytfeed.md
@@ -383,7 +383,7 @@
-| Q1 | Which LFM2.5 size/quant clears the bar on the target laptop (schema validity, rubric agreement, tok/s, RAM)? | Owner + agent | **Yes**. It determines the default `LLM_MODEL` |
+| Q1 | Which LFM2.5 size/quant clears the bar on the target laptop (schema validity, rubric agreement, tok/s, RAM)? | Owner + agent | **Answered 2026-11-04.** `LFM2.5-1.2B-Instruct QAD Q4_0` is the v1 default on the Intel target. `2.6B` and the dense `3B` fail the latency bar. `350M` fails quality. |
```

## Consequences

- `.env.example` and `internal/config` already use `lfm2.5-1.2b-instruct`.
- `LFM2.5-2.6B` remains optional for Apple Silicon or a GPU host.
- `LFM2.5-350M` remains a speed fallback only.
- The full evidence is in
  [`../30-implementation/spikes/s-01-model-comparison.md`](../30-implementation/spikes/s-01-model-comparison.md).

## Apply

On approval, apply the one-line change above. No other PRD line moves.
