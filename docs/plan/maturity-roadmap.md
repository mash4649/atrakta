# Atrakta Maturity and Adoption Roadmap

Version: `1.0.0-alpha.2` baseline  
Created: 2026-03-25

Japanese / 日本語: [../ja/plan/maturity-roadmap.md](../ja/plan/maturity-roadmap.md)

---

## Table of Contents

1. [Current State Assessment](#1-current-state-assessment)
2. [Maturity Challenge Map](#2-maturity-challenge-map)
3. [Phased Roadmap](#3-phased-roadmap)
4. [Technical Challenges and Decisions](#4-technical-challenges-and-decisions)
5. [Ecosystem and Adoption Challenges](#5-ecosystem-and-adoption-challenges)
6. [Risks and Mitigations](#6-risks-and-mitigations)
7. [Success Metrics (KPIs)](#7-success-metrics-kpis)

---

## 1. Current State Assessment

### 1.1 Completed (v0 Baseline)

| Area | Status | Detail |
|------|--------|--------|
| Contract foundation (10 Issues) | **Done** | Layer boundary, guidance precedence, projection model, failure routing, managed scope, legacy governance, operations capability, extension boundary, audit integrity, inspect/preview/simulate |
| Resolvers | **Done** | 12 resolvers implemented and tested |
| `atrakta run` base contract | **Done** | Single primitive, exit code contract (0/1/2/3), onboarding and normal paths |
| Zero-config onboarding | **Done** | detect → propose → accept, proposal bundle generation, risk detection |
| Schema-driven validation | **Done** | bundle-input/output, fixtures-report, onboarding-proposal |
| CI / snapshot gate | **Done** | GitHub Actions, deterministic snapshot comparison, coverage verification |
| Specification docs | **Done** | Bilingual EN/JA, 3 ADRs |
| Surface portability | **In progress** | Semantic classification for `agents_md`, `ide_rules`, `repo_docs`, `skill_bundle` |

### 1.2 Outstanding Gaps

| Gap | Priority | Summary |
|-----|----------|---------|
| init lifecycle unification | **Partial** | Minimal `init` exists; wrap/hook/ide-autostart integration still missing |
| wrap / hook / ide-autostart | **Required** | No tool binding layer |
| projection render / status / repair | **Done** | Initial disk-write render/status/repair exist; target expansion remains |
| state / progress / task-graph hardening | **Partial** | Base persistence exists; schema/versioning and replay semantics still need hardening |
| legacy events.jsonl | **Low** | `run-events` is the v0 canonical stream; legacy `/.atrakta/events.jsonl` is read-only historical data |
| Fast Path hardening | **Important** | Fast path exists for `start`/`resume`; key policy and observability need refinement |
| Interface resolution extensibility | **Important** | Core resolution exists; wrapper/hook-trigger integration and binding-driven extension remain |
| gc / migrate | **Partial** | Initial commands exist; policy and migration guidance still need hardening |
| Binary distribution / installer | **Important** | Requires `go run`; no distribution form |
| import / capability / recipe | **Nice to have** | No external asset import pipeline |

---

## 2. Maturity Challenge Map

### 2.1 Architecture Maturity

```
┌──────────────────────────────────────────────────────────────────┐
│                  Atrakta Maturity Challenge Map                   │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [Done] Contract base ──→ [Gap] Runtime integration              │
│    │                        ├── init lifecycle unification       │
│    │                        ├── Interface resolution extensibility│
│    │                        ├── Fast Path hardening              │
│    │                        └── state / progress / events        │
│    │                                                             │
│  [Done] Resolvers ──→ [Gap] Tool binding                         │
│    │                    ├── wrap install / run                    │
│    │                    ├── hook install / repair                 │
│    │                    └── ide-autostart                         │
│    │                                                             │
│  [WIP] Portability ──→ [Gap] Projection materialization          │
│    │                     ├── projection render                   │
│    │                     ├── projection status / repair           │
│    │                     └── drift detection and auto-repair      │
│    │                                                             │
│  [Done] Audit base ──→ [Gap] Operational tools                   │
│                          ├── gc (tmp / events)                   │
│                          ├── migrate check                       │
│                          └── doctor expansion                    │
│                                                                  │
│  [Gap] Distribution                                              │
│    ├── Binary build (goreleaser etc.)                            │
│    ├── Install scripts (curl / brew)                             │
│    ├── Docker image                                              │
│    └── Package manager publishing                                │
│                                                                  │
│  [Gap] Ecosystem expansion                                       │
│    ├── Plugin SDK                                                │
│    ├── MCP integration                                           │
│    ├── Third-party IDE adapters                                  │
│    └── Community templates                                       │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 2.2 Quality and Reliability

| Challenge | Current | Target |
|-----------|---------|--------|
| Test coverage | Resolver unit + contract + fixture tests | Add E2E, integration, fuzz tests |
| Error handling | Basic exit code contract | Structured errors, recovery guidance, diagnostic logs |
| Performance | Unmeasured | Benchmarks, large-repo validation |
| Session handoff fidelity | `auto-state` and audit exist, but restart context is implicit | Structured handoff bundle (`feature-spec`, acceptance criteria, checkpoint, next action) that survives `resume` and clean restarts |
| Verification depth | Contract validation + inspect/apply flow | Dedicated evaluator loop with browser/app checks and artifact-based scoring |
| Harness drift vs model evolution | Harness assumptions reviewed manually | Profile and ablation benchmarks per model generation; retire non-load-bearing scaffolding |
| Security | Risk detection (external send / destructive ops) | Security audit, formal permission model verification |
| External deps | stdlib-only (no go.sum) | Dependency policy documented |

---

## 3. Phased Roadmap

### Phase 1: Runtime Integration (alpha → beta)

**Goal**: `atrakta run` works as a minimal harness

| Task | Depends on | Size | Priority |
|------|------------|------|----------|
| `start` lifecycle hardening (edge-case diagnostics, metadata consistency) | run contract | M | P0 |
| Interface resolution extensibility (binding-driven detection, wrapper/hook triggers) | start | M | P0 |
| state.json / progress.json schema hardening and migration policy | start | M | P0 |
| run-events taxonomy expansion and emission coverage | start | M | P0 |
| Structured handoff bundle deepening (`feature-spec`, acceptance, checkpoint, next action) | start, state | M | P0 |
| `resume` fidelity hardening from handoff/auto-state artifacts | start, state | M | P0 |
| Fast Path expansion (key policy + observability + compatibility) | start, state | M | P1 |
| `init` command (wrap + hook + ide-autostart + start) | wrap, hook, start | L | P1 |
| task-graph semantics hardening (resume replay and node lifecycle) | start | S | P1 |

### Phase 2: Tool Binding and Projection (early beta)

**Goal**: Real connection to AI tools, projection materialization

| Task | Depends on | Size | Priority |
|------|------------|------|----------|
| `wrap install / uninstall / run` | start | L | P0 |
| `hook install / uninstall / status / repair` | start | L | P0 |
| `ide-autostart` (.vscode/tasks.json etc.) | hook | M | P1 |
| `projection render` (canonical + contract → agents_md file generation) | start, surface-portability | L | P0 |
| `projection status` (drift detection between projection and canonical) | projection render | M | P1 |
| `projection repair` (auto-fix for drift) | projection status | M | P1 |
| Cursor adapter binding | wrap, hook | M | P0 |
| VS Code adapter binding | wrap, hook | M | P1 |
| Browser-backed evaluator runner (Playwright / CLI app verification) | start, wrap | L | P1 |

### Phase 3: Operational Maturity (beta → RC)

**Goal**: Production-quality tooling

| Task | Depends on | Size | Priority |
|------|------------|------|----------|
| `gc` command (tmp / events scopes) | events.jsonl | M | P1 |
| `migrate check` (schema version verification and migration guidance) | state, events | M | P1 |
| doctor expansion (state integrity, projection parity, event chain) | start, projection | M | P1 |
| Structured error messages with recovery guidance | all | L | P1 |
| Acceptance artifact generation (prompt → executable spec + scoring rubric) | start, state | M | P1 |
| Harness profile / ablation benchmarks by model generation | start, performance benchmarks | M | P1 |
| Selective orchestration policy (when planner/evaluator/checkpoints are actually required) | acceptance artifacts, profile benchmarks | M | P1 |
| Performance benchmarks | start | M | P2 |
| Large repository support (1000+ files, monorepo) | start, detect | L | P2 |
| Security audit and formal permission model verification | all | L | P2 |
| Fuzz testing | resolvers | M | P2 |

### Phase 4: Distribution and Ecosystem (RC → GA)

**Goal**: Installable and usable by anyone

| Task | Depends on | Size | Priority |
|------|------------|------|----------|
| goreleaser config (multi-platform binaries) | all | M | P0 |
| Install scripts (curl / brew / scoop) | binary build | M | P0 |
| Docker image (CI/CD embedding) | binary build | S | P1 |
| GitHub Releases automation | goreleaser | S | P0 |
| Plugin SDK (for extension developers) | extension-boundary | L | P2 |
| MCP server integration (IDE-native atrakta access) | all | L | P2 |
| Community templates (language/framework initial configs) | onboarding | M | P2 |
| Package manager publishing (npm / pip / cargo) | binary | M | P2 |

### Phase 5: Documentation, DX, and Adoption (parallel with all phases)

**Goal**: Self-service documentation and community

| Task | Depends on | Size | Priority |
|------|------------|------|----------|
| Tutorial ("Get started in 5 minutes") | Phase 1 | M | P0 |
| Use-case guides (brownfield adoption, CI integration, team workflows) | Phase 2 | L | P1 |
| Integrated API reference (auto-generated) | all resolvers | M | P1 |
| Official glossary (v1 vocabulary unification) | vocabulary-alignment | S | P1 |
| CONTRIBUTING.md / dev environment setup | all | S | P0 |
| Error catalog | Phase 3 | M | P1 |
| Official website / landing page | Phase 4 | L | P2 |
| Logo and brand identity | -- | S | P2 |
| Blog / technical articles (design philosophy, comparison pieces) | Phase 4 | M | P2 |

### Harness Learnings Feed (Anthropic, March 24, 2026)

Reference: [Anthropic, "Harness design for long-running application development"](https://www.anthropic.com/engineering/harness-design-long-running-apps)

- Keep the harness minimal, but not simplistic: re-benchmark every major model generation and preserve only load-bearing components.
- Treat `resume` as an artifact handoff problem, not only a context-window problem. The restart path needs explicit spec, checkpoint, and next-step artifacts.
- Separate generation from evaluation. For app-quality tasks, the evaluator should use real tools such as a browser, not only schema validation.
- Convert subjective quality into executable acceptance criteria and scoring rubrics so the evaluator can gate reliably.
- Make orchestration conditional. Planner/evaluator/reset loops should be enabled when they improve outcomes, not unconditionally on every task.

---

## 4. Technical Challenges and Decisions

### 4.1 Architecture Decisions Required

| Decision | Options | Criteria | Recommendation |
|----------|---------|----------|----------------|
| legacy events.jsonl handling | (A) read-only historical data / (B) import-on-demand | Preserve v0 ownership and minimize coupling | **(A)**: keep legacy `/.atrakta/events.jsonl` read-only; no conversion layer |
| State schema versioning | (A) 0.14.1 compatible subset / (B) v0 own + version field | Existing user migration path | **(B)**: Version with `schema_version` field, use migrate check for conversion |
| Interface resolution extensibility | (A) Hardcoded / (B) Plugin-based / (C) Binding definition files | Ease of adding new IDEs | **(C)**: Extend existing `adapters/bindings/*/binding.json` |
| Projection write strategy | (A) Full overwrite / (B) Merge / (C) proposal-only | User customization preservation | **(C) → (B)**: Start proposal-only, merge after approval |
| External dependency policy | (A) Stdlib-only / (B) Curated libraries allowed | Build speed, binary size, security | **(A)** as default; (B) only with justification |
| CI/CD platform | (A) GitHub Actions only / (B) Multi-CI support | Target user base | **(A)** first; (B) on demand |

### 4.2 Technical Debt and Refactoring Candidates

| Target | Issue | Proposed fix |
|--------|-------|--------------|
| `cmd/atrakta/cmd_run.go` (~1200 lines) | Session lifecycle and apply orchestration logic is concentrated | Split shared start/run/resume lifecycle into smaller command-local files/packages |
| Resolver common output structure | Consolidation in progress at `resolvers/common/output.go` | Enforce unified output interface across all resolvers |
| Fixture management | Manual management under `fixtures/` | Consider table-driven tests + fixture auto-generation |
| Terminology drift | Acknowledged in `follow-up-vocabulary-alignment.md` | Resolve in Phase 5 glossary work |

---

## 5. Ecosystem and Adoption Challenges

### 5.1 User Adoption Barriers

| Barrier | Severity | Solution |
|---------|----------|----------|
| Go build environment required | **High** | Binary distribution (Phase 4) |
| Conceptual learning curve (resolvers, projections, layers) | **High** | Tutorials + progressive disclosure (Phase 5) |
| Anxiety about adopting into existing projects | **Medium** | Zero-config + proposal-only defaults (done, enhance docs) |
| Difficulty seeing value | **Medium** | Before/after demos, metrics dashboard |
| Competition / coexistence with other tools | **Medium** | Comparison guides, coexistence pattern documentation |

### 5.2 Surface Support Priority

| Surface | Priority | Reason |
|---------|----------|--------|
| Cursor | **P0** | Primary development target, MCP integration possible |
| VS Code + Copilot | **P1** | Largest user base |
| CLI (Codex etc.) | **P1** | Essential for CI/CD integration |
| JetBrains IDEs | **P2** | Enterprise demand |
| Neovim / Emacs | **P3** | Community-driven support |

---

## 6. Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Runtime integration complexity delays | High | Medium | Keep Phase 1 scope minimal, expand incrementally |
| AI tool API changes breaking compatibility | High | High | Binding definition abstraction, loose coupling in adapter layer |
| Conceptual complexity driving user attrition | High | Medium | Enforce progressive disclosure, safe defaults |
| Harness overfitting to older model behavior | High | Medium | Run profile / ablation benchmarks per model generation and retire non-load-bearing steps |
| Evaluation blind spots for UX or integration quality | High | Medium | Add acceptance artifacts, browser-backed evaluators, and score thresholds |
| Security incident (unauthorized apply etc.) | High | Low | proposal-only default, explicit approval enforcement, audit log |
| Schema changes breaking backward compatibility | Medium | Medium | Stabilize contracts during alpha, strict versioning policy |
| Bus factor from sole developer | High | -- | Documentation quality, ADR records, community building |

---

## 7. Success Metrics (KPIs)

### 7.1 Technical Maturity

| Metric | Alpha (current) | Beta target | GA target |
|--------|-----------------|-------------|-----------|
| Resolver coverage | 12/12 (100%) | 12/12 + new resolvers | Stable |
| Test coverage (lines) | Unmeasured | 70%+ | 85%+ |
| CI pass criteria | Snapshot gate only | E2E + integration | Including fuzz |
| Supported IDEs | 0 (binding defs only) | 2 (Cursor + CLI) | 4+ |
| Average `run` latency | Unmeasured | < 2s | < 500ms (Fast Path) |
| Resume success rate | Unmeasured | 80%+ | 95%+ |
| Evaluator catch rate on harness fixtures | Unmeasured | Baseline established | 90%+ |
| Harness overhead vs direct execution | Unmeasured | < 30% median | < 10% median |

### 7.2 Adoption

| Metric | Beta target | GA target | 1-year target |
|--------|-------------|-----------|---------------|
| GitHub Stars | -- | 100+ | 1000+ |
| Monthly installs | -- | 50+ | 500+ |
| Contributors | 1 | 3+ | 10+ |
| Plugins / templates | 0 | 3+ | 20+ |
| Documentation pages | 53 | 80+ | 120+ |

---

## Overall Timeline (Estimate)

```
2026 Q2          Q3              Q4              2027 Q1         Q2
  │               │               │               │              │
  ├── Phase 1 ────┤               │               │              │
  │  Runtime      ├── Phase 2 ────┤               │              │
  │  integration  │  Tool binding  ├── Phase 3 ────┤              │
  │               │  Projection    │  Operational   ├── Phase 4 ──┤
  │               │               │  maturity      │  Distribution│
  │               │               │               │  Ecosystem   │
  │                                                              │
  ├── Phase 5 (Documentation / DX) ── parallel with all phases ──┤
  │                                                              │
  ▼ alpha.2       ▼ beta.1        ▼ beta.2        ▼ RC.1        ▼ GA
```

---

## Immediate Next Actions

1. **Phase 1 hardening design**: Define remaining lifecycle hardening scope (`init`, handoff fidelity, fast-path compatibility policy)
2. **run-events schema hardening**: Expand runtime event taxonomy and keep the write path v0-only
3. **run lifecycle split**: Break large `cmd_run` lifecycle logic into smaller units
4. **CONTRIBUTING.md creation**: Dev environment setup and contribution rules
5. **License selection**: Decide license for public release
