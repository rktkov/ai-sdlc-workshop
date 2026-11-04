# Architecture documentation backlog

> **Status:** active · **Last updated:** 2026-11-04
> **Scope:** composing the full system-design documentation suite for `ytfeed`.
>
> **Not the implementation backlog.** `DOC-11` reserves
> [`docs/30-implementation/backlog.md`](../30-implementation/backlog.md) as the single
> source of truth for code work. This file tracks *documentation* tasks only. The
> implementation backlog absorbs it at `ARCH-601` (`DOC-19`: a temporary document is
> deleted or folded, never left to rot).

## Counters

| Total | Done | In review | In progress | Blocked | Remaining |
|-------|------|-----------|-------------|---------|-----------|
| 39 | 39 | 0 | 0 | 0 | 0 |

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

## Documentation language — STE-100

`DOC-20` makes ASD-STE100 the documentation standard. All 29 documents are now written in
STE-100. The pre-STE originals are archived at
[`_archive/pre-ste-2026-11-03/`](../_archive/pre-ste-2026-11-03/). New documents are
written in STE from the first draft.

The `.ste.md` suffix was temporary. It is no longer used.

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
| SD-7 | Friends' instances | Not drawn at Level 1. One person, one system. Mentioned in the written summary only. | Owner |
| SD-8 | Artifact location | `docs/20-system-design/` per `DOC-1` and `DOC-2`, not the guide's `docs/architecture/` | Rules precedence |
| SD-9 | Transcript durability | Transcripts are staged in `./data/tmp/transcripts/` between fetch and analysis commit (crash-safe resume). They are deleted on commit and TTL-swept. They never enter the DB, logs, traces or backups. | Owner, 2026-11-02 |

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
| ARCH-107 | Data flow — batch pipeline: candidates → dedupe → transcript → analysis → scoring | `flows/batch-pipeline.md` | ARCH-105, ARCH-105a | M | `DONE` |
| ARCH-108 | Control flow — session lifecycle, weekly prompt, degradation paths | `flows/session-and-degradation.md` | ARCH-103, ARCH-106 | M | `DONE` |
| ARCH-109 | C4 consistency check against the `A-9` package tree | Review note appended to `c4/02-container-diagram.md` | ARCH-103…106 | S | `DONE` |

**Level 1 constraints (SD-5…SD-7):** the LLM runtime is drawn as an external system. The
privacy boundary text says "transcripts are not persisted by ytfeed", not "never leave the
machine". `yt-dlp` is external and drawn once, with a note that `feed` depends on it and
`library` does not. Only the owner actor is drawn.

**Gate:** the owner confirms each level before the next starts (`level_by_level`,
`confirm_each`). Level 4 is not produced (`no_code_level`).

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
different split, the task stops (`BLOCKED`) and produces an ADR plus a rules-PR proposal.
It does not silently redraw the boundary (`H-9`).

**Gate:** boundaries confirmed before Phase 3 (`confirm_boundaries`).

---

## Phase 3 — DDD tactical design

Full depth for the four core contexts. Light for the two supporting ones (`SD-3`,
`core_contexts_first`).

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-301 | `taste` — taste profile axes, signals, cold periods, aggregates + invariants | `ddd/tactical/taste/` | ARCH-205 | L | `DONE` |
| ARCH-302 | `feed` — session, card, ordering invariant, gesture outcomes | `ddd/tactical/feed/` | ARCH-205 | L | `DONE` |
| ARCH-303 | `discovery` — candidate pool, batch, quota ledger, micro-blogger constraint | `ddd/tactical/discovery/` | ARCH-205 | L | `DONE` |
| ARCH-304 | `analysis` — analysis result, failure states, rubric version binding | `ddd/tactical/analysis/` | ARCH-205 | L | `DONE` |
| ARCH-305 | `identity` — light: credential, subscription import, token lifecycle | `ddd/tactical/identity/` | ARCH-205 | S | `DONE` |
| ARCH-306 | `library` — light: saved video vs downloaded file (see `OQ` on the ambiguity) | `ddd/tactical/library/` | ARCH-205 | S | `DONE` |
| ARCH-307 | Evented pattern assessment; expected outcome: no CQRS, no event sourcing, one in-process process manager for batch resume | `ddd/tactical/evented-patterns.md` | ARCH-301…306 | M | `DONE` |

