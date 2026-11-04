# Rules coverage audit

## Status

In review for the final gate.

## Scope

Every `A-*` rule in `02-architecture.md`, every `D-*` rule in
`04-data-persistence.md` and every `CLI-*` rule in `08-cli-and-ux.md`. A rule is either
**placed** in a design artifact or explicitly marked **non-architectural**. A rule may
also be placed by an ADR and a tactical model.

## Result

| Rule group | Total | Placed | Non-architectural | Unplaced | Contradictions |
|------------|------:|-------:|------------------:|---------:|---------------:|
| `A-*` | 24 | 24 | 0 | 0 | 0 |
| `D-*` | 26 | 24 | 2 | 0 | 0 |
| `CLI-*` | 30 | 30 | 0 | 0 | 0 |
| **Total** | **80** | **78** | **2** | **0** | **0** |

The counts include the lettered rules (`A-10a`, `A-10b`, `A-12a`, `A-12b`, `D-11a`,
`D-11b`, `CLI-14a`, `CLI-20a`).

## `A-*` rules

| Rule | Placement | Status |
|------|-----------|--------|
| `A-1` | `ADR-0003`. `README.md` global invariants | Placed |
| `A-2` | `c4/03-components/01-transport.md`. `ADR-0003` | Placed |
| `A-3` | `ADR-0003`. all tactical `README.md` | Placed |
| `A-4` | `ADR-0002`. `ddd/tactical/*/repository.md` | Placed |
| `A-5` | `ADR-0002`. repository contracts | Placed |
| `A-6` | `ADR-0002`. `D-3`. repository contracts | Placed |
| `A-7` | `ADR-0003`. `ddd/strategic/context-map.md` | Placed |
| `A-8` | `ADR-0003`. `context-map.md` §5.3 | Placed |
| `A-9` | `ADR-0001`. `c4/02-container-diagram.md` | Placed |
| `A-10` | `ADR-0001`. `flows/session-and-degradation.md` | Placed |
| `A-10a` | `ADR-0001`. `c4/02` trust boundary | Placed |
| `A-10b` | `ADR-0001`. `c4/03-components/01-transport.md` | Placed |
| `A-11` | `ADR-0001`. `ddd/tactical/feed/aggregates.md` §2 | Placed |
| `A-12` | `ADR-0004`. `c4/02` | Placed |
| `A-12a` | `ADR-0004`. `ddd/tactical/analysis/repository.md` | Placed |
| `A-12b` | `ADR-0004`. `ddd/tactical/analysis/aggregates.md` | Placed |
| `A-13` | `c4/02`. `c4/03-components/01-transport.md` | Placed |
| `A-14` | `ADR-0005`. `ddd/tactical/evented-patterns.md` | Placed |
| `A-15` | `ADR-0002`. `c4/03-components/01-transport.md` | Placed |
| `A-16` | `c4/03-components/01-transport.md` | Placed |
| `A-17` | `c4/03-components/03-discovery-analysis.md` | Placed |
| `A-18` | `ADR-0002` | Placed |
| `A-19` | `ADR-0004`. `ddd/tactical/analysis/aggregates.md` | Placed |
| `A-20` | `ADR-0004`. `ddd/tactical/taste/aggregates.md` | Placed |

## `D-*` rules

