# Test-first plan

## Status

Active from 2026-11-04. This plan applies to every implementation task.

## 1. Rule

No task writes production code before its test artifacts exist and fail for the expected
reason. The task report must paste the failing output (`AI-6`, `T-21`).

The only exception is scaffolding that has no behavior. `T-001` is the current example.
The gate output is its evidence.

Every acceptance criterion maps to at least one test ID. A criterion without a test is a
defect in the task.

## 2. Test layers

| Layer | Purpose | Location | Command | Speed |
|-------|---------|----------|---------|-------|
| **Unit** | Functional behavior of one package, aggregate or pure function. Every behavior has a happy and an unhappy case. | `_test.go` beside the code | `make test` | Under 60 s |
| **Contract** | Fix the UI/server payloads, SSE events, error mapping and exit codes. Every contract has a success and an error case. | `internal/transport/contract` plus golden files | `make test-contract` | Seconds |
| **E2E** | Data flow and control flow across the real binary, SQLite and fake external services. Every flow has a success and a failure variant. | `e2e/` with `//go:build integration` | `make test-e2e` | Minutes |
| **Regression** | Every E2E flow stays in the suite. A bug fix adds a failing test first | Same as above | Same as above | Minutes |

### 2.1 Unit tests

- Standard library `testing` only (`T-1`).
- Table-driven where cases share a shape (`T-6`).
- Hand-written fakes, no generated mocks (`T-8`, `T-10`).
- No database, network, filesystem or clock in unit tests.
- Inject a `Clock` or `now func() time.Time` (`T-15`).
- Coverage floor is 80% per tested package (`T-18`).

### 2.2 Contract tests

The contract source is the confirmed transport component
([`01-transport.md`](../20-system-design/c4/03-components/01-transport.md)) and the
context contracts in [`context-map.md`](../20-system-design/ddd/strategic/context-map.md)
§6.1.

- Golden JSON files are the request and response cases.
- The server tests encode the golden responses. The CLI client tests decode the same files.
- SSE cases use an ordered event fixture, not a live stream.
- Error cases assert the HTTP status and the CLI exit code together.
- A golden change is a contract change. It needs review, not an auto-update.
- No Pact server or external contract tool. The repository uses stdlib tests and files.

| ID | Contract | Happy case | Unhappy case |
|----|----------|------------|--------------|
| `CT-01` | Health | `GET /v1/health` returns version and schema version | Server unreachable or schema mismatch |
| `CT-02` | Onboarding | OAuth completes, subscriptions import, interests save | Consent denied, revoked token, import fails |
| `CT-03` | Feed and SSE | Batch streams cards to `batch.done` | Quota stops the batch, stream disconnects, invalid event |
| `CT-04` | Gestures | Left, right, down, up, save and reason update the cursor | Duplicate gesture, stale session, invalid reason |
| `CT-05` | Library | List, save, download, remove and open | Download fails, file missing, player missing |
| `CT-06` | Diagnostics | `doctor` and `stats` report state | Unreachable service, unreadable database |
| `CT-07` | Rating | Weekly value is stored once | Duplicate week, invalid value |
| `CT-08` | Errors and exit codes | Each success maps to `0` | Each domain error maps to its `CLI-6` code |

### 2.3 E2E tests

E2E tests run the built binary against a temporary data directory. They use real SQLite
and fake external services:

- `httptest` for the YouTube API and the LLM endpoint.
- A fake `yt-dlp` script that returns a fixed transcript or fails on demand.
- A fake player script that records the file path.
- A temporary `YTFEED_DATA_DIR`.
- A fixed clock where the product allows it.

The tests carry `//go:build integration` (`T-13`). `make test-e2e` runs
`go test -tags=integration ./e2e/...`. It does not need Docker. The existing
`make test-integration` target stays for future container-backed tests.

| ID | Flow | Happy path | Unhappy path |
|----|------|------------|--------------|
| `E2E-01` | Onboarding to first card | Consent, import and interests succeed (`F1`…`F5`, `F13`) | Consent denied or import fails |
| `E2E-02` | Batch pipeline | Cards are discovered, analyzed and streamed (`F6`…`F14`) | Quota, no candidates or invalid model output (`F24`, `F25`) |
| `E2E-03` | Gesture, cold period, resume | Gesture commits and resume keeps the cursor (`F16`…`F19`, `F21`) | Duplicate gesture, stale lease or taste failure |
| `E2E-04` | Library download and playback | Save, download and open succeed (`F17`, `F20`) | Download fails, disk full or player missing |
| `E2E-05` | Degradation and cached feed | Cached cards and library are served (`F7`, `F24`) | No cached feed or unreadable database |
| `E2E-06` | Crash resume without re-fetch | Restart resumes from the stage (`F25`) | Corrupt stage or missing transcript |
| `E2E-07` | Weekly rating | One rating is stored (`F21`) | Duplicate week or invalid value |

## 3. Per-task artifacts

Each task creates the listed artifacts before code. Every ID is a **pair**: `-H` for the
happy path and `-U` for the unhappy path. A dash means the task has no behavior of that
type. `Gate` means the task only changes the build or harness.

