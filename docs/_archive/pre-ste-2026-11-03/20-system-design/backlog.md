# Architecture documentation backlog

> **Status:** active · **Last updated:** 2026-11-03
> **Scope:** composing the full system-design documentation suite for `ytfeed`.
>
> **Not the implementation backlog.** `DOC-11` reserves
> [`docs/30-implementation/backlog.md`](../30-implementation/backlog.md) as the single
> source of truth for code work. This file tracks *documentation* tasks only and is
> absorbed into the implementation backlog at `ARCH-601` (`DOC-19`: a temporary
> document is deleted or folded, never left to rot).

## Counters

| Total | Done | In review | In progress | Blocked | Remaining |
|-------|------|-----------|-------------|---------|-----------|
| 39 | 17 | 2 | 0 | 0 | 20 |

## Status legend

| Status | Meaning |
|--------|---------|
| `TODO` | Not started |
| `IN PROGRESS` | Actively being written; only one row at a time |
| `REVIEW` | Output written, awaiting the owner's gate |
| `BLOCKED` | Cannot proceed; the blocking reason is in the Notes column |
| `DONE` | Output exists and the owner confirmed it |

## Size

`S` < 1 hour · `M` 1–3 hours · `L` > 3 hours or needs a confirmation round.

---

## Documentation language — STE-100 migration

`DOC-20` makes ASD-STE100 the documentation standard. Each source has an STE edition
beside it, `<name>.ste.md`. The pre-STE originals are archived at
[`_archive/pre-ste-2026-11-03/`](../_archive/pre-ste-2026-11-03/).

**Status: 29 of 29 migrated.**

