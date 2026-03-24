# Concept Coverage Matrix (Refresh Track)

This matrix defines how v0 refresh concepts cover core operational intents that
were historically addressed in earlier Atrakta lines.

## Coverage Rule

- Goal is concept coverage and behavioral sufficiency.
- Command/data parity is not required in this track.

## Intent Mapping

| Operational intent | v0 concept surface | Verification |
|---|---|---|
| first time safe onboarding | `run` onboarding branch (`detect -> propose -> accept`) | `cmd/atrakta` run tests (approval gate + persistence checks) |
| deterministic day to day execution | `run` normal branch + ordered resolver pipeline | deterministic replay tests + run E2E |
| explicit write control | `--apply` + `--approve` gate + managed-only mutation | run apply tests + mutation envelope validation |
| audit traceability | append-only audit chain (`run_execute`) | audit append/verify tests |
| failure recovery posture | exit code contract (`0/1/2/3`) + next action hints | adapter loop tests and run schema validation |

## Acceptance Criteria

v0 refresh is accepted for this scope when:

1. all intents above are demonstrated by tests,
2. `run` remains the single operational entrypoint,
3. safety invariants remain fail-closed under missing input/approval.
