# Spike S-02 — opencode API surface

## Status

Partially executed. Blocked on a paid OpenCode API key.

## Question

What is the opencode server API surface, and is it stable enough to be provider B?

## What we verified on 2026-11-04

| Check | Result |
|-------|--------|
| Local opencode installation | `opencode v2.2.5` at `~/.opencode/bin/opencode` |
| Local server | Running on `127.0.0.1:49374`, PID 26471 |
| Direct HTTP without opencode auth | `401 Unauthorized` |
| OpenAPI spec | `opencode api GET /openapi.json` returns 138 paths and 363 schemas |
| Spec stability label | `"Experimental HttpApi surface for selected instance routes."` |
| OpenAI chat path in the spec | **None**. No `/v1/chat/completions` path. |
| Session API | `/api/session`, `/api/session/{id}/prompt`, `/api/session/{id}/model` |
| Provider list | `/api/provider` shows an OpenAI-compatible OpenCode provider |
| OpenCode inference base URL | `https://opencode.ai/inference/openai/v1` |
| `POST /chat/completions` with an empty body | `400` — the endpoint validates the OpenAI shape |
| Free model request | `429 FreeUsageLimitError` — the free limit is exhausted |
| Paid model without a key | `401 AuthError: Missing API key` |

## Findings

1. **The local server is available, but it is not an OpenAI-compatible chat API.**
   It is a session-based HTTP API. It exposes prompts, messages, models and providers. It
   does not expose `/v1/chat/completions`. Using it as provider B would require a second
   adapter. That contradicts `A-12a` and the one-adapter decision in `ADR-0004`.

2. **The API is marked experimental.** The spec description says "Experimental HttpApi
   surface for selected instance routes." It is not a stable integration contract. A
   provider B built on it would carry schema churn and version risk.

3. **OpenCode's managed provider is OpenAI-compatible.** The provider list points at
   `https://opencode.ai/inference/openai/v1`. The chat path exists and validates the OpenAI
   request shape. This can be a **preset** for the existing adapter, not a second adapter.

4. **The free tier is not dependable.** The free model returned `429` because the limit is
   exhausted. A paid model without a key returns `401 Missing API key`.

5. **Availability now:** the local server is available for inspection. Provider B is not
   usable now. It needs a paid OpenCode key, or it stays out of v1.

## Recommendation

Keep the `OQ-06` default:

- **Preset only**, using the OpenAI-compatible OpenCode inference endpoint.
- **No second adapter.**
- **No dependency on the local opencode server API.**
- **Provider B stays out of v1** until the owner supplies a key and the paid endpoint
  passes the contract tests.

This preserves `A-12a`, `ADR-0004` and the single-adapter design. The local server remains
useful for the OpenCode product, not for `ytfeed`.

## Spike plan

| Step | Action | State |
|------|--------|-------|
| `P1` | Confirm the local opencode version, process and port | Done |
| `P2` | Read the server documentation and the OpenAPI spec | Done |
| `P3` | Check for an OpenAI-compatible path in the server spec | Done — absent |
| `P4` | Inspect the provider configuration for an OpenAI-compatible base URL | Done |
| `P5` | Probe the managed endpoint with no key and with a free model | Done — 401 and 429 |
| `P6` | Probe the managed endpoint with a paid key and a JSON schema request | Blocked — needs key |
| `P7` | Measure schema validity, latency, error mapping and cost on a small fixture set | Blocked |
| `P8` | Write the provider B recommendation and close `OQ-06` | Blocked |

## How we will test it

### Test cases

| ID | Test | Pass condition | State |
|----|------|----------------|-------|
| `S02-T01` | Server info | `/api/info` returns version `2.2.5` and the expected URL | Pass |
| `S02-T02` | Spec shape | The spec has no `/v1/chat/completions` path | Pass |
| `S02-T03` | Managed endpoint reachable | `POST /chat/completions` returns a shape error, not a network error | Pass |
| `S02-T04` | Missing key | A paid model returns `401` | Pass |
| `S02-T05` | Free limit | A free model returns a clear `429`, not a hang | Pass |
| `S02-T06` | Paid key, valid request | Returns an OpenAI-shaped `choices[0].message.content` | Blocked |
| `S02-T07` | JSON schema | A schema-constrained request returns valid JSON | Blocked |
| `S02-T08` | Error mapping | `401`, `429`, `400` and timeout map to product states | Blocked |
| `S02-T09` | Latency and cost | p50 and p95 latency plus units per call recorded | Blocked |
| `S02-T10` | Stability | Version and spec hash recorded for a second run | Pending |

### Test fixture

- One short transcript, about 200 words, from the repository or a synthetic text.
- The rubric v1 axes from `Q3` once available.
- A schema with one topic field, one score per axis and one justification.
- No real user data, no secrets in prompts and no transcript in logs.

### Commands

The probe uses the existing OpenAI-compatible adapter shape:

```sh
curl -sS -o response.json -w '%{http_code}\n' \
  -X POST "$OPENCODE_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $OPENCODE_API_KEY" \
  -H 'Content-Type: application/json' \
  -d @request.json
```

`request.json` carries the model, the messages, `response_format` and `max_tokens`. The
key comes from the environment, never from a file in the repository (`C-2`).

### Pass criteria for provider B

Provider B is viable only when all of these hold:

1. `S02-T06` and `S02-T07` pass with the existing adapter and no code branch.
2. The endpoint is reachable from the owner's machine with a documented key.
3. The error mapping is explicit and stable.
4. The cost and privacy change is acceptable (`SD-5`).
5. The local opencode server is not required at runtime.

If any criterion fails, provider B stays out of v1 and `OQ-06` stays at "preset only, out
of v1".

## Blockers

| Blocker | Owner action |
|---------|--------------|
| Paid OpenCode API key | Provide a key, or confirm provider B is out of v1. |
| Model choice | Confirm the model id to test, or accept the default from `S-01`. |
| Cost ceiling | Confirm a small test budget. |
| Rubric v1 axes | `Q3` content is still open. A synthetic axis set is enough for the contract test. |

## Decision requested

Choose one:

1. **Keep provider B out of v1.** No key, no adapter. `OQ-06` stays "preset only".
2. **Test the paid preset.** Provide an OpenCode API key and a model id. I will run
   `S02-T06`…`S02-T09` and report.
3. **Use the local server API.** This requires a second adapter and an ADR. I recommend
   against it.

## Artifacts

- This report: `docs/30-implementation/spikes/s-02-opencode-api.md`.
- The OpenAPI snapshot is not committed. It is 138 paths and changes with the server
  version.
- The probe results above are the evidence for the current state.

## Next

The owner decision above. Until then, `S-02` is `BLOCKED` and `S-01` / `S-03` remain open.
