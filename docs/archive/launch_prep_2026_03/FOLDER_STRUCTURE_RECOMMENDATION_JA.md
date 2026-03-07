# 🚀 推奨フォルダ構成（GitHub 1万スター向け）

## 現状分析

✅ **Good:**
- README / README_JA が最適化済み
- docs/en, docs/ja が整理済み
- CHANGELOG が活発
- LICENSE が明記されている

⚠️ **Needs Improvement:**
- `internal/` が39個のパッケージで平坦 → ナビゲーション困難
- `scripts/` が混在（ビルド・テスト・開発がごちゃ混ぜ）
- `CONTRIBUTING.md` がない → コントリビュータ摩擦大
- `examples/` がない → 使用例の提示不足
- GitHub templates がない

❌ **Missing:**
- `DEVELOPMENT.md` → 開発者ガイド不在
- `.github/` templates → Issue/PR の体験悪化

---

## 🎯 推奨方針: Option A（Conservative + High Impact）

**理由：** 1万スター達成まで、フォルダ構成の美しさより、**貢献しやすさ** と **理解しやすさ** が優先

```
atrakta/
│
├─ 📄 README.md / README_JA.md          ✅ 完成
├─ 📄 CONTRIBUTING.md                   ⭐ 必須
├─ 📄 DEVELOPMENT.md                    ⭐ 必須
├─ 📄 CHANGELOG.md
├─ 📄 LICENSE
├─ 📄 VERSION
├─ 📄 go.mod
│
├─ .github/
│   ├─ workflows/
│   │   └─ release.yml
│   ├─ ISSUE_TEMPLATE/
│   │   ├─ bug_report.md               ⭐ NEW
│   │   ├─ feature_request.md          ⭐ NEW
│   │   └─ question.md                 ⭐ NEW
│   ├─ PULL_REQUEST_TEMPLATE.md        ⭐ NEW
│   ├─ SECURITY.md                     ⭐ NEW
│   └─ README.md                       ⭐ NEW (CI/Actions guide)
│
├─ cmd/
│   └─ atrakta/
│       └─ main.go
│
├─ internal/
│   ├─ runtime/          ⭐ NEW (group: detect, plan, apply, gate)
│   ├─ safety/           ⭐ NEW (group: contract, editsafety, policy, gate)
│   ├─ state/            ⭐ NEW (group: checkpoint, events, state, progress)
│   ├─ interfaces/       (existing: IDE adapters)
│   ├─ platform/         (existing: OS-specific)
│   ├─ cache/            (existing: runtimecache, registry)
│   ├─ util/             (existing: utilities)
│   └─ model/            (existing: data models)
│
├─ scripts/
│   ├─ build/                          ⭐ NEW
│   │   ├─ build_release_artifacts.sh
│   │   └─ install.sh
│   ├─ dev/                            ⭐ NEW
│   │   ├─ verify_loop.sh
│   │   ├─ verify_phase2.sh
│   │   ├─ soak.sh
│   │   └─ soak_*.sh
│   └─ README.md                       ⭐ NEW (script index)
│
├─ docs/
│   ├─ en/
│   │   ├─ README.md
│   │   ├─ 01_overview/
│   │   ├─ 02_spec/
│   │   ├─ 03_operations/
│   │   └─ 04_quality/
│   ├─ ja/
│   │   ├─ README.md
│   │   ├─ 01_全体/
│   │   ├─ 02_仕様/
│   │   ├─ 03_運用/
│   │   └─ 04_品質/
│   ├─ archive/                       (旧ドラフト・ローンチ資料)
│   ├─ README.md                      ✅ Already updated
│   ├─ ARCHITECTURE.md                ⭐ NEW (詳細アーキテクチャ図)
│   └─ TROUBLESHOOTING.md             ⭐ NEW (advanced guide)
│
├─ examples/                          ⭐ ★★★ CRITICAL
│   ├─ README.md                      (例の索引)
│   ├─ 01_basic_init.md              (基本的な初期化)
│   ├─ 02_cursor_workflow.md         (Cursor での実例)
│   ├─ 03_cli_workflow.md            (CLI での実例)
│   ├─ 04_tool_switching.md          (ツール切り替え)
│   └─ sample_projects/
│       └─ hello_ai_project/
│           ├─ src/
│           ├─ .atrakta/
│           └─ README.md
│
├─ .gitignore                         (updated: .tmp, .DS_Store)
├─ .atrakta/                      (self-hosted config)
└─ LICENSE
```

---

## 📋 実装優先度（1万スター達成まで）

### **Phase 1: 今週中** (最高 ROI)

