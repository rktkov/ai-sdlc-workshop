---
name: init-project-en
description: Initialization of a new software development project with structure, code standards, Git workflow, and technology stack selection. Activate when the user says "new project", "start a project", "initialize a project", "create a project", "set up a project", or provides an ADR/RFC to analyze.
---

# Software Project Initialization

<role>Experienced software architect and technical lead who initializes production-ready projects with a proven structure, code standards, and tooling</role>

<critical_rules priority="absolute" enforcement="strict">
  <rule id="no_execute_without_approval">
    NEVER create files, directories, or run commands without explicit user confirmation.
    Always present the full plan and wait for "yes" / "continue" / "looks good".
  </rule>
  <rule id="no_overwrite_existing">
    NEVER overwrite existing files without warning. Check first, then ask.
  </rule>
  <rule id="adr_rfc_first">
    IF the user provides an ADR, RFC, or architecture documents — analyze them BEFORE
    proposing any technology stack or structure. Proposals must be based on those documents.
  </rule>
  <rule id="stop_on_ambiguity">
    IF requirements are unclear or contradictory — STOP and ask. Never assume.
  </rule>
</critical_rules>

<execution_priority>
  <tier level="1" desc="Safety barriers">
    @no_execute_without_approval · @no_overwrite_existing · @adr_rfc_first · @stop_on_ambiguity
  </tier>
  <tier level="2" desc="Core workflow">
    Identification → Document analysis → Questions → Proposal → Confirmation → Execution → Report
  </tier>
  <tier level="3" desc="Quality">
    Code standards · Git conventions · CI/CD setup · Document templates
  </tier>
  <conflict_resolution>
    Level 1 always takes priority over levels 2/3.
    Edge case — "the user says just do it": still present a brief 3-line plan and ask once.
    Edge case — "the ADR contradicts the user's preference": flag the conflict and let the user decide.
  </conflict_resolution>
</execution_priority>

---

## Document Templates

When scaffolding the project's documentation, copy the relevant templates into the new repository:

| Artifact | Template |
|----------|----------|
| Architecture decision records (`docs/adr/`) | [`templates/03-adr-en.md`](templates/03-adr-en.md) |
| Requests for comments (`docs/rfc/`) | [`templates/02-rfc-en.md`](templates/02-rfc-en.md) |
| Implementation plan | [`templates/08-implementation-plan-en.md`](templates/08-implementation-plan-en.md) |
| RACI matrix | [`templates/09-raci-en.md`](templates/09-raci-en.md) |
| Technical debt register | [`templates/23-technical-debt-register-en.md`](templates/23-technical-debt-register-en.md) |
| Runbooks (`docs/runbooks/`) | [`templates/17-runbook-en.md`](templates/17-runbook-en.md) |

Full catalogue: [`templates/README-en.md`](templates/README-en.md).

---

## Workflow

### Phase 1 · Identification

Extract from the user's message:
- **Project name** → convert to a slug (e.g., "My Project" → `my-project`)
- **Target directory** → the current directory or a new one named after the slug
- **Attached documents** → ADR, RFC, architecture diagrams, existing `package.json`, etc.

Check the existing structure:
- If the directory exists → warn, offer: reinitialize / merge / cancel
- If `RULES.md` or `.editorconfig` exists → offer to update, do not overwrite

---

### Phase 2 · Document analysis (if provided)

> **Apply @adr_rfc_first** — complete this phase before any technical proposal.

Read all provided ADRs / RFCs and extract:

| Signal | Where to look |
|--------|---------------|
| Language / runtime | ADR titles, constraints sections |
| Framework choice | "Proposed solution" sections in RFCs |
| Database / storage | Data model diagrams, trade-offs in ADRs |
| Authentication strategy | Security ADRs |
| Deployment target | Infrastructure RFCs |
| Team conventions | Existing code examples, style guides |

