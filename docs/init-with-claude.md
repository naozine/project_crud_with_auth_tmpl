# Claude Code を使った派生プロジェクト初期化

このドキュメントは、`project_crud_with_auth_tmpl` をベースに派生プロジェクトを作る際、Claude Code に渡す指示書として機能します。

## 前提

- このリポジトリを「Use this template」または `git clone + rm -rf .git` で複製済み
- 新リポジトリのディレクトリで Claude Code を起動済み
- ユーザーは派生プロジェクト固有の値（モジュールパス、アプリ名、業務ドメイン名など）を決めている

## ユーザーが Claude に渡すプロンプト例

以下のプロンプトをコピーし、`<...>` を埋めて Claude Code に投げる:

```
このリポジトリは GOTD Stack のテンプレートから複製されたものです。
docs/init-with-claude.md の手順に従って、派生プロジェクトとして初期化してください。

## 設定値
- モジュールパス: <github.com/your-name/my-app>
- アプリ名 (UI 表示): <タスク管理>
- WebAuthn RP Name: <Task Manager>
- 業務ドメイン名: <tasks>  ← projects から置換するキーワード
- 業務ドメイン名 (日本語): <タスク>  ← UI 表示用
- 管理者メール: <you@example.com>

## 完了条件
- make check が緑
- 業務ドメインの CRUD が動く（一覧・新規・編集・削除）
- 初回コミットが切られている
```

## Claude が実行する作業の詳細

### 1. モジュールパスの置換

```bash
OLD=github.com/naozine/project_crud_with_auth_tmpl
NEW=<新しいモジュールパス>

go mod edit -module $NEW
grep -rl "$OLD" --include='*.go' --include='*.templ' --include='Makefile' . \
  | xargs sed -i '' "s|$OLD|$NEW|g"
go mod tidy
```

`gonew` を使った場合はこのステップは不要（`gonew` が同等の処理を行う）。

### 2. アプリ名の置換

#### 2-1. `cmd/server/routes_business.go`
```go
appconfig.AppName = "<新しいアプリ名>"
```

#### 2-2. `cmd/server/main.go` (WebAuthn RP Name のデフォルト)
```go
mlConfig.WebAuthnRPName = "<新しい WebAuthn RP Name>"
```

#### 2-3. `docker-compose.yaml` / `docker-compose.dev.yaml`
```yaml
WEBAUTHN_RP_NAME=${WEBAUTHN_RP_NAME:-<新しい WebAuthn RP Name>}
```

#### 2-4. シェルレイアウトのハードコード文字列
`web/layouts/shell.templ`, `web/layouts/guest.templ` を開き、`Project CRUD` などのハードコード文字列を新しいアプリ名に置換。

検索コマンド:
```bash
grep -rn "Project CRUD\|プロジェクト管理" --include='*.go' --include='*.templ' --include='*.yaml' .
```

### 3. 業務ドメインのリネーム (projects → 新ドメイン)

`projects` を新しい業務ドメイン名（例: `tasks`）に置換します。リネーム対象:

#### 3-1. DB レイヤー
- `db/schema_business.sql` — テーブル名 `projects` → 新名
- `db/query_business.sql` — クエリ名 `ListProjects` 等を全て新名ベースに
- `db/migrations/*.sql` — テーブル作成 SQL の更新（または新規マイグレーション作成）

#### 3-2. Go コード
- `internal/handlers/business_projects.go` → `business_<新名>.go` にリネーム + 中身の `Project` 系を `<新名>` ベースに
- `internal/routes/business.go` の `/projects` パスを `/<新名>` に
- `internal/handlers/sse_projects.go` → `sse_<新名>.go`
- `cmd/server/routes_business.go` の `RedirectURL = "/projects"` を `/<新名>` に

#### 3-3. テンプレート
- `web/components/project_*.templ` → `<新名>_*.templ`
- 中身の表示名も日本語業務ドメイン名に置換

#### 3-4. テスト
- `internal/integration/permission_test.go` の routes 配列を新パスに更新
- `internal/integration/admin_crud_test.go` などの URL を更新

### 4. 環境設定

`.env` を作成 (gitignore で除外済み):

```bash
cp .env.example .env
# ADMIN_EMAIL / ADMIN_NAME を自分の値に書き換える
```

全環境変数の意味は `.env.example` を参照。環境変数を追加したら
`.env.example` にも記載する（`make check-env-docs` で検証される）。

### 5. ドキュメント整備

- `README.md` を派生プロジェクト用に書き換え（テンプレ記述を削除し、アプリの説明に）
- `CLAUDE.md` のルールはそのまま流用

### 6. 検証

```bash
make generate     # sqlc + templ + tailwind
make check        # fmt + vet + lint + test
```

すべて緑であることを確認。

### 7. 初回コミット

```bash
git add -A
git commit -m "派生プロジェクト初期化（<アプリ名>）"
```

## トラブルシューティング

### `make generate` で sqlc がコケる
`db/*.sql` 内に日本語があるとバグる。既存の英語コメントを残し、日本語が混入していないか確認。

### `make lint` で大量警告
テンプレで導入済みの除外設定 (`.golangci.yml`) を継承していれば、業務領域のリネーム後も基本的に警告は出ないはず。出る場合はテンプレリポの最新と差分を確認。

### `make test` で integration テストがコケる
`internal/integration/permission_test.go` の routes 配列が古い URL のままになっていないか確認。新業務ドメインのパスに置換済みか確認。

## テンプレ追従

初期化後にテンプレ側で改善が入った場合、`docs/migrations/` ディレクトリにハウツーが残されている可能性があります。Claude Code に「`<テンプレリポ>/docs/migrations/<ファイル名>` を参照して、この派生にも同じ改善を入れて」と依頼してください。
