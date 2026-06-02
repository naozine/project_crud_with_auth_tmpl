# 2026-06-02: メンテナンスモード + app_settings 汎用設定テーブル

## Why

メンテナンス作業中などに、**一般ユーザーのログイン受付だけを一時停止**したい。管理者は影響を受けず作業を継続でき、既にログイン中のユーザーのセッションも維持したい。

あわせて、こうした ON/OFF フラグや小さな設定値を置く先として、汎用の key-value テーブル `app_settings` を用意する（メンテモード以外の設定にも流用できる）。

## What

新規ファイル:
- `internal/maintenance/maintenance.go` (状態の取得/設定)
- `internal/handlers/admin_maintenance.go` (管理画面ハンドラ)
- `web/components/admin_maintenance.templ` (管理画面 UI)
- `db/migrations/20260507100000_add_app_settings_table.sql`

既存ファイル変更:
- `db/schema.sql` / `db/query.sql` (`app_settings` + `GetAppSetting` / `UpsertAppSetting`)
- `internal/handlers/auth.go` (`AuthHandler` を `Queries` 保持型に変更 + `LoginPage` にメンテ分岐)
- `web/components/login_form.templ` (`LoginMaintenance` コンポーネント)
- `cmd/server/main.go` (`NewAuthHandler(queries)`)
- `internal/routes/admin.go` / `sse.go` (ルート登録)
- `web/layouts/shell.templ` (サイドナビ/モバイルメニューに管理者限定の導線)
- `internal/integration/testhelper.go` / `maintenance_test.go` (テスト基盤拡張 + 回帰テスト)

## How

### DB 層: app_settings（汎用 key-value）

```sql
-- schema.sql / migration
CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- query.sql （※ sqlc ソースに日本語コメントを入れないこと）
-- name: GetAppSetting :one
SELECT * FROM app_settings WHERE key = ? LIMIT 1;

-- name: UpsertAppSetting :exec
INSERT INTO app_settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP;
```

### maintenance パッケージ

```go
const Key = "maintenance_mode"

// 行が無い／DB エラー時は false（安全側＝サービス継続）
func IsEnabled(ctx context.Context, q *database.Queries) bool {
	s, err := q.GetAppSetting(ctx, Key)
	if err != nil { return false }
	return s.Value == "true"
}

func SetEnabled(ctx context.Context, q *database.Queries, on bool) error {
	val := "false"; if on { val = "true" }
	return q.UpsertAppSetting(ctx, database.UpsertAppSettingParams{Key: Key, Value: val})
}
```

### ログイン画面の分岐（肝: AuthHandler を Queries 保持型にする）

メンテ判定に `Queries` が要るので `AuthHandler` に持たせる。`main.go` の `NewAuthHandler()` → `NewAuthHandler(queries)` も合わせて変更。

```go
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	_, isLoggedIn, _ := appcontext.GetUser(r.Context())
	if maintenance.IsEnabled(r.Context(), h.Queries) {
		if isLoggedIn { // ログイン中はそのままホームへ通す
			http.Redirect(w, r, "/projects", http.StatusSeeOther)
			return
		}
		renderGuest(w, r, "メンテナンス中", components.LoginMaintenance())
		return
	}
	// ... 通常のログインフォーム
}
```

### 管理画面と切替（Datastar SSE）

- `GET /admin/maintenance` (admin 限定) で状態と切替ボタンを表示
- `POST /api/sse/admin/maintenance/toggle` で反転し `window.location.reload()` を実行
- `web/layouts/shell.templ` の管理者ナビに導線を追加

### 生成

```
make generate   # sqlc (app_settings) + templ (admin_maintenance / LoginMaintenance)
```

## 派生プロジェクトへの適用

- ログイン後のリダイレクト先（テンプレは `/projects` 固定）は **派生のホームに合わせる**。
- `LoginMaintenance` の見出しはテンプレでは `appconfig.AppName` を使用。派生のブランディングに合わせる。
- 既に `AuthHandler` を独自拡張している派生では、`Queries` フィールドの追加だけ取り込めばよい。

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-06-02-maintenance-mode.md を参照して、
このプロジェクトにメンテナンスモードと app_settings テーブルを追加してください。
ログイン後のリダイレクト先はこのプロジェクトのホームに合わせてください。
```

## 検証

- `make generate` 後 `go test ./internal/integration/ -run TestMaintenance` 緑（4 本）
- `make vet` / `make lint` 緑

## 関連コミット

- `601348e` メンテナンスモード機能を追加
