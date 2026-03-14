# Git運用ルール（Issue / Branch / PR）

## 1. 目的
- 変更の意図・履歴・レビュー観点を追跡しやすくする
- 小さく安全に変更を進める
- 誰が作業しても同じ流れで進められる状態にする

## 2. 基本方針
- ブランチ運用は Git Flow 準拠とする
- `main` と `develop` への直接 push は禁止し、必ず Pull Request（PR）経由で反映する
- 1 Issue = 1 Branch = 1 PR を原則とする
- 実装前に Issue で目的と受け入れ条件を明確にする
- PR は小さく保ち、レビュー可能な単位で出す

## 3. Branch 運用

### 3.1 長寿命ブランチ
- `main`: 本番リリース済みコードを保持する
- `develop`: 次リリース向けの統合ブランチとする

### 3.2 作業ブランチ命名規則
- `feature/<issue-number>-<short-description>`: `develop` から作成し、`develop` へマージする
- `release/<version>`: `develop` から作成し、`main` と `develop` へマージする
- `hotfix/<issue-number>-<short-description>`: `main` から作成し、`main` と `develop` へマージする
- `docs/<issue-number>-<short-description>`: 原則 `develop` から作成し、`develop` へマージする
- `chore/<issue-number>-<short-description>`: 原則 `develop` から作成し、`develop` へマージする

例:
- `feature/123-login-api`
- `release/v0.3.0`
- `hotfix/145-fix-login-timeout`
- `docs/188-git-guidelines`

### 3.3 Git Flow ルール
- 通常開発は `feature/*` で行い、PR で `develop` に取り込む
- リリース時は `release/*` を作成し、最終調整後に `main` と `develop` へ反映する
- 緊急修正は `hotfix/*` を `main` から作成し、修正後に `main` と `develop` へ反映する
- `main` へのマージ時はリリースタグ（例: `v0.3.0`）を付与する
- Branch 作成前に対象 Issue を `status:in-progress` に更新する
- Branch が不要になったらマージ後に削除する

## 4. Issue 運用

### 4.1 作成ルール
- 変更作業は原則 Issue 起点で開始する（軽微な typo 修正は除く）
- 必須記載:
  - 背景/目的
  - スコープ（やること/やらないこと）
  - 受け入れ条件
  - 影響範囲（API/UI/DB/Docs/運用）

### 4.2 ステータス運用
- 新規 Issue は `feature` / `bug` などの種別ラベルと `P*` ラベルを付与して分類する
- 方針判断が必要なものは `needs-discussion` を付与する
- 外部要因で停止したものは `blocked` を付与する
- 対応しない判断をしたものは `wontfix` を付与してクローズする
- 完了状態はラベルではなく Issue / PR の `Close` とマージ結果で管理する

## 5. PR 運用

### 5.1 作成ルール
- PR タイトル形式: `<type>: <summary> (#<issue-number>)`
- Draft PR を早めに作成し、方向性を共有する
- PR 本文に以下を必ず記載する:
  - 目的と変更点
  - 影響範囲
  - 確認手順
  - 関連 Issue

### 5.2 レビュー・マージ
- 最低 1 名の承認を得てからマージする
- CI が全て成功していること
- マージ方式は `Squash and merge` を標準とする
- `feature/*` は `develop` へ、`release/*` と `hotfix/*` は `main` と `develop` へ反映する
- マージ後は Issue を `done` に更新し、Branch を削除する

## 6. ラベル運用

### 6.1 作業種別
- `feature` (`0E8A16`): 機能追加
- `bug` (`D73A4A`): 不具合修正
- `refactor` (`FBCA04`): 内部構造の改善（仕様変更なし）
- `docs` (`0075CA`): ドキュメント関連
- `chore` (`C2E0C6`): ビルド・CI・雑務

### 6.2 優先度
- `P0` (`B60205`): 最優先（即対応）
- `P1` (`D93F0B`): 高優先度
- `P2` (`FBCA04`): 通常対応
- `P3` (`0E8A16`): 低優先度

### 6.3 状態・判断
- `blocked` (`000000`): 外部要因などで作業停止中
- `needs-discussion` (`5319E7`): 方針確認が必要
- `wontfix` (`FFFFFF`): 対応しないと判断

## 7. テンプレート利用ルール
- Issue は用途に応じて `Feature request` / `Bug report` テンプレートを使う
- PR は `.github/pull_request_template.md` に従って記入する
- 空欄のまま提出せず、不要項目は「該当なし」と明記する
