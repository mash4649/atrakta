# resolve_extension_order

Signature:

`resolve_extension_order(set) -> ordered_extensions`

Implemented in:

- `resolver.go` as `ResolveExtensionOrder([]Item) common.ResolverOutput`

Rules:

- evaluation order: canonical policy -> workflow -> skill -> tool hint -> runtime hook -> projection plugin
- first-pass groups: capability_adapters / orchestration_assets / runtime_hooks
- none can mutate core contracts directly
- diagnostics assets cannot constrain execution policy
- hooks cannot directly mutate canonical state
