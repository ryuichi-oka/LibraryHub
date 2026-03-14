# LibraryHub 設計（MVP）

## 1. 設計方針
- 本設計は社内利用限定のMVPを対象とする
- まずは業務運用を安定させることを優先し、拡張機能（SSO、メール通知）は差し込み可能な構造にする
- 要件ID（FR/NFR/CA）とタスクID（T-xxx）の対応を維持し、仕様変更時の追跡性を確保する

## 2. システム構成（論理）
- `Frontend (Next.js)`:
  - 一般利用者/管理者が利用するWeb UIを提供
- `Backend API (Go)`:
  - 認証、蔵書、検索、貸出/返却/予約、通知、履歴を提供
- `Reverse Proxy (Nginx)`:
  - 入口ルーティング、静的配信、API転送を担当
- `Database (PostgreSQL)`:
  - ユーザー、蔵書、貸出、予約、通知、監査ログを保存
- `Batch/Scheduler (Go)`:
  - 延滞判定、督促通知、新着/おすすめ集計を定期実行

## 2.1 採用技術スタック（確定）
- Frontend: `Next.js 15` + TypeScript（App Router）
- Backend: `Go 1.24`
- Reverse Proxy: `Nginx 1.26`
- Database: `PostgreSQL 16`
- 開発環境: `WSL2 + Docker Compose`

## 3. ロールと権限制御
- `一般利用者`:
  - ログイン
  - 蔵書検索/閲覧
  - 貸出申請、予約、返却状況確認
  - 新着/おすすめ閲覧
  - 通知閲覧
- `管理者`:
  - 一般利用者機能に加えて、蔵書CRUD、ユーザー有効/無効、貸出/返却処理、ピックアップ設定、監査参照
- API は各エンドポイントでロールチェックを行い、権限外アクセスは拒否する

## 4. 画面設計（MVP）
- `ログイン画面`
  - メールアドレスまたは社員ID + パスワード
- `ホーム画面`
  - 新着一覧
  - おすすめ一覧（人気70% + ピックアップ30%）
  - 自分の貸出中一覧と期限近接通知
- `蔵書検索画面`
  - 検索条件: 書名、著者、ISBN、カテゴリ
  - 絞り込み: カテゴリ、貸出状態
  - 表示: 基本情報 + 貸出可否
- `貸出管理画面（管理者）`
  - 貸出処理、返却処理、予約状況確認
- `蔵書管理画面（管理者）`
  - 蔵書の登録/編集/削除、複本管理、カテゴリ更新
- `通知・履歴画面`
  - 通知履歴
  - 貸出/返却/予約履歴
  - 管理操作ログ（管理者のみ）

## 5. データ設計（最小）

- 物理スキーマ定義は [schema.sql](./schema.sql) を参照
- 人間向けテーブル定義書は [table-definitions.md](./database/table-definitions.md) を参照

### 5.1 エンティティ
- `users`
  - `id`, `employee_id`, `email`, `password_hash`, `role`, `status`, `created_at`, `updated_at`
- `books`
  - `id`, `title`, `author`, `isbn`, `publisher`, `published_year`, `category_id`, `location`, `status`, `created_at`, `updated_at`
- `book_copies`
  - `id`, `book_id`, `copy_code`, `status`, `created_at`, `updated_at`
- `categories`
  - `id`, `name`, `is_active`
- `loans`
  - `id`, `user_id`, `book_copy_id`, `loaned_at`, `due_at`, `returned_at`, `extension_count`, `status`
- `reservations`
  - `id`, `user_id`, `book_id`, `priority`, `status`, `created_at`, `notified_at`
- `notifications`
  - `id`, `user_id`, `type`, `message`, `scheduled_at`, `sent_at`, `read_at`, `status`
- `admin_picks`
  - `id`, `book_id`, `start_at`, `end_at`, `priority`, `is_active`
- `audit_logs`
  - `id`, `actor_user_id`, `action`, `resource_type`, `resource_id`, `payload`, `created_at`

