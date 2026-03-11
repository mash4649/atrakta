# CAPABILITY_REGISTRY_SPEC

Registry path: `.atrakta/capabilities/registry.json`

## Schema
- `v`
- `entries[]`
  - required:
    - `id`
    - `kind` (`skill|recipe_candidate|reference_memory|gateway|api|unsupported`)
    - `path`
    - `provenance`
  - optional extension metadata:
    - `source_type`
    - `source_path`
    - `import_batch_id`
    - `analysis_status`
    - `quarantine_reason`
    - `conversion_status`
    - `default_memory_surface`

## Compatibility
- Existing entries remain readable without migration.
- New metadata fields are optional and fail-open for older rows.
- Capability kind set is fixed (no kind expansion in this extension).
