# 07. Brownfield導入

[English](../../en/03_operations/07_brownfield_integration.md) | [日本語](./07_Brownfield導入.md)

## 目的

既存リポジトリに Atrakta を破壊的変更なしで導入します。

## 基本手順

```bash
atrakta init \
  --mode brownfield \
  --interfaces cursor \
  --merge-strategy append \
  --agents-mode append \
  --no-overwrite
```

## 原則

- user-managed 領域を上書きしない
- managed block / managed include file のみ変更する
- 自動マージ不能時は proposal patch を出す

## 導入後確認

```bash
atrakta doctor
```

`doctor --parity` / `doctor --integration` は parity バックログ上の追加項目です（現行安定CLIには未導入）。
