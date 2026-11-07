# I Am Lead — Document Templates (English)

This folder contains the English-only versions of the document templates. Russian originals are
kept alongside with the same base name; English versions use the `-en` suffix.

## Usage rules

- Fill sections with concrete facts, numbers, and links.
- Do not fill sections formally: if a section is not applicable, explain why.
- Always record the document status and owner.
- Keep the terminology consistent with your ubiquitous language glossary.

---

## Template catalogue

Use this table to pick the right template for the artifact you are generating.

### Discovery and product definition

| Template | Artifact | Used in |
|----------|----------|---------|
| [`01-prd-en.md`](01-prd-en.md) | Product Requirements Document | `01-feature` (feature spec), `03-user-stories` (story scope) |
| [`02-rfc-en.md`](02-rfc-en.md) | Technical proposal / design doc | `01-feature`, `05-init-project` (input analysis) |
| [`03-adr-en.md`](03-adr-en.md) | Architecture Decision Record | `02-architecture` (prompt 04), `05-init-project` |

### Architecture

| Template | Artifact | Used in |
|----------|----------|---------|
| [`04-c4-architecture-overview-en.md`](04-c4-architecture-overview-en.md) | C4 architecture overview (Levels 1–3) | `02-architecture` (prompt 01, prompt 05) |
| [`05-ddd-context-map-en.md`](05-ddd-context-map-en.md) | DDD domain context map (contexts, aggregates, events) | `02-architecture` (prompts 02–03) |
| [`06-data-flow-en.md`](06-data-flow-en.md) | Data flow description | `02-architecture` (prompt 03), `03-user-stories` (prompt 2) |
| [`07-control-flow-en.md`](07-control-flow-en.md) | Control flow / sequence description | `02-architecture` (prompt 03), `03-user-stories` (prompt 2) |

### Planning and delivery

| Template | Artifact | Used in |
|----------|----------|---------|
| [`08-implementation-plan-en.md`](08-implementation-plan-en.md) | Implementation plan (epics, tasks, dependencies) | `03-user-stories` (prompt 3), `05-planning` |
| [`09-raci-en.md`](09-raci-en.md) | RACI responsibility matrix | `05-planning`, project setup |
| [`10-raid-log-en.md`](10-raid-log-en.md) | RAID log (risks, assumptions, issues, dependencies) | `05-planning`, delivery tracking |
| [`13-release-plan-en.md`](13-release-plan-en.md) | Release plan | Delivery, `04-testing` |

### Quality and testing

| Template | Artifact | Used in |
|----------|----------|---------|
| [`11-test-strategy-en.md`](11-test-strategy-en.md) | Test strategy | `04-testing`, `03-user-stories` (prompt 3) |
| [`12-test-matrix-en.md`](12-test-matrix-en.md) | Test matrix (requirement → test) | `04-testing`, `03-user-stories` (prompt 3) |
| [`14-launch-checklist-en.md`](14-launch-checklist-en.md) | Launch readiness checklist | Delivery, release |

### Operations and incidents

| Template | Artifact | Used in |
|----------|----------|---------|
| [`15-rollback-plan-en.md`](15-rollback-plan-en.md) | Rollback plan | Release, operations |
| [`16-playbook-en.md`](16-playbook-en.md) | Operational playbook | Operations |
| [`17-runbook-en.md`](17-runbook-en.md) | Technical runbook | `05-init-project` (`docs/runbooks/`), operations |
| [`18-incident-protocol-en.md`](18-incident-protocol-en.md) | Incident protocol | Operations |
| [`19-postmortem-en.md`](19-postmortem-en.md) | Post-incident review | Operations |

### Technical debt

| Template | Artifact | Used in |
|----------|----------|---------|
| [`23-technical-debt-register-en.md`](23-technical-debt-register-en.md) | Technical debt register | `05-init-project`, ongoing delivery |
| [`24-technical-debt-item-en.md`](24-technical-debt-item-en.md) | Technical debt item | Ongoing delivery |
| [`25-debt-repayment-plan-en.md`](25-debt-repayment-plan-en.md) | Debt repayment plan | Planning cycles |
| [`26-debt-decision-record-en.md`](26-debt-decision-record-en.md) | Debt decision record | Architecture / planning reviews |

### Workshop facilitation and coaching

| Template | Artifact | Used in |
|----------|----------|---------|
| [`20-session-analysis-en.md`](20-session-analysis-en.md) | Internal session analysis | Workshop facilitation |
| [`21-session-result-en.md`](21-session-result-en.md) | Session result for the participant | Workshop facilitation |
| [`22-grade-growth-rubric-en.md`](22-grade-growth-rubric-en.md) | Grade growth rubric | Coaching and review |

---

## Phase quick reference

| Phase folder | Primary templates |
|--------------|-------------------|
| `01-feature/` | `01-prd-en.md`, `02-rfc-en.md` |
| `02-architecture/` | `03-adr-en.md`, `04-c4-architecture-overview-en.md`, `05-ddd-context-map-en.md`, `06-data-flow-en.md`, `07-control-flow-en.md` |
| `03-user-stories/` | `01-prd-en.md`, `06-data-flow-en.md`, `07-control-flow-en.md`, `08-implementation-plan-en.md`, `11-test-strategy-en.md`, `12-test-matrix-en.md` |
| `04-testing/` | `11-test-strategy-en.md`, `12-test-matrix-en.md`, `13-release-plan-en.md`, `14-launch-checklist-en.md` |
| `05-planning/` | `08-implementation-plan-en.md`, `09-raci-en.md`, `10-raid-log-en.md`, `13-release-plan-en.md` |
| `05-init-project-en.md` | `02-rfc-en.md`, `03-adr-en.md`, `08-implementation-plan-en.md`, `09-raci-en.md`, `17-runbook-en.md`, `23-technical-debt-register-en.md` |
