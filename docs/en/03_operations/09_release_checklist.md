# Release Checklist (Parity/Extensions/Brownfield)

[English](./09_release_checklist.md) | [日本語](../../ja/03_運用/09_リリースチェックリスト.md)

Use this checklist as a release gate. Any unchecked item blocks release.

## Required Gate Items

1. [ ] `docs/en/02_spec/06_parity_contract.md` and `docs/ja/02_仕様/06_Parity Contract.md` exist.
2. [ ] `docs/en/02_spec/07_extension_surface.md` and `docs/ja/02_仕様/07_Extension Surface.md` exist.
3. [ ] `.atrakta/contract.json` includes both `parity` and `extensions` sections (`v=1`).
4. [ ] `.atrakta/projections/manifest.json` is generated after projection/start.
5. [ ] `.atrakta/extensions/manifest.json` is generated after projection/start.
6. [ ] Native projections are produced for the major three:
   - Cursor (`.cursor/AGENTS.md`, `.cursor/rules/00-atrakta.mdc`)
   - Claude Code (`CLAUDE.md`, `.claude/*`)
   - Codex (`AGENTS.md`, `.codex/config.toml`)
7. [ ] `atrakta doctor --parity` detects drift correctly (blocking vs warning).
8. [ ] `atrakta doctor --integration` detects brownfield conflicts correctly.
9. [ ] AGENTS append/include flow is idempotent (no managed-block duplication).
10. [ ] Verification scripts pass:
    - `./scripts/dev/verify_parity.sh`
    - `./scripts/dev/verify_extensions.sh`
    - `./scripts/dev/verify_brownfield.sh`
    - `./scripts/dev/verify_projection_repair.sh`
11. [ ] Fast path / SLO / fail-closed invariants remain intact.
12. [ ] Windows native parity gate is green:
    - `./scripts/dev/verify_windows_native_parity.sh`

## Recommended Gate Run Order

```bash
./scripts/dev/verify_parity.sh
./scripts/dev/verify_extensions.sh
./scripts/dev/verify_brownfield.sh
./scripts/dev/verify_projection_repair.sh
./scripts/dev/verify_perf_gate.sh
./scripts/dev/verify_windows_native_parity.sh
atrakta doctor --parity --json
atrakta doctor --integration --json
atrakta projection status --json
```

## Evidence to Attach

- verification JSON outputs from `scripts/dev/verify_*.sh`
- `doctor --parity --json` and `doctor --integration --json` logs
- projection and extension manifest snapshots
- CI links for Linux verification and Windows native parity jobs

## Fail-closed Rule

If any gate item fails or is missing evidence, do not release.
