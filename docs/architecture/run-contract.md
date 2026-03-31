# Run Contract

## Purpose

`atrakta run` is the single execution primitive.
The machine-executable contract is `.atrakta/contract.json`.
Surface portability is defined separately in [Surface Portability](surface-portability.md).

Positioning:

- v0 is a refresh line, not a compatibility layer for `v0.14.1`.
- Behavioral sufficiency is prioritized over command/data parity.
- Legacy assets may be transformed later via explicit import/transform flows.

Harness entrypoint note:

- `atrakta run` is the execution primitive.
- `atrakta start` is the recommended human/IDE entrypoint and is defined as a harness-level alias built on top of `run`.
- Adapters and wrappers MUST continue invoking `atrakta run` per the adapter invocation contract.

- First run: onboarding flow (`detect -> propose -> accept`)
- Subsequent run: normal flow (`detect -> plan -> apply` variant)

Other commands (`onboard`, `accept`, `inspect`, `preview`, `simulate`) are
implementation parts and debug surfaces. The operational contract is `run`.

## Inputs

Optional:

- `--project-root <dir>` (default: detected from current directory)
- `--interface <id>`
- `--non-interactive`
- `--json`
- `--apply`
- `--approve` (explicit approval token for non-interactive write path)

Adapter-set environment variables:

- `ATRAKTA_TRIGGER_INTERFACE`
- `ATRAKTA_NONINTERACTIVE`

Priority:

- interface: `--interface` > `ATRAKTA_TRIGGER_INTERFACE` > detect > `NEEDS_INPUT`
- non-interactive: `--non-interactive` > `ATRAKTA_NONINTERACTIVE`

## Outputs

Exit codes:

- `0`: success (state update may or may not occur)
- `1`: runtime or contract error
- `2`: `NEEDS_INPUT` (insufficient input for deterministic continuation)
- `3`: `NEEDS_APPROVAL` (explicit approval required before write path)

Output channels:

- human-readable summary on stdout by default
- machine-readable envelope with `--json` (`schemas/operations/run-output.schema.json`)
- mutation details are emitted as `planned_mutations[]` / `applied_mutations[]`
- adapter hints may include `required_inputs[]` and `approval_scope`
- portability metadata includes `portability`, `resolved_projection_targets[]`,
  `degraded_surfaces[]`, `missing_projection_targets[]`, `portability_status`,
  and `portability_reason`

## Canonical Presence States

`run` checks project state before routing:

- `canonical_present`
- `onboarding_complete`
- `partial_state`
- `corrupt_state`

Current minimal canonical-present condition:

- `.atrakta/canonical/policies/registry/index.json` exists
- `.atrakta/contract.json` exists as the machine contract alongside canonical state

Additional state classes:

- `partial_state`: `.atrakta/state/onboarding-state.json` exists but canonical index is missing
- `corrupt_state`: canonical/state directory exists but required marker files are missing

If partial/corrupt state is detected, `run` returns `1` with a
diagnostic reason. `run` never silently repairs canonical state.

## Flow Branching

### No Canonical (Onboarding Path)

1. detect: onboarding proposal generation
2. propose: render proposal (text/json)
3. approval gate:
   - non-interactive or reject => `NEEDS_APPROVAL` (`3`)
   - approve => `accept` route
4. accept route writes canonical/state/audit

Write guarantee:

- no canonical/state/audit write before explicit approval
- if interface cannot be detected in onboarding path, `run` falls back to `generic-cli`

### Canonical Present (Normal Path)

1. load machine contract from `.atrakta/contract.json`
2. load canonical state (`canonical/policies/registry/index.json` and optional onboarding state)
3. detect current workspace surfaces
4. run ordered resolver pipeline (`inspect` baseline)
5. optional apply route builds managed mutation plans for generated managed targets:
   - `.atrakta/generated/repo-map.generated.json`
   - `.atrakta/generated/capabilities.generated.json`
   - `.atrakta/generated/guidance.generated.json`
6. resolve surface portability before managed apply
7. degraded or unsupported portability falls back to proposal-only and skips apply
8. apply executes only when explicitly approved (`--approve` or interactive approval)
9. update state/audit only for accepted/apply-success paths

Idempotency:

- if no action is required for same project + same effective input, `run`
  performs no write and returns `0`

## Lazy Persistence Policy

Persistence is event-driven only:

- onboarding `accept` success
- managed `apply` success
- machine contract refresh, when emitted, follows the same accept/apply gate

No other path writes canonical generated artifacts.

Audit policy:

- audit append is allowed for operational trace
- audit append is not treated as canonical/state mutation
- normal path appends `run_execute` audit event (A2) for replay/traceability

State update on apply:

- successful apply writes `.atrakta/state/run-state.json`
- successful `start`/`resume` (non-fast-path) updates session runtime files:
  - `.atrakta/state.json`
  - `.atrakta/progress.json`
  - `.atrakta/task-graph.json`

## Adapter Invocation Contract

See [Adapter Invocation Contract](adapter-invocation.md).