| Rule | Placement | Status |
|------|-----------|--------|
| `D-1` | `ADR-0002`. all `repository.md` files | Placed |
| `D-2` | `ADR-0002`. repository contracts | Placed |
| `D-3` | `ADR-0002`. repository contracts | Placed |
| `D-4` | `ADR-0002` | Placed |
| `D-5` | `ADR-0002` | Placed |
| `D-6` | `ADR-0002` | Placed |
| `D-7` | `ADR-0002` | Placed |
| `D-8` | `ADR-0002`. `ddd/tactical/*/repository.md` | Placed |
| `D-9` | `ADR-0002` | Placed |
| `D-10` | `ADR-0002` | Placed |
| `D-11` | `ADR-0002` | Placed |
| `D-11a` | `ADR-0002`. `ADR-0005` | Placed |
| `D-11b` | `ADR-0006`. `c4/01` failure table | Placed |
| `D-12` | `ADR-0002` | Placed |
| `D-13` | `ddd/tactical/*/repository.md`. value objects | Placed |
| `D-14` | `ddd/tactical/*/repository.md` | Placed |
| `D-15` | `ddd/tactical/*/repository.md` | Placed |
| `D-16` | `ddd/tactical/taste/aggregates.md` §4. `feed/aggregates.md` §4 | Placed |
| `D-17` | All `repository.md` files | Placed |
| `D-18` | `ADR-0002`. logical schemas | Placed |
| `D-19` | `discovery`, `taste`, `feed` aggregates | Placed |
| `D-20` | `taste/aggregates.md` §4. `feed/aggregates.md` §5 | Placed |
| `D-21` | `taste/aggregates.md` §3 | Placed |
| `D-22` | `context-map.md` §7. all logical schemas | Placed |
| `D-23` | `07-testing.md` test policy | Non-architectural |
| `D-24` | `07-testing.md` test policy | Non-architectural |

## `CLI-*` rules

| Rule | Placement | Status |
|------|-----------|--------|
| `CLI-1` | `c4/03-components/01-transport.md` | Placed |
| `CLI-2` | `01-transport.md` | Placed |
| `CLI-3` | `01-transport.md` | Placed |
| `CLI-4` | `01-transport.md` | Placed |
| `CLI-5` | `01-transport.md` | Placed |
| `CLI-6` | `ADR-0006`. `01-transport.md` | Placed |
| `CLI-7` | `ADR-0006`. `01-transport.md` | Placed |
| `CLI-8` | `01-transport.md` | Placed |
| `CLI-9` | `01-transport.md` | Placed |
| `CLI-10` | `01-transport.md` renderer | Placed |
| `CLI-11` | `01-transport.md` renderer | Placed |
| `CLI-12` | `01-transport.md` renderer | Placed |
| `CLI-13` | `01-transport.md` renderer | Placed |
| `CLI-14` | `01-transport.md` renderer | Placed |
| `CLI-14a` | `ddd/tactical/feed/aggregates.md` §1.4 | Placed |
| `CLI-15` | `01-transport.md` renderer | Placed |
| `CLI-16` | `01-transport.md` renderer | Placed |
| `CLI-17` | `01-transport.md` | Placed |
| `CLI-18` | `taste/aggregates.md` §2. `feed/aggregates.md` §5 | Placed |
| `CLI-19` | `ddd/tactical/library/model.md` | Placed |
| `CLI-20` | `01-transport.md`. `taste/aggregates.md` §3 | Placed |
| `CLI-20a` | `taste/aggregates.md` §3.2 | Placed |
| `CLI-21` | `ddd/tactical/feed/aggregates.md` §2 | Placed |
| `CLI-22` | `ADR-0006`. `01-transport.md` | Placed |
| `CLI-23` | `library/model.md`. `01-transport.md` | Placed |
| `CLI-24` | `feed/use-cases.md`. `01-transport.md` | Placed |
| `CLI-25` | `identity/model.md` §5 | Placed |
| `CLI-26` | `identity/model.md` §5 | Placed |
| `CLI-27` | `identity/model.md` §5. `feed/use-cases.md` | Placed |
| `CLI-28` | `feed/aggregates.md` §3 | Placed |

## Contradictions and gaps

| Item | Result |
|------|--------|
| Unlinked contradiction with a rule | None. |
| Rule fork without an ADR | None. |
| Rule that needed a PRD change | `F9` transcript staging was amended on 2026-11-02 (`OQ-05`). |
| Rule wording ambiguity | `A-7` "rubric weights" is resolved by `OQ-20`. |
| Non-architectural rules | `D-23` and `D-24` are test practice, not design. |
| Open rule risk | `OQ-01` quota exit code and `OQ-07` rubric location remain open. They do not contradict a rule. |

## Verdict

**No blocking contradiction.** Every `A-*`, `D-*` and `CLI-*` rule is placed or marked
non-architectural. The two PRD-linked changes and the one wording ambiguity are recorded
in the open-questions register. The design does not silently fork a rule.
