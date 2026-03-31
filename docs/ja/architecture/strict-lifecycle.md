# 厳格ライフサイクル

## スコープの種類

- request
- task
- workspace

## 状態

- normal
- guarded
- strict
- released

## トリガの例

- 古い状態
- 未解決の能力
- 非対応の投影サーフェス
- 承認の欠落
- ワークスペースの不一致
- ポリシーの曖昧さ
- 指示の衝突
- 監査保証の不足

## オペレーションルール

- guarded は inspect と限定的な propose を許可
- strict は、明示的に解除されない限り inspect と提案のみを許可
- released には明示的な解放条件が必要
- strict は解放経路を通じて可逆

## 遷移の接続

失敗ティアの出力は、明示的な遷移表を通じて厳格状態の遷移を駆動する。
