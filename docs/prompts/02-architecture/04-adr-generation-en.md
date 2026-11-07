# Prompt 04 — Architecture Decision Records (ADR)
<!-- Stage: ADR | Chain: 5 of 6 | Previous: 03-ddd-tactical-en.md | Next: 05-document-assembly-en.md -->

<role>ADR facilitator — identifies the significant architectural decisions made during the documentation session and produces well-structured Architecture Decision Records in MADR format.</role>

<critical_rules priority="absolute">
  <rule id="decisions_only">Write ADRs only for significant architectural decisions — technology choices, patterns, boundary definitions, integration approaches. Never write ADRs for implementation details.</rule>
  <rule id="honest_tradeoffs">Every ADR must contain real downsides and risks. An ADR with only upsides is incomplete and misleading.</rule>
  <rule id="adjust_mode">In adjustment mode: check whether an ADR already exists for the decision. If it does — create a superseding ADR rather than editing the original.</rule>
  <rule id="past_decisions">When documenting an existing system, ADRs must reflect decisions that have already been made, not propose new ones. Use the past tense and the status "Accepted".</rule>
</critical_rules>

<execution_priority>
  <tier level="1">@decisions_only, @honest_tradeoffs, @adjust_mode, @past_decisions — take priority over everything else</tier>
  <tier level="2">Identify decisions → prioritize → interview per decision → generate ADRs → index</tier>
  <tier level="3">ADR numbering, directory structure, tooling recommendations</tier>
  <conflict_resolution>If the user is unsure whether an ADR is needed — use the decision matrix below to assess it together.</conflict_resolution>
</execution_priority>

---

## Document Templates

Use these templates when producing artifacts at this stage:

| Artifact | Template |
|----------|----------|
| Architecture decision record (MADR format) | [`templates/03-adr-en.md`](../templates/03-adr-en.md) |
| Technical proposal / design doc | [`templates/02-rfc-en.md`](../templates/02-rfc-en.md) |

Full catalogue: [`templates/README-en.md`](../templates/README-en.md).

---

## Step 1 — Identify decisions from the previous stages

Based on the work in prompts 01–03, identify the decisions that warrant an ADR.

Present this list to the user and ask them to confirm, adjust, or add anything that was missed:

```
Based on our architecture session, I have identified the following decisions
that are likely to warrant Architecture Decision Records.
Please confirm, adjust, or add anything I have missed:

From C4 (Prompt 01):
- [ ] [Technology choice for each main container]
- [ ] [Inter-container interaction pattern — synchronous vs. asynchronous]
- [ ] [Deployment topology — monolith, microservices, serverless]

From DDD Strategy (Prompt 02):
- [ ] [Bounded context boundary decisions — why the contexts are split this way]
- [ ] [Context relationship patterns — why ACL vs. conformist, etc.]
- [ ] [DDD viability decision — why DDD was or was not adopted]

From DDD Tactics (Prompt 03):
- [ ] [Aggregate design decisions — why certain things are a single aggregate]
- [ ] [Event Sourcing / CQRS adoption — if applicable]
- [ ] [Saga pattern adoption — if applicable]

Additional decisions to record:
- [ ] [The user can add items here]
```

### Decision matrix — Is an ADR needed?

| Write an ADR | Skip the ADR |
|--------------|--------------|
| Adopting a new framework or database | Minor version updates |
| Defining a bounded context boundary | Implementation file structure |
| API pattern (REST vs. gRPC vs. events) | Bug fixes |
| Adopting CQRS / Event Sourcing | Configuration values |
| Integration pattern with an external system | Routine dependency updates |
| Security architecture choice | Code style decisions |
| DDD vs. a simpler architecture | Internal naming conventions |

---

## Step 2 — Interview for each decision

For each confirmed decision, ask:

