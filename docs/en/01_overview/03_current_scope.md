# Current Scope

[English](./03_current_scope.md) | [日本語](../../ja/01_全体/03_現状スコープ.md)

## Implemented (Phase1/Phase2)

- Auto-create `AGENTS.md` (when missing)
- Sync levels 0/1/2 with proposal-only mode
- Optional projection templates (`contract-json`, `atrakta-link`)
- Extended gate quick/heavy checks
- Migration replay determinism verification
- Cross-build verification for Linux/Windows
- Linux binary runtime verification (via Docker)
- Autonomous subworker decision (Single Writer, Multi Worker)
- Deterministic merge via `single_writer_queue`
- DAG task graph execution
- Lock + atomic updates for `state/events`
- Non-destructive Git automation (auto init/snapshot/checkpoint)
- Fixed events schema v2 (legacy schema fail-closed)
- Edit safety precheck (Go AST / JSON parse)
- Run checkpoint + `resume`
- Threshold-based `.tmp` GC + proposal-only `events` GC
- Always-loaded `context.conventions` + read-only protection
- Staged conventions loading for token efficiency
- `repo_map` budget/refresh control with runtime cache
- Start Snapshot Fast Gate (`start_fast_v2`)
- Faster `start` with cached chain verification + batched/group event appends
- Hybrid wrapper stamp (quick check + deep fallback + periodic deep sampling)
- Runtime metrics recording for `start`/`doctor`
- Automated releases (CalVer tag + artifacts + auto release notes)

## Not Yet Implemented / Future Extensions

- Native Windows execution test in current environment
- Operational automation beyond planning for branch lanes (while keeping Single Writer principle)
- Quality gate tuning by language/project type
