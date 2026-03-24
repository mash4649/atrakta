# Audit Integrity and Retention Policy

## Integrity Levels

- A0: append-only only
- A1: append-only plus per-event integrity hash
- A2: append-only plus previous-hash chain
- A3: append-only plus chain verification and checkpoint

Integrity level is a logical contract independent from storage implementation.

## Retention and Protection

- append-only is mandatory
- core audit head is never GC target
- manifest and projection linkage are outside normal GC

## Archival Rule

- dry-run first
- destructive cleanup is proposal-only

Audit guarantee shortfall must connect to strict trigger and failure routing.

## Resolver API

`resolve_audit_requirements(action) -> audit_requirements_decision`
