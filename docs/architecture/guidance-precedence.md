# Guidance Strength and Precedence

## Strength Classes

- `authoritative_constraint`
- `orchestration_constraint`
- `executable_guidance`
- `advisory_map`
- `tool_hint`

## Surface Classes

- `decision`
- `orchestration`
- `mutation`
- `projection`
- `diagnostics`

## Mapping

- `canonical_policy` -> authoritative_constraint; decision, mutation
- `workflow_binding` -> orchestration_constraint; orchestration, mutation
- `skill_asset` -> executable_guidance; orchestration, mutation
- `repo_docs` -> advisory_map; projection, diagnostics
- `ide_rules` -> tool_hint; diagnostics

## Precedence Rule

1. `canonical_policy`
2. `workflow_binding`
3. `skill_asset`
4. `repo_docs`
5. `ide_rules`

`canonical_policy` always wins.
Reference asset is advisory unless mapped to `canonical_policy`.
`repo_docs` and `ide_rules` cannot override decisions or approvals.
Unmapped legacy guidance is advisory only.

## Resolver API

`resolve_guidance_precedence(set) -> ordered_list`

Output includes ranking, conflict reason, and next allowed action.

For portability of guidance carriers such as `agents_md` and `ide_rules`, see
[Surface Portability](surface-portability.md).