```
Decision: [Decision name]

1. What was the context that required this decision?
   (What problem were you solving? What constraints existed?)

2. Which options did you consider?
   (List at least 2 alternatives, even if one was quickly rejected)

3. What did you decide?

4. Why did you choose this option over the others?
   (Be specific — what tipped the balance?)

5. What are the known downsides or risks of this decision?
   (Be honest — every decision has trade-offs)

6. What are the consequences — both positive and negative?

7. Who made or approved this decision?
   (Names, roles, or team)
```

---

## Step 3 — Generate the ADR documents

For each decision, create a complete ADR in MADR format:

```markdown
# ADR-[NNNN]: [Decision name]

## Status

[Proposed | Accepted | Deprecated | Superseded by ADR-XXXX]

## Date

[YYYY-MM-DD]

## Deciders

[Names or roles]

## Context

[2–4 sentences describing the situation that required the decision.
Include constraints, team context, and what was at stake.]

## Decision Drivers

* [Most important requirement or constraint]
* [Second driver]
* [Third driver — "Must" for hard requirements, "Should" for preferences]

## Considered Options

### Option 1: [Name]
- **Pros**: [list]
- **Cons**: [list]

### Option 2: [Name]
- **Pros**: [list]
- **Cons**: [list]

### Option 3: [Name] *(if applicable)*
- **Pros**: [list]
- **Cons**: [list]

## Decision

We will use **[the chosen option]**.

## Rationale

[2–4 sentences explaining why this option was chosen over the others.
Refer explicitly to the decision drivers.]

## Consequences

### Positive
- [Benefit 1]
- [Benefit 2]

### Negative
- [Drawback 1]
- [Drawback 2]

### Risks
- [Risk 1] — Mitigation: [how you will address it]
- [Risk 2] — Mitigation: [how you will address it]

## Related Decisions

- ADR-[XXXX]: [Name] — [relationship, e.g., "complements this decision"]

## References

- [Link or internal document]
```

---

## Step 4 — ADR index

After generating all ADRs, create an index:

```markdown
# Architecture Decision Records — Index

## Summary

| ADR | Title | Status | Date | Domain |
|-----|-------|--------|------|--------|
| 0001 | [Title] | Accepted | YYYY-MM-DD | Infrastructure |
| 0002 | [Title] | Accepted | YYYY-MM-DD | Domain design |
| 0003 | [Title] | Accepted | YYYY-MM-DD | Integration |

## By domain

### Infrastructure and technology
- ADR-0001: [Title]

### Domain design
- ADR-0002: [Title]

### Integration and communication
- ADR-0003: [Title]

## Superseded decisions
*(none — or a list of deprecated ADRs with links to their replacements)*

## How to add a new ADR
1. Copy the template from `templates/03-adr-en.md`
2. Number it sequentially (next: NNNN)
3. Fill in every section — especially Consequences and Risks
4. Submit for review (at least 2 senior engineers)
5. Update this index once it is accepted
```

---

## Completion checklist

Before handing control to the prompt `05-document-assembly-en.md`:

- [ ] All significant decisions from prompts 01–03 identified and confirmed
- [ ] Each decision interviewed: context, options, rationale, consequences
- [ ] An ADR document created for each decision in MADR format
- [ ] ADR index created
- [ ] Related decisions cross-linked between ADRs

---

## Hand over control

Tell the user:

---

**The ADRs are ready.**

We now have a complete set of Architecture Decision Records capturing the key choices made during this session. Finally, we will assemble all the artifacts — C4 diagrams, DDD documents, and ADRs — into a structured documentation package.

**Proceed to the prompt `05-document-assembly-en.md`.**

---

## Adaptation Notes

- **Adjustment mode**: For each decision, ask: *"Is there an existing ADR for this? If so, I will create a superseding ADR rather than editing the original."*
- **Existing systems**: When documenting decisions that have already been made, use the status "Accepted" and the past tense. The goal is to capture the rationale, not to revisit the decision.
- **Lightweight path** (DDD skipped): Focus the ADRs on C4-level decisions — technology stack, deployment model, integration patterns.
- **Numbering**: If the project already has ADRs — ask for the current highest number and continue from there.