**Per-context acceptance:** every aggregate has at least one stated invariant. Value objects
have validation rules. Domain events are past-tense facts with payloads. Repository
contracts state transaction boundaries. No evented pattern is adopted without explicit
justification (`evented_gate`).

---

## Phase 4 — Architecture decision records

Six ADRs (`SD-2`). `DECISIONS.md` R-1…R-27 are adopted by reference and not duplicated
(`DOC-18`).

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-401 | ADR-0001 — Single binary, single process, client/server split over loopback HTTP | `adr/0001-single-binary-client-server.md` | ARCH-102, ARCH-103 | M | `DONE` |
| ARCH-402 | ADR-0002 — SQLite behind repository interfaces; durability and migration policy | `adr/0002-sqlite-repository-boundary.md` | ARCH-109 | M | `DONE` |
| ARCH-403 | ADR-0003 — Six bounded contexts as vertical slices in a layered architecture | `adr/0003-six-bounded-contexts.md` | ARCH-205 | M | `DONE` |
| ARCH-404 | ADR-0004 — Analysis contract: one OpenAI-compatible adapter, schema-constrained JSON, versioned rubric | `adr/0004-analysis-contract.md` | ARCH-304 | M | `DONE` |
| ARCH-405 | ADR-0005 — Explicit resumable pipeline; transcript-only ingestion; bounded transcript staging | `adr/0005-pipeline-and-transcripts.md` | ARCH-303, ARCH-304 | M | `DONE` |
| ARCH-406 | ADR-0006 — Degradation and public contracts: exit codes, cached fallback, local OTel traces | `adr/0006-degradation-and-contracts.md` | ARCH-108 | M | `DONE` |
| ARCH-407 | ADR index with the known gap: rules without an ADR | `adr/INDEX.md` | ARCH-401…406 | S | `DONE` |

**Per-ADR acceptance:** fixed `DOC-5` structure. At least two considered options. Real
negatives and risks. Cross-links. Status `Accepted`. Dated 2026-11-02.

---

## Phase 5 — Assembly and audit

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-501 | Master index: navigation, system overview, artifact status, gaps | `README.md` (in `20-system-design/`) | ARCH-101…407 | M | `DONE` |
| ARCH-502 | Executive summary, separate from the technical view | `executive-summary.md` | ARCH-501 | S | `DONE` |
| ARCH-503 | Traceability matrix: PRD F1–F28 → component → bounded context | `traceability.md` | ARCH-501 | M | `DONE` |
| ARCH-504 | Gap analysis, maintenance guide, session summary | Sections in `README.md` | ARCH-501 | M | `DONE` |
| ARCH-505 | Rules-consistency audit: every `A-*`, `D-*`, `CLI-*` rule placed or explicitly marked non-architectural | `audit/rules-coverage.md` | ARCH-501 | M | `DONE` |

**ARCH-503 acceptance:** any P0 requirement without a component and context is listed as a
gap, not omitted. `F28`, `F27` and `F23` are expected to map to transport or storage
components rather than domain contexts. That is a valid mapping, but it must be stated.

**ARCH-505 acceptance:** zero contradictions with `docs/00-rules/` unless linked to an ADR
and a proposed rules-PR note. Findings are reported, never applied (`H-9`, SD-4).

---

## Phase 6 — Handoff (outside this backlog's scope)

| ID | Task | Output | Deps | Size | Status |
|----|------|--------|------|------|--------|
| ARCH-601 | Propose the implementation backlog: epics E1–E6, Sprint-1 walking skeleton, spikes Q1/Q2/R3; fold this file into it and delete it (`DOC-19`) | `docs/30-implementation/backlog.md` + `tasks.md` | ARCH-505 | L | `DONE` |

