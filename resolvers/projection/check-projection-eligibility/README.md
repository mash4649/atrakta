# check_projection_eligibility

Signature:

`check_projection_eligibility(source) -> allowed | conditional | forbidden`

Implemented in:

- `resolver.go` as `CheckProjectionEligibility(Source) common.ResolverOutput`

Rules:

- allowed: policy, repo_map, skill, workflow
- conditional: decision, result
- forbidden: task_state, audit_event
- conditional source cannot be decision root without canonical anchor
- output follows `schemas/operations/inspect-output.schema.json`
