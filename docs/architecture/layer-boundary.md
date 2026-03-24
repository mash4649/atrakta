# Layer Boundary Contract

## Goal

Fix ownership between Core, Canonical, and Extension layers.

## Allowed and Forbidden by Layer

### Core Contract

Allowed:

- Request
- Decision
- Result
- Error

Forbidden:

- Runtime profile
- Asset reference
- Derived projection body
- Task state as projection source
- Audit event as decision source

### Canonical Store

Allowed:

- Capability
- `canonical_policy`
- Task State
- Audit Event

Forbidden:

- Derived projection body
- Runtime-only temporary state
- Direct extension executable assets

### Extension Asset

Allowed:

- Runtime profile
- `repo_docs`
- `skill_asset`
- `workflow_binding`
- Provenance

Forbidden:

- Direct `canonical_policy` override
- Direct mutation of core contracts

## Owner Mapping

- Request -> core
- Decision -> core
- Result -> core
- Error -> core
- Capability -> canonical
- `canonical_policy` -> canonical
- Task State -> canonical
- Audit Event -> canonical
- Runtime Profile -> extension
- `repo_docs` -> extension
- `skill_asset` -> extension
- `workflow_binding` -> extension
- Provenance -> extension

## Layer API

`classify_layer(item) -> core | canonical | extension`

Input requirements:

- `item.kind`
- `item.schema_id`

Output requirements:

- `decision` as one of `core`, `canonical`, `extension`
- `reason`
- `evidence`

## Forbidden Rule

Any item that crosses layer ownership MUST be rejected at mutation time and reported through inspect output.
