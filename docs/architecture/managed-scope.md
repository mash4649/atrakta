# Managed Scope and Mutation Policy

## Scope Taxonomy

- managed_block
- managed_include
- generated_projection
- unmanaged_user_region
- proposal_patch_only

## Mutation Policy

- managed_block: managed apply allowed with decision envelope
- managed_include: append/include preferred
- generated_projection: replace allowed for generated output only
- unmanaged_user_region: implicit mutate forbidden
- proposal_patch_only: proposal-only, explicit approval required

`repo_docs`: append/include preferred.
Tool config: include/proposal-only preferred.
`canonical_policy` store: replace disallowed except managed-only paths.
Existing user rules: ambiguity falls back to proposal-only.

## Migration Principle

Use dual-read and single-write during migration.
No reverse auto-sync from generated artifacts to canonical store.

## Resolver API

`check_mutation_scope(target) -> scope_decision`