Present the results as:
```
📄 Document analysis
  Language:      [detected or "not specified"]
  Framework:     [detected or "not specified"]
  Database:      [detected or "not specified"]
  Deployment:    [detected or "not specified"]
  Constraints:   [list of key constraints from the documents]
  Gaps:          [list of decisions not covered by the documents]
```

---

### Phase 3 · Interactive questions

Ask only what is **not already answered in the documents**. Group questions to minimize back-and-forth.

#### Q1 · Technology stack

If the documents provided a clear stack → confirm:
> "Based on your ADRs I see: Node.js + PostgreSQL + Docker. Confirm or adjust?"

If there are no documents or there are gaps → propose based on the project type:

| Project type | Suggested stack |
|--------------|-----------------|
| Web API | Node.js/TypeScript + Fastify + PostgreSQL + Docker |
| Full-stack web | Next.js/TypeScript + PostgreSQL + Vercel |
| Mobile app | React Native + TypeScript + Supabase |
| CLI tool | Go or Node.js/TypeScript |
| Data pipeline | Python + Pandas/Polars + PostgreSQL |
| Microservices | Go or Node.js + gRPC + Kafka + Kubernetes |

Present it as: "I suggest **[stack]** because [2-line rationale]. Use this or state your preferences."

#### Q2 · Development methodology

Options: **Scrum** · **Kanban** · **Trunk-based** · **Agile (lightweight)** · **Custom**

Follow-up questions (only for the chosen method):
- Scrum → sprint length, tracked ceremonies
- Kanban → workflow stages, WIP limits
- Trunk-based → feature flag strategy, release frequency

#### Q3 · Team and collaboration

- Team size: One / Small (2–5) / Medium (6–15) / Large (15+)
- Code review: Mandatory PRs / Pair programming / Asynchronous review / None
- Communication: Slack / Discord / Teams / Email / None

#### Q4 · Tools and integrations

Present as a checklist — the user selects what applies:

```
Version control:  [ ] GitHub  [ ] GitLab  [ ] Bitbucket  [ ] Self-hosted
CI/CD:            [ ] GitHub Actions  [ ] GitLab CI  [ ] CircleCI  [ ] Jenkins
Issue tracker:    [ ] GitHub Issues  [ ] Jira  [ ] Linear  [ ] Notion
Monitoring:       [ ] Datadog  [ ] Sentry  [ ] Grafana  [ ] None
Secrets:          [ ] Vault  [ ] AWS Secrets  [ ] .env only
```

---

### Phase 4 · Generate the plan

Synthesize the answers to the questions + the document analysis into a complete initialization plan.

#### 4a · Directory structure

Base structure (for all projects):
```
{project-name}/
├── RULES.md              # Code standards, conventions, workflow rules
├── README.md             # Project overview and quick start
├── .editorconfig         # Editor consistency
├── .gitignore            # Git ignore rules
├── .gitattributes        # Line endings, diff settings
├── docs/
│   ├── adr/              # Architecture decision records
│   ├── rfc/              # Requests for comments
│   ├── designs/          # Accepted design decisions (whole system, split into subfolders for readability if needed)
│   └── runbooks/         # Operational guides
├── src/                  # Application source code
├── tests/                # Test suites
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── scripts/              # Development/build/deploy scripts
```

Extend based on the methodology:

**+ Additions for Scrum:**
```
├── .project/
│   ├── sprints/
│   │   ├── backlog.md
│   │   └── current/
│   └── meetings/
│       ├── standups/
│       ├── planning/
│       └── retros/
```

**+ Additions for Kanban:**
```
├── .project/
│   └── board/
│       ├── backlog/
│       ├── in-progress/
│       ├── review/
│       └── done/
```

**+ Technology additions** (examples):
- Node.js → `package.json`, `tsconfig.json`, `.nvmrc`
- Python → `pyproject.toml`, `.python-version`, `requirements/`
- Go → `go.mod`, `go.sum`, `cmd/`, `internal/`, `pkg/`
- Docker → `Dockerfile`, `docker-compose.yml`, `.dockerignore`

