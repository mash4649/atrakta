# resolve_operation_capability

Signature:

`resolve_operation_capability(command_or_alias) -> capability_decision`

Implemented in:

- `resolver.go` as `ResolveOperationCapability(Input) common.ResolverOutput`

Rules:

- canonical action classes: inspect_only / propose_only / apply_mutation
- aliases: doctor, parity, integration, repair
- BLOCK tier ceiling clamps to inspect_only
- PROPOSAL_ONLY tier ceiling clamps to propose_only
