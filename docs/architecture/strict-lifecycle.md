# Strict Lifecycle

## Scope Types

- request
- task
- workspace

## States

- normal
- guarded
- strict
- released

## Trigger Examples

- stale state
- unresolved capability
- unsupported projection surface
- missing approval
- workspace mismatch
- policy ambiguity
- instruction conflict
- audit guarantee shortfall

## Operation Rules

- guarded allows inspect and limited propose
- strict allows inspect and proposal-only unless explicitly lifted
- release requires explicit release condition
- strict is reversible through release path

## Transition Link

Failure tier output drives strict state transition through explicit transition table.