`ARCH-601` is a **proposal** presented for approval. It is not executed here. `H-4` forbids
widening scope, and `AI-2` requires task IDs and acceptance criteria that do not exist yet.

---

## Definition of Done (every architecture task)

- [ ] The output file exists at the stated path, with kebab-case naming (`DOC-3`)
- [ ] Every diagram is Mermaid and renders without external tooling
- [ ] Every factual claim cites a PRD section, a rule ID, or an ADR. No unsourced prose.
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
| G5 | Tactical model confirmed? | Yes — 2026-11-04; seven models confirmed; no CQRS, no event sourcing, one in-process process manager | 2026-11-04 |
| G6 | ADR set confirmed? | Yes — 2026-11-04; six ADRs and the index confirmed; no seventh ADR | 2026-11-04 |
| Final | Assembly and audit confirmed? | Yes — 2026-11-04; all artifacts confirmed; `ARCH-601` remains a proposal | 2026-11-04 |

## Notes log

Append-only. One line per event, newest last.

- 2026-11-02 — Backlog created. Phase 0 complete except `ARCH-004`.
- 2026-11-02 — The guide's proposed `docs/architecture/` tree is rejected in favour of
  `docs/20-system-design/` because `DOC-1` wins on precedence. Recorded as SD-8 so a
  later session does not "correct" it back.
- 2026-11-02 — `ARCH-004` done: 15 open questions registered with provisional defaults.
- 2026-11-02 — `ARCH-101` drafted: C4 Level 1 written. Awaiting gate `G1`.
- 2026-11-02 — Gate `G1` passed. `ARCH-101` done. `ARCH-102` started.
- 2026-11-02 — `ARCH-102`: Level 2 exposes a packaging/runtime gap — one binary runs as
  two process roles (short-lived CLI client, long-lived API server). Recorded as `OQ-16`.
- 2026-11-02 — `ARCH-102`: `--json` (single JSON value) conflicts with incremental card
  streaming. Recorded as `OQ-17`.
- 2026-11-02 — `ARCH-105a` created on owner request: bounded transcript staging. `OQ-05`
  moved to Resolved. L1/L2 updated for consistency. PRD `F9`/Privacy-NFR amendment
  proposed in the component document §16, **not applied** (`H-9`).
- 2026-11-02 — Transcript-staging amendment **approved and applied** to the PRD (`F9`,
  Privacy NFR, `Q8`, status line). `ARCH-105a` done. Gate `G2` passed.
- 2026-11-02 — Level 3 drafted: `ARCH-103`…`ARCH-106` in `REVIEW`, awaiting gate `G3`.
- 2026-11-02 — `OQ-18` added: `F5` requires gestures to change the profile but the update
  rule is unspecified.
- 2026-11-03 — Gate `G3` passed. `ARCH-103`…`ARCH-106` done. `ARCH-107`/`108` drafted
  (data flow, control flow). `ARCH-109` done with 0 blocking contradictions, 2 additive
  package entries and `shared/` deferred to tactical design.
- 2026-11-03 — `handoff.md` written for a fresh agent session. Next task `ARCH-201`
  (DDD viability check).
- 2026-11-03 — `ARCH-201` drafted: DDD viability scored **3/4** ("DDD is appropriate").
  The 0–1 lightweight path does not apply. Decision and sensitivity recorded in
  `ddd/strategic/context-map.md` §1. Awaiting `G4`.
- 2026-11-03 — `ARCH-202` drafted: core = `taste`, `analysis`. Supporting = `discovery`,
  `feed`, `library`. Generic = `identity`. The differentiator is named as the
  rubric-judgment ↔ gesture-profile loop, not either context alone.
