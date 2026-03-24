# Operations Capability Model

## Action Classes

- inspect_only
- propose_only
- apply_mutation

## Canonical Capabilities

- inspect_health
- inspect_drift
- inspect_parity
- inspect_integration
- propose_repair
- apply_repair

## Legacy Alias Mapping

- doctor -> inspect_health
- parity -> inspect_parity
- integration -> inspect_integration
- repair -> propose_repair

## Failure Tier Ceiling

- BLOCK -> inspect_only
- PROPOSAL_ONLY -> propose_only
- DEGRADE_TO_STRICT -> propose_only
- WARN_ONLY -> apply_mutation

Implicit mutation is forbidden unless effective action class is apply_mutation.
