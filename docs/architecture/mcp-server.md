# MCP Server Integration

MCP integration is the IDE-native adapter surface for Atrakta.
It is modeled as an adapter-class extension, not as a core execution primitive.

## Role

- expose diagnostics and inspection through an MCP server
- participate in extension ordering as an adapter-capable integration
- remain read-only with respect to canonical state

## Contract

- `schema_version`: `mcp-server.v1`
- `kind`: `mcp`
- `transport.type`: `stdio` or `http`
- `transport.command`: required launch command
- `capabilities`: one or more of:
  - `adapter`
  - `diagnostics`
  - `projection`
  - `inspection`

## Safety

- MCP servers must not mutate core contracts.
- MCP servers must not write canonical state directly.
- MCP servers can support diagnostics and projection inspection only.

## `.mcp.json`

Projects that use MCP can declare a local `.mcp.json` or `mcp.json`.
Atrakta detects those assets and resolves the `mcp` interface during runtime detection.

## Relationship to Adapter Invocation

MCP should invoke `atrakta run`, not `atrakta start`.
See [Adapter Invocation Contract](adapter-invocation.md).