| Source | STE edition |
|--------|-------------|
| [`README.md`](../00-rules/README.md) | [`README.ste.md`](../00-rules/README.ste.md) |
| [`01-go-language.md`](../00-rules/01-go-language.md) | [`01-go-language.ste.md`](../00-rules/01-go-language.ste.md) |
| [`02-architecture.md`](../00-rules/02-architecture.md) | [`02-architecture.ste.md`](../00-rules/02-architecture.ste.md) |
| [`03-errors.md`](../00-rules/03-errors.md) | [`03-errors.ste.md`](../00-rules/03-errors.ste.md) |
| [`04-data-persistence.md`](../00-rules/04-data-persistence.md) | [`04-data-persistence.ste.md`](../00-rules/04-data-persistence.ste.md) |
| [`05-config-and-secrets.md`](../00-rules/05-config-and-secrets.md) | [`05-config-and-secrets.ste.md`](../00-rules/05-config-and-secrets.ste.md) |
| [`06-logging-and-observability.md`](../00-rules/06-logging-and-observability.md) | [`06-logging-and-observability.ste.md`](../00-rules/06-logging-and-observability.ste.md) |
| [`07-testing.md`](../00-rules/07-testing.md) | [`07-testing.ste.md`](../00-rules/07-testing.ste.md) |
| [`08-cli-and-ux.md`](../00-rules/08-cli-and-ux.md) | [`08-cli-and-ux.ste.md`](../00-rules/08-cli-and-ux.ste.md) |
| [`09-git-workflow.md`](../00-rules/09-git-workflow.md) | [`09-git-workflow.ste.md`](../00-rules/09-git-workflow.ste.md) |
| [`10-docs-and-decisions.md`](../00-rules/10-docs-and-decisions.md) | [`10-docs-and-decisions.ste.md`](../00-rules/10-docs-and-decisions.ste.md) |
| [`11-ai-agent-workflow.md`](../00-rules/11-ai-agent-workflow.md) | [`11-ai-agent-workflow.ste.md`](../00-rules/11-ai-agent-workflow.ste.md) |
| [`DECISIONS.md`](../00-rules/DECISIONS.md) | [`DECISIONS.ste.md`](../00-rules/DECISIONS.ste.md) |
| [`prd-ytfeed.md`](../10-product/prd-ytfeed.md) | [`prd-ytfeed.ste.md`](../10-product/prd-ytfeed.ste.md) |
| [`flows/batch-pipeline.md`](flows/batch-pipeline.md) | [`flows/batch-pipeline.ste.md`](flows/batch-pipeline.ste.md) |
| [`flows/session-and-degradation.md`](flows/session-and-degradation.md) | [`flows/session-and-degradation.ste.md`](flows/session-and-degradation.ste.md) |
| [`open-questions.md`](open-questions.md) | [`open-questions.ste.md`](open-questions.ste.md) |
| [`c4/01-system-context.md`](c4/01-system-context.md) | [`c4/01-system-context.ste.md`](c4/01-system-context.ste.md) |
| [`c4/02-container-diagram.md`](c4/02-container-diagram.md) | [`c4/02-container-diagram.ste.md`](c4/02-container-diagram.ste.md) |
| [`c4/03-components/01-transport.md`](c4/03-components/01-transport.md) | [`c4/03-components/01-transport.ste.md`](c4/03-components/01-transport.ste.md) |
| [`c4/03-components/02-identity-taste.md`](c4/03-components/02-identity-taste.md) | [`c4/03-components/02-identity-taste.ste.md`](c4/03-components/02-identity-taste.ste.md) |
| [`c4/03-components/03-discovery-analysis.md`](c4/03-components/03-discovery-analysis.md) | [`c4/03-components/03-discovery-analysis.ste.md`](c4/03-components/03-discovery-analysis.ste.md) |
| [`c4/03-components/03a-transcript-staging.md`](c4/03-components/03a-transcript-staging.md) | [`c4/03-components/03a-transcript-staging.ste.md`](c4/03-components/03a-transcript-staging.ste.md) |
| [`c4/03-components/04-feed-library.md`](c4/03-components/04-feed-library.md) | [`c4/03-components/04-feed-library.ste.md`](c4/03-components/04-feed-library.ste.md) |
| [`ddd/strategic/context-map.md`](ddd/strategic/context-map.md) | [`ddd/strategic/context-map.ste.md`](ddd/strategic/context-map.ste.md) |
| [`ddd/strategic/subdomain-map.md`](ddd/strategic/subdomain-map.md) | [`ddd/strategic/subdomain-map.ste.md`](ddd/strategic/subdomain-map.ste.md) |
| [`ddd/strategic/glossary.md`](ddd/strategic/glossary.md) | [`ddd/strategic/glossary.ste.md`](ddd/strategic/glossary.ste.md) |
| [`backlog.md`](backlog.md) | [`backlog.ste.md`](backlog.ste.md) |
| [`handoff.md`](handoff.md) | [`handoff.ste.md`](handoff.ste.md) |

---

## Session decisions (frozen for this documentation effort)

| # | Decision | Value | Source |
|---|----------|-------|--------|
| SD-1 | Mode of work | From scratch | Entry gate |
| SD-2 | ADR scope | 6 architecture-shaping ADRs; `DECISIONS.md` R-1…R-27 adopted by reference, not duplicated | Owner |
| SD-3 | Tactical DDD depth | Full for `taste`, `feed`, `discovery`, `analysis`; light for `identity`, `library` | Owner |
| SD-4 | Contradiction handling | Record as open questions only; no resolution imposed, no rule edited | Owner |
| SD-5 | LLM trust boundary | Any `LLM_BASE_URL` allowed; LLM is an **external** system; privacy claim = "not persisted by ytfeed" | Owner |
| SD-6 | yt-dlp dependency | Required for `feed`; optional for `library`; absence degrades, never crashes (`C-11`) | Owner |
| SD-7 | Friends' instances | Not drawn at Level 1 — one person, one system; mentioned in the written summary only | Owner |
| SD-8 | Artifact location | `docs/20-system-design/` per `DOC-1`/`DOC-2`, not the guide's `docs/architecture/` | Rules precedence |
| SD-9 | Transcript durability | Transcripts are staged in `./data/tmp/transcripts/` between fetch and analysis commit (crash-safe resume), deleted on commit, TTL-swept, never in DB/logs/traces/backups | Owner, 2026-11-02 |

---