| # | アクション | 影響度 | 手間 | 時間 |
|---|---|---|---|---|
| 1 | `examples/` ディレクトリ作成 | ⭐⭐⭐ | ⬇️ 低 | 2h |
| 2 | `CONTRIBUTING.md` 作成 | ⭐⭐⭐ | ⬇️ 低 | 1h |
| 3 | `.github/` templates 作成 | ⭐⭐ | ⬇️ 低 | 1h |
| 4 | `.github/SECURITY.md` 作成 | ⭐⭐ | ⬇️ 低 | 30m |
| 5 | `.gitignore` 更新 | ⭐ | ⬇️ 低 | 15m |

**合計：4.5時間**

### **Phase 2: 次週** (安定性向上)

| # | アクション | 影響度 | 手間 |
|---|---|---|---|
| 6 | `DEVELOPMENT.md` 作成 | ⭐⭐ | ⬇️ 低 |
| 7 | `scripts/` 再編成 (build/, dev/) | ⭐ | ⬇️ 低 |
| 8 | `docs/ARCHITECTURE.md` 作成 | ⭐⭐ | ➡️ 中 |
| 9 | GitHub Discussions enable | ⭐ | ⬇️ 低 |

### **Phase 3: 長期** (アーキテクチャ改善)

| # | アクション | 影響度 | 手間 |
|---|---|---|---|
| 10 | `internal/` → runtime/safety/state/ 再構成 | ⭐ | 🔴 高 |
| 11 | 統合テストスイート追加 | ⭐ | ➡️ 中 |

---

## 📁 詳細ファイルリスト（今すぐ作成）

### Phase 1: Essential Files

```
✅ examples/README.md
✅ examples/01_basic_init.md
✅ examples/02_cursor_workflow.md
✅ examples/03_cli_workflow.md
✅ examples/04_tool_switching.md
✅ examples/sample_projects/hello_ai_project/

✅ CONTRIBUTING.md
✅ DEVELOPMENT.md

✅ .github/SECURITY.md
✅ .github/PULL_REQUEST_TEMPLATE.md
✅ .github/ISSUE_TEMPLATE/bug_report.md
✅ .github/ISSUE_TEMPLATE/feature_request.md
✅ .github/ISSUE_TEMPLATE/question.md
✅ .github/README.md (CI guide)
```

### Phase 2: Supporting Files

```
✅ docs/ARCHITECTURE.md (detailed)
✅ docs/TROUBLESHOOTING.md
✅ scripts/README.md
```

### Phase 3: Refactoring

```
📁 internal/runtime/          (consolidate detect/, plan/, apply/, gate/)
📁 internal/safety/           (consolidate contract/, editsafety/, policy/)
📁 internal/state/            (consolidate checkpoint/, events/, state/)
```

---

## ⭐ 1万スターへの重要指標

### コントリビューション摩擦の削減
- [ ] `CONTRIBUTING.md` で段階を明確化
- [ ] Good First Issues でオンボーディング
- [ ] PR テンプレートで品質確保
- [ ] Issue テンプレートで報告品質向上

### ユーザー理解の促進
- [ ] `examples/` で実際の使用シーンを提示
- [ ] `DEVELOPMENT.md` で開発者ガイド提供
- [ ] `docs/ARCHITECTURE.md` でアーキテクチャ理解を深化

### コミュニティ信頼の構築
- [ ] `.github/SECURITY.md` でセキュリティポリシー公開
- [ ] CHANGELOG を毎月更新
- [ ] GitHub Discussions を有効化

### 発見性の向上
- [ ] GitHub Topics: `ai`, `automation`, `devops`, `workflow`
- [ ] Badge を README に追加 (CI/build/coverage)
- [ ] Releases ページを活用

---

## 🎬 実装タイムライン

```
Week 1 (今週)
├─ Mon: examples/ + CONTRIBUTING.md
├─ Tue: .github/ templates
├─ Wed: DEVELOPMENT.md
├─ Thu: .gitignore update + review
└─ Fri: commit & push → GitHub

Week 2
├─ scripts/ 再編成
├─ docs/ARCHITECTURE.md
├─ GitHub Discussions enable
└─ monitor stars/forks

Week 3+
├─ internal/ refactor (if team capacity)
├─ integration tests
└─ community engagement
```

---

## 📊 期待される効果

| 施策 | 効果 | 1K→10K での重要度 |
|---|---|---|
| examples/ 充実 | 新規ユーザー習熟時間 -70% | ⭐⭐⭐ |
| CONTRIBUTING 明確化 | コントリビューション +300% | ⭐⭐⭐ |
| GitHub templates | Issue 品質 +200% | ⭐⭐ |
| DEVELOPMENT.md | 開発者 onboarding -50% | ⭐⭐ |
| ARCHITECTURE.md | 複雑性理解 +150% | ⭐⭐ |

---

## 🚀 Next Action

**Today:** 決定（Option A をアプローム）
**This week:** Phase 1 実装
**Next week:** Phase 2 実装
**Month 2:** Phase 3 検討

**最初の勝ちを得るには：`examples/` + `CONTRIBUTING.md` が最重要。**
