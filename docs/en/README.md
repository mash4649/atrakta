# Atrakta Documentation

[English](./README.md) | [日本語](../ja/README.md)

This is the English documentation set for the current Go implementation.

- Target version: `v0.14.1` ([VERSION](../../VERSION))
- Last updated: `2026-03-11`

## Structure

- `01_overview`: purpose, design principles, current scope
- `02_spec`: CLI spec, execution flow, data model, sync policy, parity contract, extension surface
- `03_operations`: onboarding, daily operations, troubleshooting, update/GC/distribution, brownfield integration, release checklist, issue bootstrap
- `04_quality`: verification commands, benchmark criteria, parity verification, extension verification

## Recommended Reading Order

1. `01_overview/01_overview.md`
2. `02_spec/01_cli_spec.md`
3. `02_spec/06_parity_contract.md`
4. `02_spec/07_extension_surface.md`
5. `03_operations/01_setup.md`
6. `03_operations/07_brownfield_integration.md`
7. `03_operations/09_release_checklist.md`
8. `03_operations/08_github_issue_bootstrap.md`
9. `04_quality/03_parity_verification.md`
10. `04_quality/04_extension_verification.md`
11. `04_quality/01_verification_commands.md`
12. `../../CHANGELOG.md`

## Documentation Policy

- Keep one source of truth per feature.
- Do not keep legacy docs in-tree; use Git history and `CHANGELOG.md` when needed.
- Remove any statement that drifts from implementation.
