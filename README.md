# LibraryHub

LibraryHub は、蔵書の管理や検索、利用状況の把握をしやすくする図書アプリを作るためのプロジェクトです。
この README は、人が素早く全体像を理解するための入口であり、AI が作業前に目的と配置を把握するための最小コンテキストでもあります。

## プロジェクト概要

- 目的: 図書アプリ `LibraryHub` の要件整理、設計、実装タスクを一貫して管理する
- 対象: 開発メンバー、人間のレビュアー、実装や文書整理を支援する AI
- 現在地: まずは `docs/` 配下で要件・設計・タスクを整理し、そこから実装へつなげる前提の構成

## プロジェクト構造

```text
LibraryHub/
├── README.md
├── AGENTS.md
├── .env.example
├── docker-compose.yml
├── apps/
│   ├── api/
│       ├── README.md
│       ├── cmd/server/main.go
│       ├── internal/
│       └── go.mod
│   └── web/
│       ├── app/
│       └── package.json
├── infra/
│   ├── docker/
│   │   ├── api/Dockerfile
│   │   ├── web/Dockerfile
│   │   └── README.md
│   └── nginx/
│       ├── conf.d/default.conf
│       └── README.md
├── db/
│   ├── migrations/
│   └── README.md
└── docs/
    ├── development/
    │   ├── audit-log-guidelines.md
    │   └── datetime-guidelines.md
    ├── database/
    │   └── table-definitions.md
    ├── requirements.md
    ├── design.md
    ├── tasks.md
    ├── setup.md
    └── schema.sql
```

- [README.md](./README.md): プロジェクトの入口。概要と構造を簡潔にまとめる
- [AGENTS.md](./AGENTS.md): AI エージェント向けの作業ルールと利用可能スキルの案内
- [.env.example](./.env.example): ローカル開発用の環境変数テンプレート
- [docker-compose.yml](./docker-compose.yml): Next.js / Go / Nginx / PostgreSQL の起動定義
- [apps/api](./apps/api): Go製 API（`/auth/login`, `/healthz`）
- [apps/web](./apps/web): Next.js フロントエンドアプリ配置場所
- [infra](./infra): Nginx と Docker 関連のインフラ設定
- [db](./db): マイグレーションなどDB関連ファイル
- [docs/requirements.md](./docs/requirements.md): アプリで満たすべき要件を整理する場所
- [docs/design.md](./docs/design.md): 画面、データ、振る舞いなどの設計をまとめる場所
- [docs/tasks.md](./docs/tasks.md): 実装や検討のタスクを管理する場所
- [docs/setup.md](./docs/setup.md): 開発環境の初期セットアップ手順
- [docs/schema.sql](./docs/schema.sql): MVP の最小データモデル（PostgreSQL DDL）
- [docs/database/table-definitions.md](./docs/database/table-definitions.md): 人間向けのテーブル定義書
- [docs/development/datetime-guidelines.md](./docs/development/datetime-guidelines.md): 日時処理の実装ルール
- [docs/development/audit-log-guidelines.md](./docs/development/audit-log-guidelines.md): 監査ログの共通出力方針
- [docs/development/git-guidelines.md](./docs/development/git-guidelines.md): Issue / Branch / PR / ラベル運用ルール

## 人とAIへのメモ

- 人向け: まず README で全体像を確認し、その後 `docs/` を更新しながら仕様を固める
- AI 向け: 作業前に README と `AGENTS.md` を読み、必要に応じて `docs/` を参照して意図を揃える
