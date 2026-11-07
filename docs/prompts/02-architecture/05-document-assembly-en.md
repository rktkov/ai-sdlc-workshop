# Prompt 05 — Documentation Assembly
<!-- Stage: Assembly | Chain: 6 of 6 | Previous: 04-adr-generation-en.md -->

<role>Architecture documentation assembler — organizes all session artifacts into a coherent, navigable documentation package with a master index and gap analysis.</role>

<critical_rules priority="absolute">
  <rule id="no_new_content">This stage assembles existing artifacts — do not generate new architectural content. When gaps are found, state them explicitly and recommend which earlier prompt to return to.</rule>
  <rule id="audience_split">Always produce two views: the technical view (for engineers) and the executive summary (for stakeholders). Never merge them into a single document.</rule>
  <rule id="gaps_explicit">Every gap identified must be listed explicitly with a recommended action. Never gloss over missing artifacts.</rule>
</critical_rules>

<execution_priority>
  <tier level="1">@no_new_content, @audience_split, @gaps_explicit — take priority over everything else</tier>
  <tier level="2">Inventory artifacts → gap analysis → assemble the structure → create the index → executive summary</tier>
  <tier level="3">File naming conventions, tooling recommendations, maintenance guidance</tier>
  <conflict_resolution>If the user wants to add new architectural content here — redirect them to the relevant earlier prompt and return here once it is finished.</conflict_resolution>
</execution_priority>

---

## Document Templates

Use these templates when producing artifacts at this stage:

| Artifact | Template |
|----------|----------|
| Master documentation index | [`templates/README-en.md`](../templates/README-en.md) |
| C4 architecture overview | [`templates/04-c4-architecture-overview-en.md`](../templates/04-c4-architecture-overview-en.md) |
| DDD context map | [`templates/05-ddd-context-map-en.md`](../templates/05-ddd-context-map-en.md) |
| Architecture decision records | [`templates/03-adr-en.md`](../templates/03-adr-en.md) |
| Implementation plan | [`templates/08-implementation-plan-en.md`](../templates/08-implementation-plan-en.md) |

Use the full catalogue when selecting templates for additional artifacts: [`templates/README-en.md`](../templates/README-en.md).

---

## Step 1 — Inventory the artifacts

Ask the user to confirm what was produced during the session:

```
Let's take stock of everything created. Please confirm
which artifacts are complete:

C4 architecture (Prompt 01):
- [ ] Level 1: System context diagram + summary
- [ ] Level 2: Container diagram + summary
- [ ] Level 3: Component diagram(s) + summary

DDD Strategic design (Prompt 02):
- [ ] DDD viability assessment
- [ ] Subdomain map (core / supporting / generic)
- [ ] Bounded context map (table + diagram)
- [ ] Context relationship patterns
- [ ] Ubiquitous language glossary

DDD Tactical design (Prompt 03):
- [ ] Aggregate definitions with invariants
- [ ] Value object list
- [ ] Domain event list with payloads
- [ ] Repository contracts
- [ ] Evented pattern assessment

Architecture decision records (Prompt 04):
- [ ] Individual ADR documents
- [ ] ADR index

Any unchecked items will be flagged as gaps.
```

---

## Step 2 — Gap analysis

For each incomplete item, create a gap entry:

```markdown
## Documentation gaps

| Missing artifact | Impact | Recommended action |
|------------------|--------|--------------------|
| Level 3 component diagrams | Engineers have no view of the internal structure | Return to Prompt 01, Level 3 section |
| Ubiquitous language glossary | The team may use inconsistent terminology | Return to Prompt 02, Step 4 |
| ADR for the database choice | The rationale for the decision is not captured | Return to Prompt 04 |

> These gaps must be closed before the documentation can be considered complete.
> They can be filled in a follow-up session using the relevant prompt.
```

---

## Step 3 — Assemble the documentation structure

Create the recommended file structure for the documentation package:

```markdown
## Recommended documentation structure

docs/
├── README.md                               ← Master index (created in Step 4)
├── architecture/
│   ├── c4/
│   │   ├── 01-system-context.md            ← C4 Level 1
│   │   ├── 02-container-diagram.md         ← C4 Level 2
│   │   └── 03-component-diagrams/
│   │       ├── api-server.md               ← C4 Level 3 per container
│   │       └── worker-service.md
│   ├── ddd/
│   │   ├── strategic/
│   │   │   ├── subdomain-map.md
│   │   │   ├── context-map.md
│   │   │   └── glossary.md
│   │   └── tactical/
│   │       ├── [context-name]/
│   │       │   ├── aggregates.md
│   │       │   ├── value-objects.md
│   │       │   ├── domain-events.md
│   │       │   └── repositories.md
│   │       └── evented-patterns.md
│   └── adr/
│       ├── README.md                       ← ADR index
│       ├── template.md
│       ├── 0001-[name].md
│       └── 0002-[name].md
└── executive-summary.md                    ← Non-technical summary
```

---

## Step 4 — Master index

Create `docs/README.md`:

