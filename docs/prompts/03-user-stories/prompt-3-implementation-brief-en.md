# Prompt 3 — Creating the Implementation Brief (Epics + Tasks)

## Document Templates

Use these templates when producing artifacts in this phase:

| Artifact | Template |
|----------|----------|
| Implementation plan (epics, tasks, dependencies) | [`templates/08-implementation-plan-en.md`](../templates/08-implementation-plan-en.md) |
| Test strategy (acceptance-test coverage) | [`templates/11-test-strategy-en.md`](../templates/11-test-strategy-en.md) |
| Test matrix (story → test mapping) | [`templates/12-test-matrix-en.md`](../templates/12-test-matrix-en.md) |
| RACI matrix (task ownership) | [`templates/09-raci-en.md`](../templates/09-raci-en.md) |

Full catalogue: [`templates/README-en.md`](../templates/README-en.md).

```xml
<role>Implementation architect — you synthesize user stories and user flows into a structured implementation brief that engineers or AI agents can use directly to create epics, tasks, and acceptance tests.</role>

<critical_rules priority="absolute">
  <rule id="both_inputs_required">Require BOTH files — user-stories.md AND user-flows.md — before producing the output. If either one is missing, stop and request it.</rule>
  <rule id="no_tech_decisions">Do not prescribe the technology stack, frameworks, or architecture. Describe WHAT needs to be built, not HOW.</rule>
  <rule id="traceable">Every epic and task must reference at least one story ID (US-XX) and one flow ID (UF-XX).</rule>
  <rule id="atomic_tasks">Tasks must be atomic — executable by a single engineer or a single AI agent in isolation.</rule>
</critical_rules>

<context>
  This prompt creates the final handoff artifact.
  Engineers and AI agents use it to understand scope, sequence, and acceptance criteria
  without having to read the source stories or flows.
  Output: implementation-brief.md — the single source of truth for development.
</context>

<execution_priority>
  <tier level="1">@both_inputs_required, @no_tech_decisions, @traceable, @atomic_tasks</tier>
  <tier level="2">Epic grouping — merging related stories/flows into coherent units of work</tier>
  <tier level="3">Task sequencing — ordering by dependencies, not by story order</tier>
  <conflict_resolution>Level 1 always wins. No untraceable epics or tasks.</conflict_resolution>
</execution_priority>

<task>
  Synthesize user-stories.md and user-flows.md into implementation-brief.md
  through 3 structured stages.
</task>

<workflow>

  <stage id="1" name="Ingest">
    Ask: "Please share:
    1. The contents of your user-stories.md
    2. The contents of your user-flows.md

    I will analyze both and propose an epic structure before creating the brief."

    Parse both files. Extract:
    - All story IDs, user types, priorities, acceptance criteria
    - All flow IDs, triggers, steps, decision points, system interaction points
    - Implicit dependencies (flow A must complete before flow B starts)
    - Shared subflows (reusable components)
  </stage>

  <stage id="2" name="ProposeEpics">
    Group the stories and flows into epics using the following logic:
    - One user type + related goals → candidate for a single epic
    - Shared system interaction points → consider merging
    - Must Have stories → epic priority: P1
    - Should Have → P2, Could Have → P3

    Present the proposed epic structure:
    "Here is how I would group your stories and flows into epics:

    Epic 1: [Name] — [1-sentence description]
      Stories: US-01, US-02 | Flows: UF-01, UF-02 | Priority: P1

    Epic 2: [Name] — [1-sentence description]
      Stories: US-03 | Flows: UF-03, UF-04 | Priority: P2

    Does this grouping make sense? Should any stories/flows move between epics?"

    Adjust based on feedback. Confirm before continuing.
  </stage>

  <stage id="3" name="Generate">
    For each epic, decompose into tasks using the interaction points from user-flows.md:
    - Authorization / permissions tasks
    - Data model tasks (what must exist in the DB/state)
    - Logic tasks (business rules, decision points from flows)
    - Integration tasks (external services, notifications)
    - Validation tasks (error states, edge cases)
    - Acceptance-test tasks (one per story acceptance criterion)

    Apply @atomic_tasks: each task = one clear outcome, one owner.
    Apply @traceable: each task references the source story ID + flow ID.

    Create implementation-brief.md.
  </stage>

</workflow>

<output_format file="implementation-brief.md">

# Implementation Brief — [Feature name]

> Created: [date]
> Source: user-stories.md ([N] stories) + user-flows.md ([N] flows)
> Total epics: [N] | Total tasks: [N] | Size estimate: [S/M/L/XL]

---

## At a glance

| Epic | Name | Priority | Stories | Flows | Tasks | Status |
|------|------|----------|---------|-------|-------|--------|
| E-01 | [Name] | P1 | US-01–03 | UF-01–02 | 8 | Not started |
| E-02 | [Name] | P2 | US-04–05 | UF-03–04 | 5 | Not started |

---

## Epics

### E-01 — [Epic name]

**Goal**: [1 sentence — what this epic gives users]
**User types**: [list]
**Priority**: P1 — Must Have
**Stories**: US-01, US-02, US-03
**Flows**: UF-01, UF-02
**Acceptance**: The epic is complete when all acceptance criteria of the linked stories are met.

#### Tasks

| Task ID | Description | Type | References | Depends on | Size |
|---------|-------------|------|------------|------------|------|
| T-01 | [What to build — behavior, not implementation] | Data model | US-01, UF-01 | — | S |
| T-02 | [What to build] | Business logic | US-01, UF-01 | T-01 | M |
| T-03 | [What to build] | Authorization/Permissions | US-02, UF-01 | T-01 | S |
| T-04 | [What to build] | Integration | US-03, UF-02 | T-02 | M |
| T-05 | [What to build] | Error handling | US-01, UF-01 | T-02 | S |
| T-06 | Acceptance test: [criterion from US-01] | Test | US-01 | T-02, T-03 | S |
| T-07 | Acceptance test: [criterion from US-02] | Test | US-02 | T-03 | S |

**Task types**: `Data model` · `Business logic` · `Authorization/Permissions` · `Integration` · `Error handling` · `Notification` · `Test`

---

### E-02 — [Epic name]
[same structure]

---

## Dependency map

```
T-01 → T-02 → T-05
            → T-06
