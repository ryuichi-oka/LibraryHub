# LibraryHub 開発環境セットアップ（T-001）

## 1. 目的
- この手順は、LibraryHub の開発を開始できる共通環境を整えるためのもの
- 本手順は `WSL2 + Dockerコンテナ` 前提の開発環境を対象とする

## 2. 前提
- 対象: 社内開発メンバー
- OS: Windows 11 + WSL2（Ubuntu想定）
- リポジトリ: `LibraryHub`
- 採用スタック: `Next.js + Go + Nginx + PostgreSQL`

## 3. 必須ツール
- Git `2.40+`
- エディタ（推奨: VS Code）
- ターミナル（WSL2 bash）
- Docker Desktop `4.x`（WSL2連携を有効化）
- Docker Compose Plugin `v2+`

## 4. 初期セットアップ手順

### 4.1 リポジトリ取得
```bash
git clone <repo-url>
cd LibraryHub
```

### 4.2 作業ブランチ作成
```bash
git switch -c chore/bootstrap-dev-env
```

### 4.3 ドキュメント確認
- `README.md` を確認し、プロジェクト概要と構成を把握する
- `AGENTS.md` を確認し、作業ルールを把握する
- `docs/requirements.md` と `docs/design.md` の最新内容を確認する
- `docs/tasks.md` で着手対象タスクを確認する

### 4.4 環境変数テンプレート作成
```bash
cp .env.example .env
```

`.env.example` が未作成の場合は、次の最小項目で作成する。

```dotenv
APP_ENV=local
APP_TIMEZONE=Asia/Tokyo
APP_PORT=3000
DB_HOST=localhost
DB_PORT=5432
DB_NAME=libraryhub
DB_USER=libraryhub
DB_PASSWORD=libraryhub
DATABASE_URL=postgres://libraryhub:libraryhub@db:5432/libraryhub?sslmode=disable
JWT_SECRET=change-this-in-dev
JWT_EXPIRES_HOURS=24
```

### 4.5 コンテナ起動（ローカル）
```bash
docker compose up -d
docker compose ps
```

サービス想定:
- `next`: Next.js フロントエンド
- `api`: Go API
- `nginx`: リバースプロキシ
- `db`: PostgreSQL

### 4.6 動作確認
- Git 操作ができること
- `.env` が配置済みであること
- `docker compose ps` で主要コンテナが `Up` であること
- `http://localhost` へアクセスできること（Nginx経由）
- タイムゾーン設定が `Asia/Tokyo` であること

## 5. チェックリスト
- [ ] リポジトリを clone した
- [ ] 作業ブランチを作成した
- [ ] 必須ドキュメントを読んだ
- [ ] `.env` を準備した
- [ ] Docker コンテナを起動した

## 6. 今後の追記ポイント
- `make` または `npm/pnpm/go` の実行コマンド統一
- `docker compose` の開発/テスト用 override 構成
- テスト実行手順と CI 前提条件