- 2026-11-03 — `ARCH-203` drafted: six contexts of `A-7` validated, none merged, split or
  added. Every runtime arrow points into `feed`. No `A-7` change is proposed. Two additive
  ownership notes: weekly rating (`feed`) and transcript staging state (`analysis`).
- 2026-11-03 — `ARCH-204` drafted: glossary built only from PRD/rule terms, with 18
  cross-context collisions. `OQ-19` (interests/topics/axes/topic tags) and `OQ-20`
  ("rubric weights" in `A-7`) added to the register.
- 2026-11-03 — `ARCH-205` prepared: the `G4` decision request is in
  `ddd/strategic/context-map.md` §10. Awaiting owner confirmation. Gate `G4` is **not**
  passed and no Phase 2 artifact is marked `DONE`.
- 2026-11-03 — `G4-4` confirmed: `identity` reclassified **generic → supporting** because
  the subscription graph is treated as a proprietary taste asset. `subdomain-map.md` and
  `context-map.md` (§3, §4, §8.1, §9) updated. Tactical depth stays **light** per `SD-3`.
  Class and depth are separate decisions.
- 2026-11-03 — `G4-6` confirmed: onboarding stays in `identity` as a use-case coordinator.
  No C4 edit. Gate `G4` remains open on the other seven rows (`G4-1`, `G4-2`, `G4-3`,
  `G4-5`, `G4-7`, `G4-8`, `G4-9`).
- 2026-11-03 — `G4-3` default reworded while recording the above: "all runtime arrows point
  into `feed`" was imprecise, because supplier edges also enter `discovery` and `analysis`.
  The accurate finding is that the map is acyclic and `feed` is its only sink. No diagram
  change. The relationship table is unchanged.
- 2026-11-03 — **Gate `G4` passed.** The owner confirmed `G4-1`, `G4-2`, `G4-3`, `G4-5`,
  `G4-7`, `G4-8` and `G4-9`. `G4-4` (supporting) and `G4-6` (keep) were confirmed earlier.
  `ARCH-201`…`ARCH-205` set to `DONE`. Counters `39 | 17 | 2 | 0 | 0 | 20`. `OQ-19` and
  `OQ-20` moved to Resolved in `open-questions.md`. Phase 3 starts at `ARCH-301` (taste).
- 2026-11-03 — `DOC-20` adopted: all documentation is written in ASD-STE100. All 29
  documents were converted to STE-100 and took the original file names. The pre-STE
  originals are archived at `docs/_archive/pre-ste-2026-11-03/`.
- 2026-11-03 — `ARCH-301` drafted: tactical model for `taste` in
  `ddd/tactical/taste/` (README, aggregates, value objects, events, repository, use
  cases). Three aggregates (`TasteProfile`, `Signal`, `ColdPeriod`); one gesture
  transaction; `D-301`…`D-309`; provisional weight rule (`OQ-18`). Awaiting gate `G5`.
- 2026-11-03 — `ARCH-302` drafted: tactical model for `feed` in
  `ddd/tactical/feed/` (README, aggregates, value objects, events, repository, use
  cases). Three aggregates (`Batch`, `Session`, `WeeklyRating`); card identity cursor;
  frozen seen prefix; `D-311`…`D-319`. Awaiting gate `G5`.
- 2026-11-03 — `ARCH-303` drafted: tactical model for `discovery` in
  `ddd/tactical/discovery/` (README, aggregates, events, repository). Two aggregates
  (`CandidatePool`, `QuotaLedger`); reserve-before-call; deterministic selection;
  `D-320`…`D-329`. Awaiting gate `G5`.
- 2026-11-03 — `ARCH-304` drafted: tactical model for `analysis` in
  `ddd/tactical/analysis/` (README, aggregates, events, repository). Two aggregates
  (`Analysis`, `TranscriptStage`); explicit failure states; rubric and profile versions
  bound; `D-330`…`D-339`. Awaiting gate `G5`.
