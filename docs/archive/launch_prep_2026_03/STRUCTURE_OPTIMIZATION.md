# Repository Structure Optimization for 10K Stars

## Current Structure Analysis

```
atrakta/
├── README.md / README_JA.md          ✅ Excellent
├── AGENTS.md
├── CHANGELOG.md                       ✅ Good
├── LICENSE                            ✅ Good
├── VERSION                            ✅ Good
├── go.mod                             ⚠️ Could add go.sum
│
├── .github/workflows/                 ✅ Good for automation
├── .gitignore                         ✅ Essential
│
├── cmd/atrakta/                   ✅ Clear CLI entry
├── internal/                          ⚠️ 39 modules (see below)
├── scripts/                           ⚠️ Needs organization
│
├── docs/
│   ├── en/                           ✅ Excellent
│   ├── ja/                           ✅ Excellent
│   ├── archive/                      ✅ Good hygiene
│   └── README.md                     ✅ Good nav
│
├── .atrakta/                      ✅ Good self-eating-dog-food
└── .tmp/                              ⚠️ Should be in .gitignore
```

## Issues & Concerns

### 1. **`internal/` overcomplexity** (39 modules)
Currently flat structure makes it hard to understand subsystems:
```
internal/
  ├── adapter, apply, bootstrap, checkpoint, ...  ← 39 packages, hard to navigate
```

### 2. **`scripts/` organization**
```
scripts/
  ├── build_release_artifacts.sh
  ├── install.sh
  ├── soak_*.sh                        ← testing scripts mixed with build
  └── verify_*.sh                      ← verification scripts
```
Unclear which are user-facing vs. CI/developer-only.

### 3. **Missing contributor guidance**
No `CONTRIBUTING.md` or `DEVELOPMENT.md` → friction for potential contributors.

### 4. **No clear examples or quickstart code**
Users see concept but struggle with actual usage patterns.

### 5. **LICENSE + CONTRIBUTING visibility**
Both are critical for first 1K → 10K journey, should be more prominent.

### 6. **No separate build / CI docs**
`.github/workflows/` exists but no `BUILD.md` or `.github/README.md`.

---

## Recommended Optimizations

### ✅ **Option A: Conservative (Low Risk)**

Minimal structural changes, mostly documentation + file reorganization:

```
atrakta/
│
├── README.md / README_JA.md           (already optimized)
├── CONTRIBUTING.md                    ⭐ NEW
├── DEVELOPMENT.md                     ⭐ NEW
├── CHANGELOG.md
├── LICENSE
├── VERSION
├── go.mod / go.sum
│
├── .github/
│   ├── workflows/
│   ├── ISSUE_TEMPLATE/                ⭐ NEW (GitHub native)
│   ├── PULL_REQUEST_TEMPLATE.md       ⭐ NEW (GitHub native)
│   └── README.md                      ⭐ NEW (CI/Actions guide)
│
├── cmd/atrakta/
│   └── main.go                        (no change)
│
├── internal/
│   ├── adapter/                       (group related modules here)
│   ├── core/
│   ├── runtime/                       ⭐ Consider grouping: detect, plan, apply, gate
│   ├── safety/                        ⭐ New: editsafety, contract, policy, gate
│   ├── state/                         ⭐ New: checkpoint, events, state, progress
│   ├── platform/                      (existing)
│   ├── util/                          (existing)
│   └── pkg/                           ⭐ Consider adding for test utilities
│
├── scripts/
│   ├── build/                         ⭐ NEW: release build scripts
│   │   └── build_release_artifacts.sh
│   │   └── install.sh
│   ├── dev/                           ⭐ NEW: developer scripts
│   │   ├── verify_loop.sh
│   │   ├── verify_*.sh
│   │   └── soak.sh
│   └── README.md                      ⭐ NEW: script guide
│
├── docs/
│   ├── en/
│   ├── ja/
│   ├── archive/
│   ├── README.md
│   └── ARCHITECTURE.md                ⭐ NEW: detailed arch
│
├── examples/                          ⭐ NEW (critical!)
│   ├── README.md                      Quick example index
│   ├── 01_basic_init.md              How to init
│   ├── 02_cursor_workflow.md         Real Cursor example
│   ├── 03_cli_workflow.md            Real CLI example
│   └── sample_projects/              Small sample repos
│       └── hello_ai_project/
│
└── .gitignore                         (update to exclude .tmp, .DS_Store)
```

