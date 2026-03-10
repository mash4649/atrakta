# 04. Extension Verification

[English](./04_extension_verification.md) | [日本語](../../ja/04_品質/04_Extension検証.md)

## Scope

Extension verification covers MCP/plugin/skill/workflow/hook projection consistency and compatibility checks.

## Baseline

Until dedicated extension verify scripts are implemented, use existing CI verification as regression baseline.

## Planned Additions

- extension projection completeness checks
- unsupported-field warning checks (no silent ignore)
- brownfield append/include idempotency checks
- projection repair scope checks (managed-only)

## Expected Properties

- deterministic output from canonical contract
- explicit warning visibility for unsupported projection targets
- no destructive overwrite of user-managed content
