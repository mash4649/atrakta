# resolve_guidance_precedence

Signature:

`resolve_guidance_precedence(set) -> ordered_list`

Implemented in:

- `resolver.go` as `ResolveGuidancePrecedence([]GuidanceItem) common.ResolverOutput`

Rules:

- Precedence: policy > workflow > skill > repo_map > tool_hint
- Canonical policy always wins
- Unmapped legacy guidance is advisory
- Repo map and tool hint cannot override decisions or approvals
- Output follows `schemas/operations/inspect-output.schema.json`
