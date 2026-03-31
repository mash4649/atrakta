# Plugin SDK Interface Definition

The plugin SDK is a declarative contract for extension developers.
It does not define a runtime loader or execution protocol.
It defines the shape of a plugin manifest that Atrakta can inspect, validate, and order.

## Version

- `schema_version`: `plugin-sdk.v1`

## Required Fields

- `id`: stable plugin identifier
- `name`: human-readable name
- `kind`: must be `plugin`
- `targets`: one or more of:
  - `adapter`
  - `projection`
  - `operations`
- `entrypoint`: how the host should invoke the plugin
- `capabilities`: declared plugin capabilities
- `permissions`: safety constraints

## Safety Constraints

- Plugins must not mutate core contracts directly.
- Plugins must not write canonical state directly.
- Plugins stay in read-only / advisory mode unless the host explicitly enables a controlled action.
- `can_mutate_core_contract` and `can_write_canonical` are always `false`.

## Entrypoints

- `binary`: an executable path or command
- `go-package`: a Go package reference that a host may load through a separate integration layer

## Example

```json
{
  "schema_version": "plugin-sdk.v1",
  "id": "projection-html",
  "name": "HTML Projection Plugin",
  "kind": "plugin",
  "targets": ["projection"],
  "entrypoint": {
    "type": "binary",
    "value": "./bin/projection-html"
  },
  "capabilities": ["render-html", "preview-static"],
  "permissions": {
    "can_mutate_core_contract": false,
    "can_write_canonical": false,
    "can_block_execution": false
  },
  "metadata": {
    "description": "Renders canonical projection output as HTML."
  }
}
```

## Relationship to Extension Boundary

- The extension boundary determines where plugins may participate.
- The plugin SDK is the contract that declares those participation targets.
- See [Extension Boundary and Evaluation Order](extension-boundary.md).
