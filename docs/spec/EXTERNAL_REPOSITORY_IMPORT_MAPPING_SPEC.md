# EXTERNAL_REPOSITORY_IMPORT_MAPPING_SPEC

## Purpose
Define deterministic import classification for external repository assets.

## Scope
- Input source is **local directory only**.
- Remote URL clone/pull is out of scope for this spec.
- Import extends existing managed behavior (`append`, `include`, managed block) without replacing current core specs.

## Classification Rules
Import normalizer must map files into one of these capability kinds:
- `skill`
- `recipe_candidate`
- `reference_memory`
- `gateway`
- `api`
- `unsupported`

No additional capability kind is introduced by import.

## Deny Rules
The import classifier must deny or explicitly quarantine these inputs:
- secret-like files (`.env`, credential-like names, key blobs)
- binary blobs
- unsupported executable blobs (e.g. shell/batch/binary artifacts)

## Provenance Rules
Every imported capability must keep provenance metadata:
- source type (`local_directory`)
- source path
- import batch id
- source relative path
- content hash
- imported timestamp

## Deterministic Normalization Rules
- File traversal order must be deterministic.
- Capability ID generation must be deterministic from normalized inputs.
- Classification must not rely on hidden mutable state.
- Same source state must produce the same capability set and content hashes.

## Quarantine-first Model
- `import complete` is **not** execution permission.
- Imported capabilities enter quarantine-first state.
- Review/conversion/promotion are separate explicit gates.

## Compatibility
This spec must stay consistent with:
- `IMPORT_PIPELINE_SPEC.md`
- `CAPABILITY_REGISTRY_SPEC.md`
