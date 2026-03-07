# ゼロベース分析：Atrakta リポジトリ名

## 前提条件

- 新規プロジェクト（まだ公開されていない想定）
- GitHub 1万スター を目指す
- Go + CLI ツール
- ユーザーは開発者・DevOps・AI ツール利用者

---

## 候補の比較

### 選択肢 1: `atrakta` (現状)

```
GitHub: github.com/afwm/atrakta
Go module: atrakta
CLI: atrakta init
Import: import "atrakta/internal/..."
```

**メリット**
- Go 標準に準拠 ✅
- CLI として短い ✅
- 検索しやすい ✅

**デメリット**
- ブランド名 "Atrakta" との距離あり
- 単語が一続きで読みにくい可能性
- CLI の自動補完で候補が多い可能性（"open" で）

---

### 選択肢 2: `open-harness` (ハイフン版)

```
GitHub: github.com/afwm/open-harness
Go module: github.com/afwm/open-harness (または go.mod で module atrakta)
CLI: open-harness init
Import: import "github.com/afwm/open-harness/internal/..."
```

**メリット**
- ブランド名と完全一致 ✅
- 読みやすい（言葉が分かれている）✅
- GitHub で目立ちやすい ✅

**デメリット**
- Go モジュール名がやや冗長
- CLI コマンドにハイフン（好みが分かれる）
- Go import が長くなる

---

### 選択肢 3: `atr` / `harness` (短縮版)

```
GitHub: github.com/afwm/harness
CLI: harness init
```

**メリット**
- CLI として超短い ✅
- 入力が少ない ✅

**デメリット**
- 汎用すぎて検索性が悪い ❌
- ブランド認識できない ❌
- 他のプロジェクトと競合（Jenkins Harness など）

**評価:** ❌ 除外

---

## ゼロベース視点での比較マトリックス

### 1. **ユーザー視点（最優先）**

| 項目 | atrakta | open-harness |
|---|---|---|
| **CLI コマンドの入力** | `atrakta` | `open-harness` |
| 入力の手軽さ | 🔴 長い（11文字） | 🟡 中（11文字だが読みやすい）|
| 記憶しやすさ | 🟡 中（一続き） | 🟢 高（言葉が分かれている）|
| README の理解 | 🟡 中 | 🟢 高（ブランドと一致）|
| Google 検索性 | 🟡 中（競合多い） | 🟢 高（ユニーク） |

### 2. **開発者視点（Go エコシステム）**

| 項目 | atrakta | open-harness |
|---|---|---|
| **Go import** | `import "atrakta/..."` | `import "github.com/afwm/open-harness/..."` |
| 入力の簡潔さ | 🟢 高 | 🟡 中（長い）|
| 慣例との一致 | 🟢 高 | 🟡 中（GitHub が含まれる）|
| パッケージ名の自然性 | 🟢 高 | 🟡 中 |

### 3. **マーケティング・発見性**

| 項目 | atrakta | open-harness |
|---|---|---|
| **ブランド認識** | 🟡 やや低 | 🟢 高 |
| Hacker News での目立ち方 | 🟡 中 | 🟢 高（ハイフンで言葉が明確）|
| Product Hunt での検索 | 🟡 中 | 🟢 高 |
| Twitter での引用 | 🟡 中 | 🟢 高（言葉が分かれているから切り出しやすい）|
| GitHub Trending 表示 | 🟢 同等 | 🟢 同等 |

### 4. **技術的フリクション**

| 項目 | atrakta | open-harness |
|---|---|---|
| **Go モジュール変更** | 0 コスト | 🟡 工夫が必要 |
| CLI 変更の自由度 | 🟢 高 | 🟢 高 |
| GitHub URL の安定性 | 🟢 安定 | 🟢 安定 |
| ドメイン取得 | 🟡 競合 | 🟢 取得しやすい |

---

## 業界慣例の調査

### ハイフンありの例（成功している大型プロジェクト）

```
Kubernetes      → k8s (ハイフンなし)
Docker          → docker (ハイフンなし)
Terraform       → terraform (ハイフンなし)
Ansible         → ansible (ハイフンなし)
Jenkins         → jenkins (ハイフンなし)
```

→ **ハイフンあり** の例は意外と少ない

### ハイフンありの例（中規模プロジェクト）

```
Open-Telemetry   → github.com/open-telemetry （複合ブランド名）
Open-Source      → 複合概念
Build-Essential  → apt パッケージ名
```

→ **複合概念・複数企業** で使われる傾向

---

## 重要な再考点

### 🔍 ブランド名の力

**現在：** "Atrakta"（ブランド名）vs "atrakta"（リポジトリ名）

**ユーザーの心理：**
```
"Atrakta を試したい" 
  ↓
Google / GitHub 検索
  ↓
リポジトリ名を入力
```

**シナリオ A: atrakta**
- "Atrakta" で検索 → 見つかる ✅
- でも "atrakta" という単語自体の記憶度は低い