## Phase 0 — Setup and inputs

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-001 | Entry gate: collect input, confirm mode, confirm understanding | Confirmation block in session | — | S | `DONE` |
| ARCH-002 | Create this backlog | `docs/20-system-design/backlog.md` | ARCH-001 | S | `DONE` |
| ARCH-003 | Resolve Level 1 input decisions (LLM boundary, yt-dlp, instance scope) | SD-5…SD-7 above | ARCH-001 | S | `DONE` |
| ARCH-004 | Create the open-questions register from the C1–C10 analysis, plus any surfaced later | `open-questions.md` | ARCH-003 | M | `DONE` |

**ARCH-004 acceptance:** every entry has an ID, the conflicting sources, an owner, a
blocking flag, and the *provisional default used in diagrams* marked as provisional.
No entry proposes a rule edit (`H-9`, SD-4).

---

## Phase 1 — C4 architecture

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-101 | Level 1 — System context (Mermaid `C4Context` + summary + trust boundary) | `c4/01-system-context.md` | ARCH-003, ARCH-004 | M | `DONE` |
| ARCH-102 | Level 2 — Container diagram, adapted for a single-binary process | `c4/02-container-diagram.md` | ARCH-101 | M | `DONE` |
| ARCH-103 | Level 3 — Transport components (CLI dispatch, HTTP server, token, session lease) | `c4/03-components/01-transport.md` | ARCH-102 | M | `DONE` |
| ARCH-104 | Level 3 — `identity` + `taste` components | `c4/03-components/02-identity-taste.md` | ARCH-102 | M | `DONE` |
| ARCH-105 | Level 3 — `discovery` + `analysis` components | `c4/03-components/03-discovery-analysis.md` | ARCH-102 | M | `DONE` |
| ARCH-105a | Transcript staging component — crash-safe resume, delta on `analysis` ingestion | `c4/03-components/03a-transcript-staging.md` | ARCH-102 | M | `DONE` |
| ARCH-106 | Level 3 — `feed` + `library` components | `c4/03-components/04-feed-library.md` | ARCH-102 | M | `DONE` |
| ARCH-107 | Data flow — batch pipeline: candidates → dedupe → transcript → analysis → scoring | `flows/batch-pipeline.md` | ARCH-105, ARCH-105a | M | `REVIEW` |
| ARCH-108 | Control flow — session lifecycle, weekly prompt, degradation paths | `flows/session-and-degradation.md` | ARCH-103, ARCH-106 | M | `REVIEW` |
| ARCH-109 | C4 consistency check against the `A-9` package tree | Review note appended to `c4/02-container-diagram.md` | ARCH-103…106 | S | `DONE` |

**Level 1 constraints (SD-5…SD-7):** LLM runtime is drawn as an external system;
privacy boundary text says "transcripts are not persisted by ytfeed", not "never leave
the machine". `yt-dlp` is external and drawn once, with a note that `feed` depends on it
and `library` does not. Only the owner actor is drawn.

**Gate:** each level is confirmed by the owner before the next starts
(`level_by_level`, `confirm_each`). Level 4 is not produced (`no_code_level`).

---

## Phase 2 — DDD strategic design

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-201 | DDD viability check; record score and the lightweight-path decision | Section in `ddd/strategic/context-map.md` | ARCH-109 | S | `DONE` |
| ARCH-202 | Subdomain map (core / supporting / generic) | `ddd/strategic/subdomain-map.md` | ARCH-201 | M | `DONE` |
| ARCH-203 | Bounded context map + relationship patterns, validating the six contexts of `A-7` | `ddd/strategic/context-map.md` | ARCH-202 | M | `DONE` |
| ARCH-204 | Ubiquitous language glossary, terms drawn only from the PRD | `ddd/strategic/glossary.md` | ARCH-203 | M | `DONE` |
| ARCH-205 | Boundary confirmation gate | Owner sign-off recorded here | ARCH-203, ARCH-204 | S | `DONE` |