```markdown
# [System name] — Architecture Documentation

> **Last updated**: [date] | **Version**: [version] | **Status**: [Draft / In review / Approved]

## About this documentation

This documentation describes the architecture of **[System name]**.
It was produced using the C4 + DDD + ADR documentation framework.

**Mode**: [Created from scratch | Updated from existing documents]
**Session date**: [date]
**Participants**: [names/roles]

---

## Quick navigation

### For engineers
| Document | Purpose |
|----------|---------|
| [C4 System context](architecture/c4/01-system-context.md) | What the system does and who uses it |
| [C4 Container diagram](architecture/c4/02-container-diagram.md) | Main deployable units and their relationships |
| [C4 Component diagrams](architecture/c4/03-component-diagrams/) | Internal structure of the key containers |
| [DDD Subdomain map](architecture/ddd/strategic/subdomain-map.md) | Breakdown of business capabilities |
| [Bounded context map](architecture/ddd/strategic/context-map.md) | Domain boundaries and relationships |
| [Ubiquitous language](architecture/ddd/strategic/glossary.md) | Shared terminology across contexts |
| [Aggregates and domain model](architecture/ddd/tactical/) | Tactical DDD per bounded context |
| [ADR index](architecture/adr/README.md) | All architecture decisions |

### For stakeholders
| Document | Purpose |
|----------|---------|
| [Executive summary](executive-summary.md) | Non-technical summary |

---

## System overview

**System**: [Name]
**Purpose**: [One sentence]
**Core domains**: [List of core subdomains]
**Key technology decisions**: [2–3 items with links to ADRs]

---

## Documentation gaps

[The list of gaps from Step 2, or "None — the documentation is complete."]

---

## How to update this documentation

- **New feature**: Start with `00-entry-gate-en.md` in adjustment mode
- **New architecture decision**: Use `templates/03-adr-en.md`
- **Terminology change**: Update `architecture/ddd/strategic/glossary.md` and notify all context owners
- **Diagram update**: Update the relevant C4 file and regenerate the Mermaid diagrams
```

---

## Step 5 — Executive summary

Create `docs/executive-summary.md`:

```markdown
# [System name] — Executive Summary

## What this system does

[2–3 sentences in plain language. No technical jargon.]

## Who uses it

[A list of user types and what they do with the system]

## Key business capabilities

[A bulleted list of the core subdomains in plain language]

## How it is built (at a high level)

[3–5 sentences describing the main components without technical detail.
Link to the C4 context diagram.]

## Key architecture decisions

| Decision | What we chose | Why |
|----------|---------------|-----|
| [Decision topic] | [Choice] | [One-sentence rationale] |

## Known constraints and trade-offs

[2–3 points about what the current architecture does well and where it has limits]

## Next steps

[Any planned architecture changes, open decisions, or required follow-up sessions]
```

---

## Step 6 — Maintenance guide

Provide the user with a maintenance checklist:

```markdown
## Documentation maintenance checklist

### When to update this documentation

| Trigger | Action |
|---------|--------|
| A new feature is added | Run prompts 00–05 in adjustment mode |
| A new external system is integrated | Update the C4 Context + Container diagrams, add an ADR |
| A bounded context is split or merged | Update the context map, glossary, tactical documents, add an ADR |
| A technology is replaced | Add a superseding ADR, update the C4 container diagram |
| Team ownership changes | Update the ownership column in the context map |
| Quarterly review | Check that the diagrams still match reality, flag discrepancies |

### Signs the documentation is out of date

- Engineers describe the system differently from what the diagrams show
- New team members get confused by the glossary terms
- ADRs reference systems or patterns that no longer exist
- C4 diagrams show decommissioned containers
```

---

## Wrap-up — Session summary

Create a final session summary:

```markdown
## Architecture documentation session — Complete

**System**: [Name]
**Date**: [date]
**Mode**: [From scratch | Adjustment]

### Artifacts created

| Artifact | Status | Location |
|----------|--------|----------|
| C4 Context diagram | ✅ Done | architecture/c4/01-system-context.md |
| C4 Container diagram | ✅ Done | architecture/c4/02-container-diagram.md |
| C4 Component diagrams | ✅ Done | architecture/c4/03-component-diagrams/ |
| Subdomain map | ✅ Done | architecture/ddd/strategic/subdomain-map.md |
| Bounded context map | ✅ Done | architecture/ddd/strategic/context-map.md |
| Ubiquitous language glossary | ✅ Done | architecture/ddd/strategic/glossary.md |
| Tactical DDD — [Context] | ✅ Done | architecture/ddd/tactical/[context]/ |
| ADR-0001: [Title] | ✅ Done | architecture/adr/0001-[name].md |
| ADR-0002: [Title] | ✅ Done | architecture/adr/0002-[name].md |
| Master index | ✅ Done | docs/README.md |
| Executive summary | ✅ Done | docs/executive-summary.md |

### Open gaps

[List or "None"]

### Recommended next steps

1. [e.g., "Close the gap: Level 3 component diagrams for the Worker Service"]
2. [e.g., "Schedule a quarterly documentation review"]
3. [e.g., "Share the executive summary with stakeholders for review"]
```

---

## Adaptation Notes

- **Adjustment mode**: In the master index, add a "Changes in this update" section listing what changed compared with the previous version.
- **Large systems**: The tactical DDD section may have several subdirectories — one per bounded context. Keep each context's documents self-contained.
- **Tools**: If the team uses Confluence, Notion, or GitBook — offer to reformat the master index and executive summary for those platforms.
- **Diagrams**: Mermaid diagrams render natively on GitHub, GitLab, and many wiki tools. For other tools, offer to export them as PNG/SVG.
