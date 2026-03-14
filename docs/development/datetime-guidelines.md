# 日時処理実装ルール（T-003）

## 1. 目的
- タイムゾーン起因の不具合（期限判定ずれ、日付境界ずれ）を防ぐ
- Backend/Frontend/DB/Batch の実装ルールを統一する

## 2. 基本方針
- 保存: `Asia/Tokyo` を基準に扱う
- API入出力: ISO-8601（`+09:00` 付き）を基本とする
- 表示: `Asia/Tokyo` で統一
- 業務日付境界: `Asia/Tokyo` 基準

## 3. DBルール（PostgreSQL）
- 日時カラムは `TIMESTAMPTZ` を使用する
- DBセッションのタイムゾーンは `Asia/Tokyo` に固定する
- `NOW()` は `Asia/Tokyo` 基準の業務時刻として扱う
- 日付比較は `Asia/Tokyo` 前提で直接判定する

例:
```sql
-- 期限超過判定（業務日付基準: Asia/Tokyo）
SELECT id
FROM loans
WHERE status IN ('ON_LOAN', 'OVERDUE')
  AND due_at::date < NOW()::date;
```

## 4. Goルール（Backend/Batch）
- アプリ起動時に `time.LoadLocation("Asia/Tokyo")` をロードし、業務時刻の基準にする
- DB書き込み前に UTC へ変換しない（JST基準のまま扱う）
- APIレスポンスは RFC3339 文字列（`+09:00`）で返す

例:
```go
loc, _ := time.LoadLocation("Asia/Tokyo")
nowJST := time.Now().In(loc)
todayJST := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day(), 0, 0, 0, 0, loc)
```

## 5. Next.jsルール（Frontend）
- APIから受け取った日時は `Asia/Tokyo` 前提で表示する
- 画面表示フォーマットは原則 `YYYY-MM-DD HH:mm`
- 入力時刻は明示的にタイムゾーンを含めて送信する

例:
```ts
const jst = new Intl.DateTimeFormat("ja-JP", {
  timeZone: "Asia/Tokyo",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
});
```

## 6. バッチルール
- 督促通知ジョブは毎日09:00（Asia/Tokyo）実行
- ジョブ内時刻判定は必ず `Asia/Tokyo` 基準で日付切り替えを行う
- 実行ログは `Asia/Tokyo` で記録し、必要時のみUTCを補助情報として出力する

## 7. テストルール
- JST日付境界（00:00付近）を含むテストケースを作成する
- 月末・年末の境界テストを追加する
- 延滞判定で「期限当日」と「期限翌日」の差分を検証する

## 8. チェックリスト
- [ ] DB日時カラムが `TIMESTAMPTZ` である
- [ ] APIがRFC3339形式（`+09:00`）で日時を返す
- [ ] 画面表示が `Asia/Tokyo` で統一されている
- [ ] バッチ判定が `Asia/Tokyo` 基準である