**ARCH-203 note:** `A-7` already fixes six contexts. If the modelling argues for a
different split, the task stops (`BLOCKED`) and produces an ADR plus a rules-PR
proposal — it does not silently redraw the boundary (`H-9`).

**Gate:** boundaries confirmed before Phase 3 (`confirm_boundaries`).

---

## Phase 3 — DDD tactical design

Full depth for the four core contexts; light for the two supporting ones (`SD-3`,
`core_contexts_first`).

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-301 | `taste` — taste profile axes, signals, cold periods, aggregates + invariants | `ddd/tactical/taste/` | ARCH-205 | L | `TODO` |
| ARCH-302 | `feed` — session, card, ordering invariant, gesture outcomes | `ddd/tactical/feed/` | ARCH-205 | L | `TODO` |
| ARCH-303 | `discovery` — candidate pool, batch, quota ledger, micro-blogger constraint | `ddd/tactical/discovery/` | ARCH-205 | L | `TODO` |
| ARCH-304 | `analysis` — analysis result, failure states, rubric version binding | `ddd/tactical/analysis/` | ARCH-205 | L | `TODO` |
| ARCH-305 | `identity` — light: credential, subscription import, token lifecycle | `ddd/tactical/identity/` | ARCH-205 | S | `TODO` |
| ARCH-306 | `library` — light: saved video vs downloaded file (see `OQ` on the ambiguity) | `ddd/tactical/library/` | ARCH-205 | S | `TODO` |
| ARCH-307 | Evented pattern assessment; expected outcome: no CQRS, no event sourcing, one in-process process manager for batch resume | `ddd/tactical/evented-patterns.md` | ARCH-301…306 | M | `TODO` |

**Per-context acceptance:** every aggregate has ≥ 1 stated invariant; value objects have
validation rules; domain events are past-tense facts with payloads; repository contracts
state transaction boundaries; no evented pattern is adopted without explicit
justification (`evented_gate`).

---

## Phase 4 — Architecture decision records

Six ADRs (`SD-2`). `DECISIONS.md` R-1…R-27 are adopted by reference and not duplicated
(`DOC-18`).

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-401 | ADR-0001 — Single binary, single process, client/server split over loopback HTTP | `adr/0001-single-binary-client-server.md` | ARCH-102, ARCH-103 | M | `TODO` |
| ARCH-402 | ADR-0002 — SQLite behind repository interfaces; durability and migration policy | `adr/0002-sqlite-repository-boundary.md` | ARCH-109 | M | `TODO` |
| ARCH-403 | ADR-0003 — Six bounded contexts as vertical slices in a layered architecture | `adr/0003-six-bounded-contexts.md` | ARCH-205 | M | `TODO` |
| ARCH-404 | ADR-0004 — Analysis contract: one OpenAI-compatible adapter, schema-constrained JSON, versioned rubric | `adr/0004-analysis-contract.md` | ARCH-304 | M | `TODO` |
| ARCH-405 | ADR-0005 — Explicit resumable pipeline; transcript-only ingestion; bounded transcript staging | `adr/0005-pipeline-and-transcripts.md` | ARCH-303, ARCH-304 | M | `TODO` |
| ARCH-406 | ADR-0006 — Degradation and public contracts: exit codes, cached fallback, local OTel traces | `adr/0006-degradation-and-contracts.md` | ARCH-108 | M | `TODO` |
| ARCH-407 | ADR index with the known gap: rules without an ADR | `adr/INDEX.md` | ARCH-401…406 | S | `TODO` |

**Per-ADR acceptance:** fixed `DOC-5` structure; at least two considered options; real
negatives and risks; cross-links; status `Accepted`; dated 2026-11-02.

---

## Phase 5 — Assembly and audit

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-501 | Master index: navigation, system overview, artifact status, gaps | `README.md` (in `20-system-design/`) | ARCH-101…407 | M | `TODO` |
| ARCH-502 | Executive summary, separate from the technical view | `executive-summary.md` | ARCH-501 | S | `TODO` |
| ARCH-503 | Traceability matrix: PRD F1–F28 → component → bounded context | `traceability.md` | ARCH-501 | M | `TODO` |
| ARCH-504 | Gap analysis, maintenance guide, session summary | Sections in `README.md` | ARCH-501 | M | `TODO` |
| ARCH-505 | Rules-consistency audit: every `A-*`, `D-*`, `CLI-*` rule placed or explicitly marked non-architectural | `audit/rules-coverage.md` | ARCH-501 | M | `TODO` |

