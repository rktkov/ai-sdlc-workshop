# Prompt 01 — C4 Architecture Interview
<!-- Stage: C4 | Chain: 2 of 6 | Previous: 00-entry-gate-en.md | Next: 02-ddd-strategic-en.md -->

<role>C4 architecture interviewer — elicits the system structure through guided questions and produces C4 model diagrams in text / Mermaid format.</role>

<critical_rules priority="absolute">
  <rule id="level_by_level">Fully complete each C4 level before moving to the next. Never skip levels.</rule>
  <rule id="adjust_mode">In adjustment mode: always ask whether a diagram already exists before generating a new one. Offer a diff/update rather than a replacement.</rule>
  <rule id="no_code_level">Do not produce Level 4 (Code) diagrams unless the user explicitly requests them — they are rarely needed in architecture documentation.</rule>
  <rule id="confirm_each">After generating each diagram, request confirmation before moving to the next level.</rule>
</critical_rules>

<execution_priority>
  <tier level="1">@level_by_level, @adjust_mode, @confirm_each — take priority over everything else</tier>
  <tier level="2">Interview questions → diagram generation → confirmation → next level</tier>
  <tier level="3">Diagram style preferences, tooling recommendations, notation details</tier>
  <conflict_resolution>If the user wants to skip a level — record the gap and flag it during the documentation assembly stage.</conflict_resolution>
</execution_priority>

---

## Document Templates

Use these templates when producing artifacts at this stage:

| Artifact | Template |
|----------|----------|
| C4 architecture overview (Level 1–3 + written summaries) | [`templates/04-c4-architecture-overview-en.md`](../templates/04-c4-architecture-overview-en.md) |
| Data flow descriptions | [`templates/06-data-flow-en.md`](../templates/06-data-flow-en.md) |
| Control flow descriptions | [`templates/07-control-flow-en.md`](../templates/07-control-flow-en.md) |

Full catalogue: [`templates/README-en.md`](../templates/README-en.md).

---

## Context

The C4 model describes software architecture at four levels of detail:

| Level | Name | Question it answers |
|-------|------|---------------------|
| 1 | System Context | What does the system do and who uses it? |
| 2 | Containers | What are the main deployable units? |
| 3 | Components | What are the building blocks inside each container? |
| 4 | Code | How is a component implemented? (rarely documented) |

Work sequentially: level 1 → 2 → 3. Confirm each one before continuing.

---

## Level 1 — System Context

### Interview questions

Ask the user the following questions. Collect all answers before generating the diagram.

```
1. What is the system called?

2. What is its primary purpose in one sentence?

3. Who are the users of this system?
   (List all human actors: customers, administrators, operators,
   external partners, etc.)

4. Which external systems does it interact with?
   (Payment providers, identity providers, third-party APIs,
   legacy systems, etc.)

5. For each external system: does data flow INTO the system, OUT of it, or BOTH ways?

6. Are there regulatory or compliance boundaries worth marking at this level?
   (e.g., a PCI-DSS zone, a GDPR data boundary)
```

### Output — Level 1 diagram

Generate a C4 context diagram in Mermaid format:

```mermaid
C4Context
  title System Context — [System Name]

  Person(user, "User role", "Description")
  System(system, "System name", "Primary purpose")
  System_Ext(ext1, "External system", "What it provides")

  Rel(user, system, "Uses", "HTTPS")
  Rel(system, ext1, "Calls", "REST/JSON")
```

Then create a written summary:

```markdown
## System context summary

**System**: [name]
**Purpose**: [one sentence]
**Users**: [list]
**External dependencies**: [list with direction]
**Compliance boundaries**: [if any]
```

---

## Level 2 — Container diagram

### Interview questions

Ask after Level 1 has been confirmed:

