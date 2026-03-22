# SSO 連携設計メモ（次期改修向け）

## 1. 目的
- 本メモは、MVP の独自ログイン（`/auth/login`）から社内 SSO へ拡張する際の連携ポイントを整理する
- 要件 `FR-005` / `AC-007` の追跡用メモとして扱い、未確定事項は明示したまま残す

## 2. 現状（MVP）
- API は `POST /auth/login` で識別子（メールまたは社員ID）+ パスワードを受け取り、JWT を返却する
- 認証済み API は `Authorization: Bearer <token>` を前提とする
- Web は `localStorage` に `token / role / expiresAt` を保存して利用する

## 3. SSO 連携ポイント

### 3.1 API 層
- 追加予定エンドポイント:
  - `GET /auth/sso/start`: 認可要求を開始し、SSO 側ログイン画面へ遷移させる
  - `GET /auth/sso/callback`: 認可コードを受け取り、トークン交換とユーザー同定を行う
- 既存の `requireAuth` / `requireRoles` はそのまま利用し、最終的には LibraryHub の JWT へ統一する
- 連携方針:
  - SSO 成功後に `users` を社員IDまたはメールで突合し、既存ユーザーへ紐付ける
  - `users.status = INACTIVE` の場合は SSO 認証成功でもログイン不可とする

### 3.2 Web 層
- ログイン画面に `SSO でログイン` 導線を追加し、`/auth/sso/start` へ遷移する
- コールバック受信後は MVP と同じセッション形式（`token / role / expiresAt`）で保存し、画面側の変更を最小化する

### 3.3 データ層
- `users` に外部IdP識別子（例: `external_subject`）を追加する余地を残す
- 初期移行では「社員IDまたはメールで突合」を優先し、外部識別子は次段階で導入する

## 4. セキュリティ・運用メモ
- OAuth/OIDC の `state` と `nonce` を検証し、CSRF/リプレイを防止する
- 失敗時（ユーザー未登録・無効ユーザー・トークン交換失敗）は監査ログに記録する
- ログアウト時はアプリ内セッション削除を必須とし、IdP 側ログアウト連携は要件確定後に実装する

## 5. 未確定事項
- 社内 IdP の種類（Azure AD / Okta / Keycloak など）
- 採用プロトコル（OIDC Authorization Code + PKCE を想定、最終決定は運用要件と合わせて実施）
- 自動ユーザープロビジョニングの可否（初期は手動運用を想定）
- SSO 専用ログイン移行時期（MVP ログイン併存期間の要否）

## 6. 次に設計で確定する項目
- 認証シーケンス図（開始→コールバック→JWT 発行）
- エラーパターン別の画面遷移（未登録/権限不足/一時障害）
- 監査ログ項目（`actor`, `idp`, `result`, `reason`）の固定化