- 2026-11-03 — `ARCH-305` drafted: light tactical model for `identity` in
  `ddd/tactical/identity/` (README, model). Two aggregates (`GoogleCredential`,
  `SubscriptionGraph`); token file only; `D-340`…`D-346`. Awaiting gate `G5`.
- 2026-11-03 — `ARCH-306` drafted: light tactical model for `library` in
  `ddd/tactical/library/` (README, model). One aggregate (`LibraryEntry`); saved versus
  downloaded; `D-350`…`D-356`. Awaiting gate `G5`.
- 2026-11-03 — `ARCH-307` drafted: `ddd/tactical/evented-patterns.md`. No CQRS, no
  event sourcing, no outbox; one in-process process manager for batch resume;
  `D-360`…`D-366`. Awaiting gate `G5`.
- 2026-11-03 — Phase 3 drafted: `ARCH-301`…`ARCH-307` in `REVIEW`. Gate `G5` is prepared
  for owner confirmation. No Phase 3 artifact is marked `DONE`.
- 2026-11-04 — **Gate `G5` passed.** The owner confirmed the seven tactical models,
  the aggregate boundaries, the gesture commit order, the analysis key, the quota rules
  and the evented decision. `ARCH-301`…`ARCH-307` set to `DONE`; counters
  `39 | 24 | 2 | 0 | 0 | 13`. Next task `ARCH-401` (ADR-0001).
- 2026-11-04 — `ARCH-401` drafted: `adr/0001-single-binary-client-server.md`. Four
  options, decision, negatives and risks. Awaiting gate `G6`.
- 2026-11-04 — `ARCH-402` drafted: `adr/0002-sqlite-repository-boundary.md`. Five
  options; durability and migration policy. Awaiting gate `G6`.
- 2026-11-04 — `ARCH-403` drafted: `adr/0003-six-bounded-contexts.md`. Five options;
  vertical slices inside three layers. Awaiting gate `G6`.
- 2026-11-04 — `ARCH-404` drafted: `adr/0004-analysis-contract.md`. One adapter,
  schema-constrained JSON, structured profile. Awaiting gate `G6`.
- 2026-11-04 — `ARCH-405` drafted: `adr/0005-pipeline-and-transcripts.md`. Bounded
  staging and resumable pipeline. Awaiting gate `G6`.
- 2026-11-04 — `ARCH-406` drafted: `adr/0006-degradation-and-contracts.md`. Explicit
  degradation, exit codes, cached feed and local traces. Awaiting gate `G6`.
- 2026-11-04 — `ARCH-407` drafted: `adr/INDEX.md`. Six ADRs, rule coverage and known
  gaps. Gate `G6` is prepared for owner confirmation. No ADR is marked `DONE`.
- 2026-11-04 — **Gate `G6` passed.** The owner confirmed ADR-0001…ADR-0006 and the
  index. `ARCH-401`…`ARCH-407` set to `DONE`; counters `39 | 31 | 2 | 0 | 0 | 6`.
  Next task `ARCH-501` (master index).
- 2026-11-04 — `ARCH-501` drafted: `20-system-design/README.md`. System overview,
  reading paths, artifact map, context summary, global invariants and known gaps.
  Awaiting the final gate.
- 2026-11-04 — `ARCH-502` drafted: `executive-summary.md`. One-page product and
  architecture summary without new claims. Awaiting the final gate.
- 2026-11-04 — `ARCH-503` drafted: `traceability.md`. All 28 P0 requirements mapped;
  three map to transport or storage. Awaiting the final gate.
- 2026-11-04 — `ARCH-504` drafted: gap analysis, maintenance guide and session summary
  in `20-system-design/README.md`. Awaiting the final gate.
- 2026-11-04 — `ARCH-505` drafted: `audit/rules-coverage.md`. 80 rules audited, 78
  placed and 2 non-architectural. Zero unplaced rules and zero contradictions.
  Awaiting the final gate.