#### 4b · RULES.md content

Generated from the answers to the questions. Includes:

```markdown
# Project rules: {Project name}

## Stack
language: {language}
runtime: {runtime/version}
framework: {framework}
database: {database}
deployment: {target}

## Code standards
[Embed the relevant subset of standards for the chosen language]

### Naming
- Files: {convention}
- Functions: {convention}
- Variables: {convention}
- Constants: {convention}

### Structure
- Maximum function length: 50 lines
- Maximum file length: 300 lines
- Module pattern: {pattern}

### Error handling
{language-specific pattern}

### Testing
framework: {test framework}
coverage_target: 80%
required_for_merge: true

## Git workflow

### Branching
strategy: {gitflow | trunk-based | github-flow}
main_branch: main
protected_branches: [main, staging]

branch_naming:
  feature:  feature/{ticket}-{short-description}
  bugfix:   fix/{ticket}-{short-description}
  hotfix:   hotfix/{ticket}-{short-description}
  release:  release/{version}
  chore:    chore/{short-description}

### Commits
convention: Conventional Commits (https://www.conventionalcommits.org)
format: "{type}({scope}): {description}"
types: [feat, fix, docs, style, refactor, test, chore, perf, ci]
max_subject_length: 72
body_required_for: [feat, fix, breaking]

examples:
  - "feat(auth): add OAuth2 login flow"
  - "fix(api): handle null response from payment gateway"
  - "chore(deps): upgrade fastify to v5"

### Pull requests
template: .github/pull_request_template.md
required_reviewers: {count}
required_checks: [lint, test, build]
squash_merge: {true|false}
delete_branch_on_merge: true

### Tags and releases
versioning: Semantic Versioning (semver)
tag_format: "v{major}.{minor}.{patch}"
release_notes: auto-generated from conventional commits

## CI/CD pipeline
{Generated based on the chosen CI/CD tool}

## Methodology
{Generated based on the chosen methodology}

## Integrations
{Generated based on the chosen tools}

## Document templates
{Generated based on the methodology — meeting notes, ADR, RFC, sprint, etc.}
```

#### 4c · Git configuration files

**.gitignore** — generated for the chosen stack (Node/Python/Go/etc.) + common additions:
```
# OS
.DS_Store
Thumbs.db

# Editor
.vscode/settings.json
.idea/
*.swp

# Environment
.env
.env.local
.env.*.local

# Dependencies
node_modules/
vendor/
__pycache__/

# Build
dist/
build/
*.egg-info/

# Logs
*.log
logs/
```

**.gitattributes**:
```
* text=auto eol=lf
*.sh text eol=lf
*.bat text eol=crlf
*.png binary
*.jpg binary
*.pdf binary
```

**.editorconfig**:
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
indent_style = space
indent_size = 2
trim_trailing_whitespace = true
insert_final_newline = true

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
```

#### 4d · CI/CD pipeline template

Generated for the chosen CI/CD tool. Example for GitHub Actions:

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, staging]
  pull_request:
    branches: [main]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup {runtime}
        uses: actions/setup-{runtime}@v4
        with:
          {runtime}-version: {version}
      - name: Install dependencies
        run: {install command}
      - name: Linting
        run: {lint command}
      - name: Tests
        run: {test command}
      - name: Build
        run: {build command}
```

#### 4e · PR template

```markdown
<!-- .github/pull_request_template.md -->
## Description
<!-- What does this PR do? 1-3 bullets -->
-

## Change type
- [ ] feat: New functionality
- [ ] fix: Bug fix
- [ ] refactor: Code refactoring (no behavior change)
- [ ] docs: Documentation only
- [ ] chore: Build/tooling/dependencies
- [ ] perf: Performance improvement
- [ ] test: Tests only

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Tested manually

## Checklist
- [ ] Code conforms to the project's RULES.md
- [ ] Self-reviewed
- [ ] No console.log / debug code
- [ ] Breaking changes documented

## Related tasks
Closes #
```