**ARCH-503 acceptance:** any P0 requirement without a component and context is listed
as a gap, not omitted. `F28`, `F27`, `F23` are expected to map to transport/storage
components rather than domain contexts — that is a valid mapping, but it must be stated.

**ARCH-505 acceptance:** zero contradictions with `docs/00-rules/` unless linked to an
ADR and a proposed rules-PR note. Findings are reported, never applied (`H-9`, SD-4).

---

## Phase 6 — Handoff (outside this backlog's scope)

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-601 | Propose the implementation backlog: epics E1–E6, Sprint-1 walking skeleton, spikes Q1/Q2/R3; fold this file into it and delete it (`DOC-19`) | `docs/30-implementation/backlog.md` + `tasks.md` | ARCH-505 | L | `TODO` |

`ARCH-601` is a **proposal** presented for approval. It is not executed here, because
`H-4` forbids widening scope and `AI-2` requires task IDs and acceptance criteria that
do not exist yet.

---

## Definition of Done (every architecture task)

- [ ] The output file exists at the stated path, with kebab-case naming (`DOC-3`)
- [ ] Every diagram is Mermaid and renders without external tooling
- [ ] Every factual claim cites a PRD section, a rule ID, or an ADR — no unsourced prose
- [ ] Every open point references an `OQ-*` entry instead of an unstated assumption
- [ ] The owner has confirmed the artifact at its stage gate
- [ ] This backlog row is set to `DONE` and the counters are recalculated (`DOC-12`)
- [ ] No file under `docs/00-rules/` was modified (`H-9`)

## Gate log

| Gate | Question | Answer | Date |
|------|----------|--------|------|
| G0 | Mode of work? | From scratch (A) | 2026-11-02 |
| G0a | ADR scope? | 6 architecture-shaping ADRs | 2026-11-02 |
| G0b | Tactical depth? | Core 4 full, identity/library light | 2026-11-02 |
| G0c | Contradiction handling? | Open questions only | 2026-11-02 |
| G0d | Level 1 inputs? | LLM external, yt-dlp required for `feed`, one person one system | 2026-11-02 |
| G1 | Level 1 context diagram confirmed? | Yes — 2026-11-02 | 2026-11-02 |
| G2 | Level 2 container diagram confirmed? | Yes — 2026-11-02, with transcript staging (`OQ-05`) | 2026-11-02 |
| G3 | Level 3 component diagrams confirmed? | Yes — 2026-11-03 | 2026-11-03 |
| G4 | Bounded context boundaries confirmed? | Yes — 2026-11-03; six contexts, all nine decisions confirmed; `identity` supporting; `OQ-19`/`OQ-20` resolved | 2026-11-03 |
| G5 | Tactical model confirmed? | — | — |
| G6 | ADR set confirmed? | — | — |

## Notes log

Append-only. One line per event, newest last.

- 2026-11-02 — Backlog created. Phase 0 complete except `ARCH-004`.
- 2026-11-02 — The guide's proposed `docs/architecture/` tree is rejected in favour of
  `docs/20-system-design/` because `DOC-1` wins on precedence. Recorded as SD-8 so a
  later session does not "correct" it back.
- 2026-11-02 — `ARCH-004` done: 15 open questions registered with provisional defaults.
- 2026-11-02 — `ARCH-101` drafted: C4 Level 1 written. Awaiting gate `G1`.
- 2026-11-02 — Gate `G1` passed. `ARCH-101` done; `ARCH-102` started.
- 2026-11-02 — `ARCH-102`: Level 2 exposes a packaging/runtime gap — one binary runs as
  two process roles (short-lived CLI client, long-lived API server). Recorded as `OQ-16`.
