# check_mutation_scope

Signature:

`check_mutation_scope(target) -> scope_decision`

Implemented in:

- `resolver.go` as `CheckMutationScope(Target) common.ResolverOutput`

Rules:

- scopes: managed_block / managed_include / generated_projection / unmanaged_user_region / proposal_patch_only
- implicit mutation is forbidden on unmanaged user region
- repo_map uses append/include preferred policy
- tool_config uses include/proposal-only preferred policy
- policy replace is disallowed except managed-only path
- reverse sync is proposal-only