**Pros:**
- Low risk: no code changes
- Easy to implement incrementally
- Addresses discoverability + contribution friction
- Follows GitHub conventions

**Cons:**
- Doesn't solve `internal/` complexity
- Less dramatic visual impact

### ⭐ **Option B: Ambitious (Medium Risk)**

Reorganize `internal/` to reflect runtime architecture:

```
atrakta/
├── (same as Option A for docs/examples/scripts/...)
│
├── internal/
│   ├── runtime/                       ⭐ Core pipeline
│   │   ├── detector.go                (formerly detect/)
│   │   ├── planner.go                 (formerly plan/)
│   │   ├── applier.go                 (formerly apply/)
│   │   └── gate.go                    (formerly gate/)
│   │
│   ├── safety/                        ⭐ Safety layer
│   │   ├── contract.go
│   │   ├── editsafety.go
│   │   └── policy.go
│   │
│   ├── state/                         ⭐ State management
│   │   ├── checkpoint.go
│   │   ├── events.go
│   │   ├── state.go
│   │   └── progress.go
│   │
│   ├── interfaces/                    ⭐ Interface adapters
│   │   ├── cursor.go
│   │   ├── claude.go
│   │   ├── codex.go
│   │   └── vscode.go
│   │
│   ├── platform/                      ⭐ Platform-specific
│   │   ├── darwin.go
│   │   ├── linux.go
│   │   └── windows.go
│   │
│   ├── cache/                         ⭐ Caching layer
│   │   ├── runtimecache.go
│   │   └── registry.go
│   │
│   ├── util/                          ⭐ Utilities
│   │   └── ...
│   │
│   └── model/                         ⭐ Data models
│       └── ...
```

**Pros:**
- Much clearer architecture
- Easier onboarding for new contributors
- Reflects "Detect → Plan → Apply → Gate" conceptually
- Improves IDE navigation + search

**Cons:**
- Breaking change: requires refactoring imports
- CI/CD must pass tests
- More effort to implement

---

## Priority Implementation for 10K Stars

### **Phase 1: Immediate (Week 1)** — High Impact, Low Risk

1. Create `CONTRIBUTING.md`
   - Link to good first issues
   - Development environment setup
   - Code style guide
   - PR review process

2. Create `DEVELOPMENT.md`
   - Architecture overview
   - Internal package guide
   - Running tests locally
   - Debugging tips

3. Add `.github/` templates
   - Issue templates (bug/feature/question)
   - PR template
   - Discussion guidelines

4. Create `examples/` directory
   - `01_basic_init.md`
   - `02_cursor_workflow.md`
   - `03_cli_switching.md`

5. Update `.gitignore` to remove `.tmp/`, `.DS_Store`

### **Phase 2: Medium (Week 2-3)** — Consolidation

1. Reorganize `scripts/` into `build/` and `dev/`
2. Add `docs/ARCHITECTURE.md` (detailed diagrams)
3. Add `docs/TROUBLESHOOTING.md` (advanced)
4. Create GitHub Discussions starter topics

### **Phase 3: Long-term (Month 2+)** — Refactor

1. Refactor `internal/` to Option B structure (if team capacity)
2. Add integration tests directory
3. Add benchmarking suite
4. Create video tutorial references

---

## Quick Wins for Star Growth

| Action | Impact | Effort | Timeline |
|---|---|---|---|
| Add `examples/` with working code | ⭐⭐⭐ | Low | Day 1 |
| Create `CONTRIBUTING.md` | ⭐⭐⭐ | Low | Day 1 |
| Add issue templates in `.github/` | ⭐⭐ | Low | Day 1 |
| Create `.github/SECURITY.md` | ⭐⭐ | Low | Day 1 |
| Refactor `scripts/` | ⭐⭐ | Low | Day 2 |
| Add `DEVELOPMENT.md` | ⭐⭐ | Low | Day 2 |
| Reorganize `internal/` | ⭐ | High | Week 2+ |

---

## Star-Winning Checklist (First 1K → 10K)