**シナリオ B: open-harness**
- "Atrakta" で検索 → 見つかる ✅
- "open-harness" という単語の記憶度が高い ✅

→ **ハイフンありのが心理的距離が近い**

---

## 競争環境

### AI コーディング領域の競合ツール

```
Cursor          （ハイフンなし）
Copilot         （ハイフンなし）
Claude Code     （スペース、ハイフンなし）
Codeium         （ハイフンなし）
```

→ この領域では **ハイフンなし** が標準

### ただし...

```
GitHub Copilot      → 複合ブランド
VS Code             → 複合ブランド
```

→ スペース＝ハイフンと考えると、複合ブランドはある程度の容認がある

---

## ゼロベース時の決定基準

### 優先度の重み付け

1. **ユーザー発見性・記憶度** （重み: 40%）
   - atrakta: 🟡 60点
   - open-harness: 🟢 85点
   - **winner: open-harness** (+25点)

2. **技術的シンプルさ** （重み: 30%）
   - atrakta: 🟢 90点
   - open-harness: 🟡 70点
   - **winner: atrakta** (+20点)

3. **業界慣例** （重み: 20%）
   - atrakta: 🟢 85点（Go 標準）
   - open-harness: 🟡 60点（少数派）
   - **winner: atrakta** (+25点)

4. **ブランド一貫性** （重み: 10%）
   - atrakta: 🟡 60点
   - open-harness: 🟢 95点
   - **winner: open-harness** (+35点)

### スコア計算

```
atrakta:
  40% × 60 + 30% × 90 + 20% × 85 + 10% × 60
  = 24 + 27 + 17 + 6 = 74点

open-harness:
  40% × 85 + 30% × 70 + 20% × 60 + 10% × 95
  = 34 + 21 + 12 + 9.5 = 76.5点
```

**結果：open-harness がやや有利（76.5 vs 74）**

---

## ゼロベース判断

### 🎯 最適な選択肢：**`open-harness`**

#### 理由

1. **ユーザー体験**
   - ブランド "Atrakta" との心理的距離が近い
   - "open-harness" は "open" + "harness" と分解しやすい
   - Hacker News / Product Hunt で目立ちやすい
   - ドキュメント・マーケティング資料で文脈が明確

2. **記憶度と検索性**
   - "Atrakta" と "open-harness" が自然に対応
   - Google 検索で競合が少ない
   - GitHub での検索でユニーク

3. **長期的なブランド構築**
   - 企業名・プロダクト名に昇格する可能性がある
   - 「AI コーディングのランタイム」という概念が定着する場合、"Atrakta" は商標・ブランドになりやすい

#### デメリットの対処法

1. **Go モジュール名の冗長性**
   ```go
   // go.mod
   module github.com/afwm/open-harness
   
   // internal は同じ
   import "github.com/afwm/open-harness/internal/detect"
   ```
   → 業界標準（他のプロジェクトも同じ）

2. **CLI コマンドにハイフン**
   ```bash
   open-harness init --interfaces cursor
   ```
   → 実際には `atr` エイリアスを用意することもできる
   ```bash
   alias atr='open-harness'
   atr init --interfaces cursor
   ```

3. **入力が長い**
   → タブ補完で解決（shell が補完）

---

## 最終推奨

### **結論：`open-harness` に変更する（ゼロベースでは最善）**

**ただし、実装する場合の注意：**

1. **今が最後のチャンス**
   - 公開前なら無料
   - 公開後は後戻り困難

2. **変更スコープ（全部で 1-2 時間）**
   ```
   - GitHub リポジトリ名: atrakta → open-harness
   - go.mod: module atrakta → module github.com/afwm/open-harness
   - scripts/install.sh: URL 更新
   - docs: URL 参照更新
   - .github/workflows: URL 参照更新
   ```

3. **公開後は以下を検討**
   - `atr` コマンドエイリアスの提供
   - Go モジュールの `v1` リリース時に確定

---

## 代替案：折衷策

現在の開発進捗が進んでいる場合：

```
GitHub: github.com/afwm/atrakta (変更なし)
CLI: 同上
Go module: github.com/afwm/atrakta (変更なし)

ただし、ドキュメント・マーケティングで常に
"Atrakta (open-harness)" と併記する
```

**メリット：**
- 技術的フリクションゼロ
- ブランド認識はできる

**デメリット：**
- CLI 名が "atrakta" のままで、ブランド表記との整合性が低い

---

## 最終判断表

| 選択 | スコア | 推奨度 | 条件 |
|---|---|---|---|
| **open-harness** | 76.5 | ⭐⭐⭐⭐⭐ | 公開前なら今すぐ |
| **atrakta** | 74 | ⭐⭐⭐⭐ | 開発が進んでいて変更コストが高い場合 |
| **atrakta + 併記** | 72 | ⭐⭐⭐ | 折衷策 |

**最強の推奨：`open-harness` に変更する** 🎯
