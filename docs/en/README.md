# Atrakta Documentation

[English](./README.md) | [日本語](../ja/README.md)

This is the English documentation set for the current Go implementation.

- Target version: `v0.14.0` ([VERSION](../../VERSION))
- Last updated: `2026-03-04`

## Structure

- `01_overview`: purpose, design principles, current scope
- `02_spec`: CLI spec, execution flow, data model, sync policy
- `03_operations`: onboarding, daily operations, troubleshooting
- `04_quality`: tests, benchmarks, cross-build verification

## Recommended Reading Order

1. `01_overview/01_overview.md`
2. `02_spec/01_cli_spec.md`
3. `03_operations/01_setup.md`
4. `03_operations/04_update_guide.md`
5. `03_operations/06_distribution_guide.md`
6. `03_operations/05_gc_operations.md`
7. `04_quality/01_verification_commands.md`
8. `../../CHANGELOG.md`

## Documentation Policy

- Keep one source of truth per feature.
- Do not keep legacy docs in-tree; use Git history and `CHANGELOG.md` when needed.
- Remove any statement that drifts from implementation.