- 2026-11-04 — **Final gate passed.** The owner confirmed the master index, the
  executive summary, the traceability matrix, the gap analysis and the rules audit.
  `ARCH-107`, `ARCH-108` and `ARCH-501`…`ARCH-505` set to `DONE`; counters
  `39 | 38 | 0 | 0 | 0 | 1`. `ARCH-601` remains a proposal.
- 2026-11-04 — `ARCH-601` drafted: `docs/30-implementation/backlog.md` and `tasks.md`.
  Epics E1–E6, Sprint 1, spikes Q1/Q2/R3 and 37 tasks. Status `PROPOSED`; awaiting
  owner approval before code work or the backlog fold.
- 2026-11-04 — `ARCH-601` approved by the owner, who started `T-001`. The
  implementation backlog `docs/30-implementation/backlog.md` is `ACTIVE`; `tasks.md`
  tracks `T-001`. The architecture backlog is kept as history until the owner confirms
  the fold and deletion (`DOC-19`).
- 2026-11-04 — `T-001` done: `go.mod`, Makefile, `.golangci.yml`, `.editorconfig`,
  `.gitattributes`, `.gitignore` and the `internal/foundation` scaffold. `make check` and
  `make coverage-check` pass.
- 2026-11-04 — `T-001a` done: contract and E2E harness, deterministic fakes, golden
  fixture and a local git repository. `make test-contract`, `make test-e2e` and
  `make check` pass.
- 2026-11-04 — `T-002` done: `internal/config` and `internal/pkg/logger`. Config
  validation lists every problem, secrets are redacted, logs are JSON or text. Happy and
  unhappy tests pass.
- 2026-11-04 — `T-003` done: `internal/storage` and `migrations/`. WAL, foreign keys,
  `synchronous=FULL`, one writer, embedded migration, backup before apply and a refusal
  for a newer schema. Happy and unhappy tests pass.
- 2026-11-04 — `T-004` done: `internal/transport/http`. Health handler, bearer token,
  loopback guard and a `0600` token file. Happy and unhappy tests pass.
- 2026-11-04 — `T-005` done: `cmd/ytfeed` and `internal/transport/cli`. `doctor` reaches
  the server, `serve` binds locally, `--allow-remote` warns, exit codes follow `CLI-6`.
  Happy and unhappy tests pass.
- 2026-11-04 — `T-006` done: `internal/domain/feed` and the CLI card renderer. Card
  validation and `CLI-13` order are tested. Happy and unhappy tests pass.
- 2026-11-04 — `T-007` done: `POST /v1/gestures` → `taste.Service` → SQLite. A left
  swipe commits a signal and a cold period in one transaction. A retry is idempotent.
  Sprint 1 code is complete. The spikes need owner resources and decisions.
- 2026-11-04 — `S-02` closed: `docs/30-implementation/spikes/s-02-opencode-api.md`. The
  local server API is session-based and experimental, not OpenAI-compatible. The owner
  put provider B out of v1; the managed endpoint stays a future preset only. `OQ-06`
  moved to Resolved.
- 2026-11-04 — `S-01` confirmed: `docs/30-implementation/spikes/s-01-model-comparison.md`.
  `LFM2.5-1.2B-Instruct QAD Q4_0` is the v1 default. `.env.example` and `internal/config`
  now use `lfm2.5-1.2b-instruct`. `2.6B` and the dense `3B` fail latency on the Intel
  laptop. `350M` fails quality.
- 2026-11-04 — `T-020` done: `taste.Profile`, validation, SQLite repository and migration
  v3. Happy and unhappy tests pass.
- 2026-11-04 — `T-021` done: signal scores, bounded ±0.05 weight update with clamp and
  migration v4. Happy and unhappy tests pass.
- 2026-11-04 — `T-022` done: cold-period validation, active query, find by target and
  refresh. Expired rows stay. Happy and unhappy tests pass.
- 2026-11-04 — `T-023` done: reason picker keys 1–4, Enter default, Esc other, scope
  preview and invalid-key error. Next task `T-024`.