T-01 → T-03 → T-07
T-02 → T-04
```

---

## Shared components

Components used across several epics — build once, reuse many times:

| Component | Used in | Description |
|-----------|---------|-------------|
| [Name] | E-01, E-02 | [what it does] |

---

## Open questions

Questions that must be answered before implementation starts:

| # | Question | Blocks | Owner |
|---|----------|--------|-------|
| Q-01 | [question] | T-03, T-07 | Product |
| Q-02 | [question] | T-04 | Engineering |

---

## Out of scope (this cycle)

Stories and flows explicitly deferred:
- US-XX: [reason]
- UF-XX: [reason]

</output_format>

<quality_checks>
  Before finalizing, check:
  - Every task references a story ID + flow ID (@traceable)
  - No task requires input from another team mid-execution (@atomic_tasks)
  - Every acceptance criterion from user-stories.md maps to a "Test" task
  - The dependency map has no circular dependencies
  - The "Open questions" block captures everything ambiguous
  - The "Out of scope" section lists all Could Have / Won't Have items
</quality_checks>

<completion>
  When implementation-brief.md is ready, say:

  "✅ All 3 files are ready:

  📄 user-stories.md — [N] stories for [N] user types
  📄 user-flows.md — [N] flows covering all stories
  📄 implementation-brief.md — [N] epics, [N] tasks, dependency map

  Your engineers or AI agents can now:
  - Create epics in GitHub/Jira from the E-XX entries
  - Create tasks/tickets from the T-XX entries
  - Use the story IDs as anchors for acceptance tests
  - Use the dependency map to plan sprints

  Recommended next step: discuss the Open questions with the team before sprint planning."
</completion>
```
