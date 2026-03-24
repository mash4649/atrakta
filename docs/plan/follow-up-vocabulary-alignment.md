# Follow-up Task Memo: Vocabulary Alignment

## Context

Architecture docs were aligned to the vocabulary defined in
`docs/architecture/surface-portability.md`.
This memo tracks remaining follow-up tasks for non-architecture docs.

## Follow-up Tasks

1. Align example payload vocabulary
   - Target: `docs/examples/sample-onboarding-proposal.json`
   - Check whether keys/values that use legacy surface names (for example
     `AGENTS.md`) should be represented as canonical portability vocabulary
     (`agents_md`, `ide_rules`, `repo_docs`, `skill_bundle`).
   - Confirm schema compatibility before changing example fields.

2. Cross-doc terminology consistency sweep
   - Re-scan `docs/README.md`, `docs/plan/*.md`, and `docs/decisions/*.md`
     for legacy labels such as `Policy`, `Workflow`, `Skill`, `Repo Map`,
     and `IDE rules`.
   - Keep human-readable file-name references where needed, but prefer
     canonical vocabulary labels for concept names.

3. Add a vocabulary note for contributors
   - Add a short "preferred terms" section to docs entrypoints
     (for example `docs/README.md`) to reduce future wording drift.

## Done Criteria

- Non-architecture docs use the same concept labels as
  `surface-portability.md`.
- Example JSON remains valid for current schemas and tests.
- Terminology guidance is documented in at least one contributor-facing doc.