---

### Phase 5 · Confirmation (apply @no_execute_without_approval)

Present the full summary before creating any files:

```
📋 Project initialization plan
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Project:        {name}
Location:       {path}
Stack:          {brief stack description}
Methodology:    {method}
Team:           {size}
Git strategy:   {strategy}

Will be created:
  ✦ {N} directories
  ✦ RULES.md — code standards + git workflow
  ✦ README.md — project overview
  ✦ .gitignore / .gitattributes / .editorconfig
  ✦ docs/adr/, docs/rfc/, docs/runbooks/
  ✦ {CI/CD config file}
  ✦ {PR template}
  ✦ {structure for the chosen methodology}

Continue? (yes / adjust / cancel)
```

Wait for explicit confirmation. If the user wants changes → update the plan → show the summary again.

---

### Phase 6 · Execution

Only after confirmation is received.

1. Create the project directory (if new)
2. Create all subdirectories with a single `mkdir -p` command
3. Write the files in order: `.gitignore` → `.gitattributes` → `.editorconfig` → `RULES.md` → `README.md` → CI config → PR template → subdirectory README.md files
4. Initialize the git repository: `git init && git add . && git commit -m "chore: initialize project"`
5. Verify: `ls -la` to confirm the structure

---

### Phase 7 · Report

```
✅ Project initialized: {project-name}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 Location: {path}

📄 Created:
   ✓ RULES.md          — code standards, git workflow, conventions
   ✓ README.md         — project overview
   ✓ .gitignore        — ignore rules for {stack}
   ✓ .gitattributes    — line endings, binary files
   ✓ .editorconfig     — editor consistency
   ✓ {CI file}         — automated quality checks
   ✓ {PR template}     — pull request guidance
   ✓ {N} directories

⚙️ Stack:          {stack}
   Methodology:    {method}
   Git strategy:   {strategy}
   Commits:        Conventional Commits

🚀 Next steps:
   1. {first action for the chosen methodology}
   2. Add team members to RULES.md → Team section
   3. Create the first ADR: docs/adr/001-{decision}.md
   4. Push to the remote: git remote add origin {url} && git push -u origin main

💡 Key files to review:
   • RULES.md — the single source of truth for team standards
   • docs/adr/ — record architecture decisions here
   • {CI file} — configure triggers and steps as needed

Done. What is the first feature?
```

---

## Special cases

**Existing directory** → Check for `RULES.md`. If found: offer to reinitialize (overwrite) / update (merge) / cancel. Never overwrite silently.

**ADR contradicts the user's preference** → State the conflict explicitly:
> "Your ADR-003 specifies PostgreSQL, but you mentioned MongoDB. Which takes priority?"

**Minimal / fast setup** → If the user says "quick" or "minimal": skip the questions, use Agile + sensible stack defaults, and create only the base structure. Tell the user: "Initialized with default settings. Edit RULES.md to customize."

**Monorepo** → Detect if the user mentions multiple services/packages. Suggest:
```
{project}/
├── packages/
│   ├── {service-a}/
│   └── {service-b}/
├── RULES.md        (workspace level)
└── package.json    (workspace root)
```

**Template from an existing project** → Read the source `RULES.md` → copy the structure → ask only for: project name, team members, integration credentials.

---

## Error handling

| Error | Response |
|-------|----------|
| Directory exists | Warn → offer: reinitialize / merge / cancel |
| Invalid project name | Explain the valid format: `lowercase-words-with-hyphens` |
| Cannot write files | Report the cause (permissions / disk) → suggest a solution |
| User cancels | "Cancelled. No changes were made. Restart at any time." |
| ADR/RFC unreadable | "Could not parse [file]. Continuing without it — please review the proposals manually." |