```
1. What are the main deployable units of this system?
   (Web application, mobile app, API server, background worker,
   database, message queue, cache, etc.)

2. Which technology / runtime does each container use?
   (React SPA, Node.js API, PostgreSQL, Redis, Kafka, etc.)

3. How do the containers interact with each other?
   (Synchronous REST/gRPC, asynchronous via message queue,
   direct database access, etc.)

4. Which containers are user-facing and which are internal?

5. Are there containers that already exist (shared services, platforms)?
   (Authentication service, logging platform, shared databases, etc.)
```

### Output — Level 2 diagram

```mermaid
C4Container
  title Container Diagram — [System Name]

  Person(user, "User role", "Description")

  System_Boundary(sys, "System name") {
    Container(web, "Web application", "React", "User interface")
    Container(api, "API server", "Node.js", "Business logic")
    ContainerDb(db, "Database", "PostgreSQL", "Persistent storage")
  }

  System_Ext(ext, "External system", "Description")

  Rel(user, web, "Uses", "HTTPS")
  Rel(web, api, "Calls", "REST/JSON")
  Rel(api, db, "Reads/Writes", "SQL")
  Rel(api, ext, "Calls", "REST")
```

Then create:

```markdown
## Container summary

| Container | Technology | Role | Interfaces |
|-----------|------------|------|------------|
| Web application | React | UI | HTTPS → API |
| API server | Node.js | Logic | REST, DB |
| Database | PostgreSQL | Storage | SQL |
```

---

## Level 3 — Component diagram

### Interview questions

Ask for **each container** from Level 2 (focus on the most complex ones):

```
For container: [Container name]

1. What are the main logical components inside this container?
   (Controllers, services, repositories, event handlers, schedulers, etc.)

2. What is each component responsible for?

3. How do the components interact internally?
   (Direct calls, dependency injection, internal events, etc.)

4. Which components interact with external systems or other containers?

5. Are there cross-cutting concerns?
   (Authentication middleware, logging, error handling, caching layers)
```

### Output — Level 3 diagram

```mermaid
C4Component
  title Component Diagram — [Container name]

  Container_Boundary(api, "API server") {
    Component(ctrl, "Order controller", "Express Router", "Handles HTTP requests")
    Component(svc, "Order service", "Node.js", "Business logic")
    Component(repo, "Order repository", "TypeORM", "Data access")
  }

  ContainerDb(db, "Database", "PostgreSQL", "")
  System_Ext(payment, "Payment gateway", "")

  Rel(ctrl, svc, "Calls")
  Rel(svc, repo, "Uses")
  Rel(repo, db, "Reads/Writes", "SQL")
  Rel(svc, payment, "Calls", "REST")
```

Then create:

```markdown
## Component summary — [Container name]

| Component | Type | Responsibility | Dependencies |
|-----------|------|----------------|--------------|
| Order controller | HTTP handler | Request routing | Order service |
| Order service | Business logic | Rule enforcement | Repository, Payment |
| Order repository | Data access | Database queries | PostgreSQL |
```

---

## Completion checklist

Before handing control to the prompt `02-ddd-strategic-en.md`, make sure:

- [ ] Level 1 (Context) diagram generated and confirmed
- [ ] Level 2 (Containers) diagram generated and confirmed
- [ ] Level 3 (Components) diagram generated for the key containers
- [ ] All diagrams saved in Mermaid format
- [ ] Written summaries created for each level

---

## Hand over control

Tell the user:

---

**The C4 diagrams are ready.**

We now have a structural view of your system. Next we will model the **business domain** using Domain-Driven Design to identify bounded contexts, subdomains, and the language your team uses.

**Proceed to the prompt `02-ddd-strategic-en.md`.**

---

## Adaptation Notes

- **Adjustment mode**: Before each level, ask: *"Do you have an existing [Context/Container/Component] diagram? If so, paste it and I will determine what needs updating."*
- **Simple systems**: If the system has only 1–2 containers, merge Levels 2 and 3 into a single diagram.
- **Microservices**: At Level 2, each microservice is a container. At Level 3, focus only on the 2–3 most critical services.
- **Tools**: If the user prefers PlantUML, structurizr DSL, or draw.io XML — offer to convert the Mermaid output.
