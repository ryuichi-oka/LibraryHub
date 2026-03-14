# 監査ログ共通出力方針（T-004）

## 1. 目的
- 監査対象操作を一貫した形式で記録し、追跡・調査・運用監視を可能にする
- 実装者ごとの差異を減らし、検索しやすいログを維持する

## 2. 対象
- 管理操作: 蔵書作成/更新/削除、ユーザー有効/無効切替、ピックアップ設定
- 貸出関連操作: 貸出処理、返却処理、延長処理、予約処理
- 認可失敗: 権限外アクセス、未認証アクセス

## 3. 出力先
- 永続監査ログ: `audit_logs` テーブル
- アプリログ: 構造化ログ（JSON）で標準出力
- 重要失敗（DB保存失敗など）はエラーレベルで出力し、再試行対象とする

## 4. 監査ログの必須項目
- `actor_user_id`: 操作ユーザーID（未認証時は `null`）
- `action`: 操作種別（例: `BOOK_UPDATE`, `LOAN_RETURN`）
- `resource_type`: 対象種別（例: `BOOK`, `LOAN`, `USER`）
- `resource_id`: 対象ID（特定不可の場合は `null`）
- `payload`: 追加情報（JSON）
- `created_at`: 記録日時（Asia/Tokyo基準）

## 5. action 命名規則
- 形式: `RESOURCE_OPERATION`
- 例:
  - `BOOK_CREATE`
  - `BOOK_UPDATE`
  - `BOOK_DELETE`
  - `USER_STATUS_UPDATE`
  - `LOAN_CHECKOUT`
  - `LOAN_RETURN`
  - `LOAN_EXTEND`
  - `RESERVATION_CREATE`
  - `AUTHZ_DENIED`

## 6. payload 共通項目
- `request_id`: リクエスト相関ID
- `result`: `SUCCESS | FAILURE`
- `reason`: 失敗理由（成功時は省略可）
- `before`: 更新前情報（必要時）
- `after`: 更新後情報（必要時）
- `ip`: クライアントIP
- `user_agent`: クライアント情報

## 7. 出力タイミング
- 成功時: トランザクション確定後に記録
- 失敗時: 失敗確定時点で記録（`result=FAILURE`）
- 認可失敗: 即時記録し、業務テーブル更新より優先

## 8. 機微情報の取り扱い
- パスワード、トークン、セッションIDは `payload` に保存しない
- 個人情報は必要最小限に限定する
- 監査ログに含める文字列はサイズ制限を設ける（過大データ防止）

## 9. 実装ルール（Go API）
- 監査ログ出力を共通関数化し、ハンドラごとの独自実装を避ける
- 監査ログ失敗時も業務処理の成否を明確に分離して扱う
- すべての管理系APIに `request_id` を付与し、ログに連携する

## 10. 運用ルール
- 保持期間: MVPでは最低1年保持
- 検索条件の標準化:
  - 時間範囲
  - actor_user_id
  - action
  - resource_type/resource_id
- 障害調査時は `request_id` 起点でアプリログと監査ログを突合する

## 11. チェックリスト
- [ ] 管理系操作が `audit_logs` に記録される
- [ ] 認可失敗が `AUTHZ_DENIED` として記録される
- [ ] `payload` から機微情報が除外されている
- [ ] `request_id` で追跡できる
