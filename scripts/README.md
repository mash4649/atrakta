# Scripts

Atrakta のビルド・開発用スクリプト集です。

## ディレクトリ構成

```
scripts/
├── build/    ← リリースビルド・インストール（ユーザー向け）
└── dev/      ← 開発・検証・テスト（開発者向け）
```

---

## build/ — リリース・インストール

| スクリプト | 用途 |
|---|---|
| `build_release_artifacts.sh` | 全プラットフォーム向けリリースバイナリを生成 |
| `install.sh` | ユーザー向けインストールスクリプト（curl で配布） |

### 使い方

```bash
# リリースバイナリを手動生成（CI 自動化のフォールバック）
./scripts/build/build_release_artifacts.sh

# インストールスクリプトをローカルでテスト
./scripts/build/install.sh
```

---

## dev/ — 開発・検証

| スクリプト | 用途 |
|---|---|
| `verify_loop.sh` | 基本テストをループ実行（CI でも使用） |
| `verify_perf_gate.sh` | SLO パフォーマンス回帰テスト（CI でも使用） |
| `verify_parity.sh` | parity ドリフト関連の検証（JSON 出力あり） |
| `verify_extensions.sh` | extension projection 関連の検証（JSON 出力あり） |
| `verify_brownfield.sh` | brownfield 導入系の検証（JSON 出力あり） |
| `verify_projection_repair.sh` | projection repair 関連の検証（JSON 出力あり） |
| `verify_phase2.sh` | Phase 2 検証スイート |
| `verify_provisional.sh` | 暫定検証スクリプト |
| `soak.sh` | 短時間ソークテスト |
| `soak_24h.sh` | 24 時間ソークテスト |
| `soak_72h.sh` | 72 時間ソークテスト |

### 使い方

```bash
# 基本テスト（PR 前に実行推奨）
./scripts/dev/verify_loop.sh

# パフォーマンス回帰テスト
./scripts/dev/verify_perf_gate.sh

# parity / extension / brownfield / repair 検証
./scripts/dev/verify_parity.sh
./scripts/dev/verify_extensions.sh
./scripts/dev/verify_brownfield.sh
./scripts/dev/verify_projection_repair.sh

# ソークテスト（長時間安定性確認）
./scripts/dev/soak.sh
./scripts/dev/soak_24h.sh
```

---

## 一般的な開発者向け

コードの変更後は以下で十分です:

```bash
go test ./...
```

`scripts/dev/` のスクリプトは主にメンテナや CI で使用します。
