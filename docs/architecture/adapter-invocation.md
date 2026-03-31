# Adapter Invocation Contract

This document defines how wrappers/hooks invoke Atrakta core.

## Scope

- Adapters are thin invokers.
- Core execution primitive is `atrakta run`.
- Adapters must not depend on internal command composition.
- v0 adapters target refreshed run-contract behavior, not `v0.14.1` parity.
- Adapters target semantic portability, not identical UX across surfaces.
- Even if `atrakta start` exists for humans/IDE entry, adapters MUST invoke `atrakta run`, not `atrakta start`.

## Invocation

Required:

- call `atrakta run`

Optional flags:

- `--project-root <dir>`
- `--interface <id>`
- `--non-interactive`
- `--json`
- `--apply`
- `--approve`

Adapter environment variables:

- `ATRAKTA_TRIGGER_INTERFACE`
- `ATRAKTA_NONINTERACTIVE`

Resolution order:

1. explicit CLI flags
2. adapter environment variables
3. runtime detect

## Exit Code Handling

- `0`: success
- `1`: runtime/contract error
- `2`: `NEEDS_INPUT` (adapter should request explicit input and retry)
- `3`: `NEEDS_APPROVAL` (adapter should collect approval and retry with `--approve`)

Recommended adapter loop:

1. invoke `atrakta run --json`
2. if exit `2`, collect missing input and retry
3. if exit `3`, collect approval and retry with `--approve`
4. stop on `0` or unrecoverable `1`

## JSON Contract

When `--json` is used, adapters must parse:

- `schemas/operations/run-output.schema.json`

The output may include:

- `planned_mutations[]`
- `applied_mutations[]`
- `required_inputs[]` (optional)
- `approval_scope` (optional)
- `portability`
- `resolved_projection_targets[]`
- `degraded_surfaces[]`
- `missing_projection_targets[]`
- `portability_status`
- `portability_reason`

Mutation object contracts:

- `schemas/operations/mutation-proposal.schema.json`
- `schemas/operations/mutation-decision-envelope.schema.json`

## Non-Goals

- Adapters do not write canonical/state directly.
- Adapters do not call resolver internals directly.
- Adapters do not promise identical prompts or interaction flow.
