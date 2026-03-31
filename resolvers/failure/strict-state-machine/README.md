# strict_state_machine

Signature:

`strict_state_machine(current_state, event, scope) -> next_state`

Implemented in:

- `resolver.go` as `Transition(StateInput) common.ResolverOutput`

States:

- normal
- guarded
- strict
- released

Rules:

- strict is releasable
- release needs explicit approval
- guarded and strict forbid direct apply actions
