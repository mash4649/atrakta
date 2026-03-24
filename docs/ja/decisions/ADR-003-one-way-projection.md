# ADR-003 一方向投影

状態: 採用

投影パイプラインは canonical → overlay → include → render の順である。
投影の出力は正規状態へ自動書き戻しできない。
逆方向の同期は提案のみ。