```
Repository Health
☐ Well-commented code (esp. interface boundaries)
☐ Comprehensive README (✓ already done)
☐ Multiple language docs (✓ already done)
☐ Active CHANGELOG (✓ already done)

Contribution Friction
☐ CONTRIBUTING.md with clear process
☐ Good first issues labeled
☐ Quick response to issues
☐ Code review rubric

Getting Started
☐ 3-minute quickstart (✓ in README)
☐ Working examples (⚠️ Missing)
☐ Video walkthrough (⚠️ In archive, needs polish)
☐ Troubleshooting guide

Discovery
☐ GitHub topic tags (ai, automation, devops, etc.)
☐ Badges for builds, coverage, license
☐ GitHub Releases with good notes
☐ Monthly updates visible

Community
☐ Discussions enabled
☐ Newsletter or announcement channel
☐ Twitter/X posts about updates
```

---

## Recommended Next Steps

### **Immediate Action (This Week)**

1. **Add `examples/` directory** with 3 working examples
2. **Create `CONTRIBUTING.md`** (use template from https://github.com/electron/electron/blob/main/CONTRIBUTING.md)
3. **Add GitHub templates** (`.github/ISSUE_TEMPLATE/` and `.github/PULL_REQUEST_TEMPLATE.md`)
4. **Create `.github/SECURITY.md`** (addresses trust)

### **Follow-up (Next Week)**

5. Reorganize `scripts/` into `scripts/build/` and `scripts/dev/`
6. Add `DEVELOPMENT.md` with detailed architecture guide
7. Create `docs/ARCHITECTURE.md` with Mermaid diagrams

### **Longer-term**

8. Refactor `internal/` to reflect architecture (Option B) — only if team capacity
9. Set up GitHub Discussions for support
10. Create Hacker News / Product Hunt launch strategy

---

## File Examples to Add

### CONTRIBUTING.md
```markdown
# Contributing to Atrakta

We'd love your contributions!

## Getting Started

1. Fork the repo
2. Clone: git clone https://github.com/YOUR_USERNAME/Atrakta.git
3. Create branch: git checkout -b feature/your-feature
4. Follow code style (see DEVELOPMENT.md)
5. Test: go test ./...
6. Push and open PR

## Good First Issues

See [issues labeled "good first issue"](...)

## Code Review Process

...
```

### DEVELOPMENT.md
```markdown
# Development Guide

## Architecture

Atrakta follows a 4-stage pipeline:

1. **Detect** — discover current project state
2. **Plan** — create task DAG based on state
3. **Apply** — execute tasks in topological order
4. **Gate** — validate safety & quality

See docs/ARCHITECTURE.md for details.

## Internal Package Structure

- `internal/runtime/` — core pipeline
- `internal/safety/` — contracts & policy
- `internal/state/` — event log & checkpoint
- `internal/interfaces/` — editor adapters

## Running Tests

```bash
go test ./...
go test -v ./internal/runtime/...
```

## Local Development

```bash
go run ./cmd/atrakta init --interfaces cursor
```
```

### examples/01_basic_init.md
```markdown
# Example 1: Basic Initialization

Initialize Atrakta in a new project.

## Step 1: Install
```bash
curl ... | bash
```

## Step 2: Init
```bash
atrakta init --interfaces cursor
```

## What Was Created
- AGENTS.md
- .atrakta/contract.json
- ...

## Next: Start a session
```bash
atrakta start --interfaces cursor
```
```

---

## Summary

| Element | Current | Phase 1 | Phase 2 | Phase 3 |
|---|---|---|---|---|
| **README** | ✅ | ✅ | ✅ | ✅ |
| **Docs (en/ja)** | ✅ | ✅ | ✅ | ✅ |
| **Examples** | ❌ | ✅ | ✅ | ✅ |
| **CONTRIBUTING** | ❌ | ✅ | ✅ | ✅ |
| **DEVELOPMENT** | ❌ | ✅ | ✅ | ✅ |
| **GitHub Templates** | ❌ | ✅ | ✅ | ✅ |
| **scripts/ organized** | ⚠️ | ⚠️ | ✅ | ✅ |
| **internal/ refactored** | ❌ | ❌ | ❌ | ⭐ |

**Next action:** Start Phase 1 this week. Examples + CONTRIBUTING are highest ROI for star growth.
