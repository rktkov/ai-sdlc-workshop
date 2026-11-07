# Architecture decision record index

## Status

Accepted at gate `G6` (2026-11-04).

## Date

2026-11-02

## Scope

`SD-2` fixes the ADR set at six architecture-shaping decisions (`0001` to `0006`).
`DECISIONS.md` R-1…R-27 are adopted by reference. They are not duplicated here (`DOC-18`).

## Decisions

| ADR | Title | Status | One-line decision |
|-----|-------|--------|-------------------|
| [`0001`](0001-single-binary-client-server.md) | Single binary with a client/server split over loopback HTTP | Accepted | One executable runs as a short-lived CLI client and a long-lived local API server over `127.0.0.1`. |
| [`0002`](0002-sqlite-repository-boundary.md) | SQLite behind repository interfaces | Accepted | SQLite is the v1 store. SQL lives in the repository adapter. The domain owns the interfaces. |
| [`0003`](0003-six-bounded-contexts.md) | Six bounded contexts as vertical slices | Accepted | The six `A-7` contexts are package-level slices inside the three layers. Contracts are published interfaces. |
| [`0004`](0004-analysis-contract.md) | The analysis contract | Accepted | One OpenAI-compatible adapter, schema-constrained JSON, structured profile, versioned rubric. |
| [`0005`](0005-pipeline-and-transcripts.md) | Resumable pipeline with bounded transcript staging | Accepted | The pipeline resumes from checkpoints. A transcript is staged briefly in a restricted directory and deleted on commit. |
| [`0006`](0006-degradation-and-contracts.md) | Degradation, public contracts and local traces | Accepted | Failures degrade explicitly with stable exit codes, cached cards and local OTel traces. |

## Rule coverage

| ADR | Rules it carries |
|-----|------------------|
| `0001` | `A-9`, `A-10`, `A-10a`, `A-10b`, `A-11`. `R-9`, `R-10` |
| `0002` | `A-4`, `A-5`, `A-15`. `D-1`…`D-24` |
| `0003` | `A-1`, `A-7`, `A-8`, `A-9`, `A-14` |
| `0004` | `A-12`, `A-12a`, `A-12b`, `A-19`, `A-20` |
| `0005` | `A-14`. PRD `F9`, `F25`. `OQ-05` |
| `0006` | `CLI-6`, `CLI-7`, `CLI-22`. `F24`. `D-11b`. `L-15` |

## Rules without an ADR

These rules are enforced. They do not need an ADR because they are conventions or process
rules, not architecture-shaping choices. The rule file is the contract.

| Rule group | Why no ADR | Where enforced |
|------------|------------|----------------|
| `E-1`…`E-17` error handling | A local convention. No alternative architecture is in play. | `03-errors.md` |
| `C-1`…`C-17` config and secrets | A security convention. The storage boundary is in `ADR-0002`. | `05-config-and-secrets.md` |
| `L-1`…`L-18` logging and traces | A convention, except `L-15`. The trace decision is in `ADR-0006`. | `06-logging-and-observability.md` |
| `T-1`…`T-24` testing | A process standard. No architecture fork. | `07-testing.md` |
| `GH-1`…`GH-21` git workflow | A process standard. | `09-git-workflow.md` |
| `DOC-1`…`DOC-20` documentation | A process standard. | `10-docs-and-decisions.md` |
| `AI-1`…`AI-15` agent workflow | A process standard. | `11-ai-agent-workflow.md` |
| `CLI-1`…`CLI-28` CLI and UX | Interaction rules. `CLI-6` and `CLI-22` are carried by `ADR-0006`. | `08-cli-and-ux.md` |

## Known gaps

| Gap | Effect | Follow-up |
|-----|--------|-----------|
| No ADR for the error taxonomy | `E-*` is a convention, not an architecture choice. | None in v1. Revisit if errors cross a service boundary. |
| No ADR for the CLI rendering model | The renderer is transport code. | None in v1. |
| No ADR for the test strategy | `T-*` is a process rule. | None in v1. |
| `DECISIONS.md` R-1…R-27 are not individual ADRs | `SD-2` adopts them by reference. | Keep the reference. Do not duplicate. |
| `OQ-01` quota exit code is provisional | `ADR-0006` cites it. | Resolve before implementation of the quota path. |
| `OQ-07` rubric location is provisional | `ADR-0004` cites it. | Resolve before the rubric is loaded. |

## Reading order

1. `0001` for the runtime shape.
2. `0002` for storage.
3. `0003` for the domain layout.
4. `0004` and `0005` for the core pipeline.
5. `0006` for failure and public contracts.

## Related documents

- [`ddd/strategic/context-map.md`](../ddd/strategic/context-map.md) — boundaries and contracts.
- [`ddd/tactical/evented-patterns.md`](../ddd/tactical/evented-patterns.md) — evented decision.
- [`c4/02-container-diagram.md`](../c4/02-container-diagram.md) — containers and package mapping.
- [`docs/00-rules/DECISIONS.md`](../../00-rules/DECISIONS.md) — R-1…R-27.