### 5.2 状態定義（主要）
- `book_copy.status`: `AVAILABLE | ON_LOAN | RESERVED | INACTIVE`
- `loan.status`: `ON_LOAN | OVERDUE | RETURNED`
- `reservation.status`: `WAITING | NOTIFIED | FULFILLED | CANCELED`
- `user.status`: `ACTIVE | INACTIVE`

## 6. 業務ルール設計

### 6.1 貸出・延長
- 貸出上限: 1利用者あたり3冊
- 貸出期間: 14日
- 延長: 1回まで、7日
- 次予約がある資料は延長不可

### 6.2 延滞・督促
- 返却期限翌日から延滞扱い
- 延滞中は新規貸出不可、返却完了で解除
- 督促通知は3日ごとにアプリ内通知で送信
- 金銭ペナルティなし

### 6.3 新着・おすすめ
- 新着: `books.created_at` の降順を基準に表示
- おすすめ: 人気 + ピックアップのハイブリッド
- 人気: 直近30日の貸出数を基準に上位を抽出
- 初期比率: 人気70% / ピックアップ30%
- 比率は管理者設定で変更可能

## 7. API設計（最小）
- `POST /auth/login`
- `POST /auth/logout`
- `GET /me`
- `GET /books`
- `POST /admin/books`
- `PATCH /admin/books/{bookId}`
- `DELETE /admin/books/{bookId}`
- `POST /loans`
- `POST /admin/loans/{loanId}/checkout`
- `POST /admin/loans/{loanId}/return`
- `POST /loans/{loanId}/extend`
- `POST /reservations`
- `GET /home/new-arrivals`
- `GET /home/recommendations`
- `GET /notifications`
- `GET /histories/loans`
- `GET /admin/audit-logs`
- `POST /admin/picks`
- `PATCH /admin/picks/{pickId}`
- `POST /admin/users/{userId}/status`

## 8. バッチ/定期処理設計
- `B-001 延滞判定ジョブ`:
  - 毎日実行し、`due_at < 当日` の未返却貸出を `OVERDUE` に更新
- `B-002 督促通知ジョブ`:
  - 毎日09:00（Asia/Tokyo）に実行し、延滞開始日から3日単位で対象者に通知を作成
- `B-003 おすすめ集計ジョブ`:
  - 人気スコアを再計算し、表示候補を更新
- `B-004 バックアップジョブ`:
  - 日次バックアップと復旧検証ログを更新

## 9. 監査・運用設計
- 監査ログの実装ルールは [audit-log-guidelines.md](./development/audit-log-guidelines.md) を参照
- 監査対象:
  - 蔵書更新、ユーザー状態変更、貸出/返却の管理操作
- 監査ログ方針:
  - `誰が / いつ / 何を / どの対象に / 結果` を記録
- 運用手順:
  - 障害時の手動運用フロー（貸出/返却の暫定記録、復旧後反映）を別紙運用手順として管理

## 10. 非機能設計
- 日時処理の実装ルールは [datetime-guidelines.md](./development/datetime-guidelines.md) を参照
- 性能:
  - 検索/一覧APIはページング前提
  - 主要検索キー（title, author, isbn, status, category）にインデックスを付与
- セキュリティ:
  - パスワードは強固なハッシュで保存
  - 認可失敗は監査記録する
- 可用性:
  - 日次バックアップ + 復旧手順を運用に組み込む
- 時刻:
  - 保存・表示・業務判定は `Asia/Tokyo` を基準とする

## 11. 要件トレーサビリティ（抜粋）
- 認証（FR-001〜FR-005）: 3章, 4章, 7章
- 蔵書管理（FR-010〜FR-012）: 4章, 5章, 7章
- 検索（FR-020〜FR-022）: 4章, 7章, 10章
- 貸出/予約（FR-030〜FR-038）: 5章, 6章, 7章
- 通知/延滞（FR-040〜FR-048）: 6章, 8章
- 履歴/監査（FR-050〜FR-051）: 5章, 9章
- 新着/おすすめ（FR-060〜FR-064）: 4章, 6章, 8章

## 12. 次期改修（設計余白）
- メール通知チャネル追加（FR-044）
- 社内SSO連携（FR-005）
- おすすめロジック高度化（行動履歴重み付けなど）
