# Extension Boundary and Evaluation Order

## First-Pass Taxonomy

- capability_adapters
- orchestration_assets
- runtime_hook

## Second-Pass Mapping

- MCP -> adapter, capability source only
- plugin -> adapter, projection, operations only
- `skill_asset` -> asset layer only
- `workflow_binding` -> asset plus orchestration only
- `runtime_hook` -> runtime and operations lifecycle only

## Plugin SDK

- Plugin SDK interface definition: [plugin-sdk.md](plugin-sdk.md)

## MCP Integration

- MCP server contract: [mcp-server.md](mcp-server.md)

None can mutate core contracts directly.
Diagnostics assets cannot constrain execution policy.
Hooks cannot directly change canonical state.

## Evaluation Order

1. `canonical_policy`
2. `workflow_binding`
3. `skill_asset`
4. `ide_rules`
5. `runtime_hook`
6. projection plugin

## Resolver API

`resolve_extension_order(set) -> ordered_extensions`