| Task | Unit | Contract | E2E |
|------|------|----------|-----|
| `T-001` | — | — | Gate: `make check` |
| `T-001a` | Harness self-test | Golden fixtures load | Harness smoke |
| `T-002` | `UT-002` config validation | `CT-01` | — |
| `T-003` | `UT-003` schema, backup | — | `E2E-02` |
| `T-004` | `UT-004` token, loopback | `CT-01`, `CT-02` | — |
| `T-005` | — | `CT-08` exit mapping | `E2E-01` |
| `T-006` | `UT-006` card order | `CT-03` | `E2E-02` |
| `T-007` | `UT-007` signal and cold period | `CT-04` | `E2E-03` |
| `T-010` | `UT-010` PKCE and state | `CT-02` | `E2E-01` |
| `T-011` | `UT-011` token lifecycle | `CT-02` | `E2E-01` |
| `T-012` | `UT-012` import idempotency | `CT-02` | `E2E-01` |
| `T-013` | `UT-013` interest validation | `CT-02` | `E2E-01` |
| `T-014` | `UT-014` onboarding steps | `CT-02` | `E2E-01` |
| `T-020` | `UT-020` profile invariants | — | — |
| `T-021` | `UT-021` signal and weight | `CT-04` | `E2E-03` |
| `T-022` | `UT-022` cold-period rules | `CT-04` | `E2E-03` |
| `T-023` | `UT-023` reason picker | `CT-04` | `E2E-03` |
| `T-024` | `UT-024` snapshot | `CT-03` | — |
| `T-030` | `UT-030` query planner | — | `E2E-02` |
| `T-031` | `UT-031` YouTube adapter | — | `E2E-02` |
| `T-032` | `UT-032` quota rules | — | `E2E-05` |
| `T-033` | `UT-033` dedupe and balance | — | `E2E-02` |
| `T-034` | `UT-034` staging | — | `E2E-06` |
| `T-035` | `UT-035` schema validation | `CT-03` | `E2E-02` |
| `T-036` | `UT-036` failure states | — | `E2E-02`, `E2E-05` |
| `T-037` | `UT-037` pipeline order | — | `E2E-02`, `E2E-06` |
| `T-040` | `UT-040` order and cursor | `CT-03` | `E2E-02` |
| `T-041` | `UT-041` session and lease | `CT-04` | `E2E-03` |
| `T-042` | `UT-042` gesture outcomes | `CT-04` | `E2E-03` |
| `T-043` | `UT-043` SSE and JSON | `CT-03` | `E2E-02` |
| `T-044` | `UT-044` library states | `CT-05` | `E2E-04` |
| `T-045` | `UT-045` player launch | `CT-05` | `E2E-04` |
| `T-050` | `UT-050` doctor output | `CT-06` | — |
| `T-051` | `UT-051` stats | `CT-06` | — |
| `T-052` | `UT-052` trace fields | — | `E2E-02` |
| `T-053` | `UT-053` weekly rating | `CT-07` | `E2E-07` |
| `T-054` | `UT-054` degradation | — | `E2E-05`, `E2E-06` |
| `T-055` | Coverage final | Every `CT-*` | Every `E2E-*` |

## 4. Definition of ready and done

A task is **ready** when:

- The acceptance criteria are written as test cases.
- Every behavior has a happy case and an unhappy case.
- The test files and golden files exist.
- The new tests fail for the expected reason.
- The failure output is pasted in the task report.

A task is **done** when:

- The new tests pass, including both paths.
- The full unit and contract suites pass.
- The E2E suite passes for the affected flow.
- No test is skipped without a linked issue (`T-23`).
- Coverage stays at or above 80% for tested packages.
- The backlog and `tasks.md` are updated.

## 5. Regression policy

1. Every E2E flow is permanent. Do not delete a flow to make a change pass.
2. A bug fix starts with an unhappy-path test that fails on the old code (`T-20`).
3. A contract change updates the golden file and both sides. The diff is reviewed.
4. A flaky test is fixed or removed with a reason. It is never ignored.
5. The regression suite runs before every task is marked `DONE`.

## 6. Fake services

| Service | Fake | Rule |
|---------|------|------|
| YouTube API | `httptest.Server` with fixtures | No real API in tests (`T-14`). |
| LLM | `httptest.Server` returning valid or invalid JSON | No real model in unit or contract tests. |
| `yt-dlp` | Script on `PATH` | Deterministic transcript, failure or missing binary. |
| Player | Script on `PATH` | Records the path. Never opens a real player. |
| Clock | Injected `Clock` or `now` | No `time.Sleep` (`T-16`). |
| Filesystem | `t.TempDir()` | Clean per test. |

## 7. Open choices

| Choice | Decision | Reason |
|--------|----------|--------|
| Contract tool | Golden files and stdlib tests | No extra dependency (`H-10`). |
| E2E tag | `integration` | `G-5` allows only this tag. |
| E2E runner | `make test-e2e` | Distinct from the future Docker integration target. |
| Coverage tool | `go test -cover` | No external service. |
| Test data | Fixtures in the repository | Reviewable and deterministic. |
