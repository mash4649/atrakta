# Execution Flow

[English](./02_execution_flow.md) | [日本語](../../ja/02_仕様/02_実行フロー.md)

## `start` Sequence (Fast Path First, Strict Path On Demand)

1. Load/init contract and normalize settings
2. Evaluate Start Snapshot Fast Gate (`runtime/meta.v2.start_fast_v2`)
3. If fast hit and managed artifacts are intact:
   - Run `VerifyChainCached`
   - Record `start_fast_hit`, `step(DONE)`, and checkpoint
   - Exit without strict pipeline
4. On fast miss or drift, escalate to strict path:
   - Init default prompt policy file when needed
   - Verify events chain (`schema_version = 2`)
   - Run git setup and pre snapshot
5. Ensure `AGENTS.md` and resolve context (`nearest_with_import`)
6. Load/refresh repo map and record `repo_map` event
7. Resolve routing and initialize `progress.json` when needed
8. Resolve interfaces (explicit -> trigger -> auto-last -> detect -> prompt)
9. Run detect
10. Build projections + subworker phaseA + single_writer_queue
11. Build/save plan and task DAG
12. Apply prompt policy conditionally
13. Request approval when required
14. Run security preflight (`read_only` blocks mutation)
15. Run apply (task graph topological order)
16. Run gate (safety + quick/heavy + route quality)
17. Update state/progress, write git post snapshot/checkpoint, record step event
18. Save `start_fast_v2` snapshot for the next run
19. After `DONE`, trigger threshold-based auto GC asynchronously (outside critical path)

## Typical Strict Escalation Conditions

- Fast snapshot mismatch (contract hash / workspace stamp / interface set / feature / config key)
- Strict interval exceeded (default 10 minutes)
- Managed artifact missing/drifted
- Fast gate unavailable (e.g. corrupted meta)

## Typical Block Conditions

- Active feature and requested feature mismatch
- Interface unresolved in non-interactive run (`NEEDS_INPUT`)
- Context import cycle or depth overflow
- Missing/invalid required prompt policy
- Security profile and required permission mismatch
- Apply failure
- Gate failure
- Unrecoverable corruption in events/state