- 2026-11-02 — `ARCH-102`: `--json` (single JSON value) conflicts with incremental card
  streaming. Recorded as `OQ-17`.
- 2026-11-02 — `ARCH-105a` created on owner request: bounded transcript staging. `OQ-05`
  moved to Resolved. L1/L2 updated for consistency. PRD `F9`/Privacy-NFR amendment
  proposed in the component document §16, **not applied** (`H-9`).
- 2026-11-02 — Transcript-staging amendment **approved and applied** to the PRD (`F9`,
  Privacy NFR, `Q8`, status line). `ARCH-105a` done; gate `G2` passed.
- 2026-11-02 — Level 3 drafted: `ARCH-103`…`ARCH-106` in `REVIEW`, awaiting gate `G3`.
- 2026-11-02 — `OQ-18` added: `F5` requires gestures to change the profile but the update
  rule is unspecified.
- 2026-11-03 — Gate `G3` passed. `ARCH-103`…`ARCH-106` done. `ARCH-107`/`108` drafted
  (data flow, control flow); `ARCH-109` done with 0 blocking contradictions, 2 additive
  package entries and `shared/` deferred to tactical design.
- 2026-11-03 — `handoff.md` written for a fresh agent session. Next task `ARCH-201`
  (DDD viability check).
- 2026-11-03 — `ARCH-201` drafted: DDD viability scored **3/4** ("DDD is appropriate");
  the 0–1 lightweight path does not apply. Decision and sensitivity recorded in
  `ddd/strategic/context-map.md` §1. Awaiting `G4`.
- 2026-11-03 — `ARCH-202` drafted: core = `taste`, `analysis`; supporting = `discovery`,
  `feed`, `library`; generic = `identity`. The differentiator is named as the
  rubric-judgment ↔ gesture-profile loop, not either context alone.
- 2026-11-03 — `ARCH-203` drafted: six contexts of `A-7` validated, none merged, split or
  added; every runtime arrow points into `feed`. No `A-7` change is proposed. Two additive
  ownership notes: weekly rating (`feed`) and transcript staging state (`analysis`).
- 2026-11-03 — `ARCH-204` drafted: glossary built only from PRD/rule terms, with 18
  cross-context collisions. `OQ-19` (interests/topics/axes/topic tags) and `OQ-20`
  ("rubric weights" in `A-7`) added to the register.
- 2026-11-03 — `ARCH-205` prepared: the `G4` decision request is in
  `ddd/strategic/context-map.md` §10. Awaiting owner confirmation; gate `G4` is **not**
  passed and no Phase 2 artifact is marked `DONE`.
- 2026-11-03 — `G4-4` confirmed: `identity` reclassified **generic → supporting** because
  the subscription graph is treated as a proprietary taste asset. `subdomain-map.md` and
  `context-map.md` (§3, §4, §8.1, §9) updated. Tactical depth stays **light** per `SD-3`;
  class and depth are separate decisions.
- 2026-11-03 — `G4-6` confirmed: onboarding stays in `identity` as a use-case coordinator;
  no C4 edit. Gate `G4` remains open on the other seven rows (`G4-1`, `G4-2`, `G4-3`,
  `G4-5`, `G4-7`, `G4-8`, `G4-9`).
- 2026-11-03 — `G4-3` default reworded while recording the above: "all runtime arrows point
  into `feed`" was imprecise, because supplier edges also enter `discovery` and `analysis`.
  The accurate finding is that the map is acyclic and `feed` is its only sink. No diagram
  change; the relationship table is unchanged.
- 2026-11-03 — **Gate `G4` passed.** The owner confirmed `G4-1`, `G4-2`, `G4-3`, `G4-5`,
  `G4-7`, `G4-8` and `G4-9`; `G4-4` (supporting) and `G4-6` (keep) were confirmed earlier.
  `ARCH-201`…`ARCH-205` set to `DONE`; counters `39 | 17 | 2 | 0 | 0 | 20`. `OQ-19` and
  `OQ-20` moved to Resolved in `open-questions.md`. Phase 3 starts at `ARCH-301` (taste).
