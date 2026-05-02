# GOTH Stack Base Template

**Go + Echo + templ + Tailwind CSS + sqlc + SQLite**

Web アプリケーション開発のベーステンプレートです。
認証（Magic Link / WebAuthn）、ユーザー管理、DB マイグレーション、ホットリロード、バックアップを備えています。

## 技術スタック

| カテゴリ | 技術 |
|---|---|
| 言語 | Go 1.25+ |
| フレームワーク | Echo v4 |
| テンプレート | templ |
| スタイル | Tailwind CSS（CLI ローカルビルド） |
| データベース | SQLite (modernc.org/sqlite, CGO free) |
| SQL 生成 | sqlc |
| マイグレーション | goose |
| 認証 | nz-magic-link (Magic Link + WebAuthn/Passkey) |
| バックアップ | Litestream → Cloudflare R2 |
| デプロイ | fly.io (Docker) / VPS (Docker + Caddy) |

## 使い方

### 1. プロジェクトの作成

このリポジトリをテンプレートとして使用するか、クローンして新しいリポジトリを作成します。

GitHub の "Use this template" を使うか、ローカルで:

```bash
git clone git@github.com:naozine/project_crud_with_auth_tmpl.git my-app
cd my-app
rm -rf .git && git init
```

その上で、以下の **派生プロジェクト初期化チェックリスト** を順に進めます。

### 派生プロジェクト初期化チェックリスト

#### A. プロジェクト識別子

- [ ] **モジュールパスの置換**

  全ファイルの `github.com/naozine/project_crud_with_auth_tmpl` を新しいパスに置換します。

  ```bash
  OLD=github.com/naozine/project_crud_with_auth_tmpl
  NEW=github.com/your-name/my-app
  go mod edit -module $NEW
  grep -rl $OLD --include='*.go' --include='*.templ' --include='Makefile' . \
    | xargs sed -i '' "s|$OLD|$NEW|g"   # macOS の sed は -i '' が必要
  go mod tidy
  make generate
  ```

- [ ] **フォルダ名**

  `Makefile` は `notdir $(CURDIR)` から `PROJECT_NAME` を生成し、Docker イメージ名・Cookie 名・fly.io app 名・Litestream バケット名に流用します。フォルダ名 = アプリ識別子になる前提で命名してください（`_` は `-` に自動変換）。

#### B. ブランディング

- [ ] **アプリ名 (UI 表示)**: `cmd/server/routes_business.go` の `appconfig.AppName = "..."`
- [ ] **WebAuthn RP Name**:
  - デフォルト値: `cmd/server/main.go` 内の `mlConfig.WebAuthnRPName = "Project CRUD"`
  - 環境変数経由: `docker-compose.yaml` / `docker-compose.dev.yaml` の `WEBAUTHN_RP_NAME`
- [ ] **シェルレイアウト・ロゴ**: `web/layouts/shell.templ`, `web/layouts/guest.templ` 内のアプリ名表示と SVG アイコン
- [ ] **Favicon**: `web/static/favicon.svg`, `web/static/favicon.ico`

ハードコードされたアプリ名の見つけ方:

```bash
grep -rn "Project CRUD\|プロジェクト管理" --include='*.go' --include='*.templ' --include='*.yaml' .
```

#### C. 環境設定

- [ ] `.env` を作成（`.gitignore` で除外済み）し、最低限以下を設定:

  ```env
  APP_ENV=dev
  ADMIN_EMAIL=you@example.com   # 初回起動時に admin ユーザーが自動作成される
  ADMIN_NAME=Your Name
  PORT=8080

  # 本番のみ (dev では .bypass_emails があるので不要)
  SMTP_HOST=smtp.example.com
  SMTP_PORT=587
  SMTP_USERNAME=...
  SMTP_PASSWORD=...
  SMTP_FROM=noreply@example.com
  ```

- [ ] `cp .env.production.example .env.production` し、本番用の値を入れる
- [ ] 初回 `air` 起動後、`ADMIN_EMAIL` のユーザーが DB に作成されたことを `.dev_local/` 等のツールで確認

#### D. ビジネスロジックの実装

既存の `projects` テーブル / ハンドラ / テンプレートを参考に、以下を独自モデルで置き換えます。`*_business.*` のサフィックスは「ビジネスロジック領域」を示すマーカーです。

- [ ] `db/schema_business.sql` — 独自テーブル定義
- [ ] `db/query_business.sql` — sqlc 用クエリ
- [ ] `db/migrations/<timestamp>_init.sql` — `make migrate-new NAME=create_xxx` で雛形作成
- [ ] `internal/handlers/business_*.go` — ハンドラ実装
- [ ] `internal/routes/business.go` — ルート登録
- [ ] `web/components/*.templ` — UI コンポーネント
- [ ] `cmd/server/routes_business.go` の `RedirectURL` — ログイン後のランディング先
- [ ] `internal/integration/permission_test.go` の routes 配列 — 新ルートの権限期待値

#### E. デプロイ準備（必要なものだけ）

##### fly.io
- [ ] `cp fly.toml.example fly.toml`
- [ ] `make fly-setup`（初回のみ。アプリ作成 + Volume 作成）
- [ ] シークレット設定（README の "デプロイ" セクション参照）
- [ ] `make fly-deploy`

##### VPS (Docker + Caddy)
- [ ] `cp deploy.config.example deploy.config`（接続先・ドメインを設定）
- [ ] `cp .env.production.example .env.production`
- [ ] `make caddy-setup`（初回のみ）
- [ ] `make docker-deploy`

#### F. ドキュメント整備

- [ ] `README.md` を派生プロジェクト用の説明に書き換える（現状は「これはテンプレート」）
- [ ] 不要なら `GEMINI.md` を削除（プロジェクトで Gemini を使わない場合）
- [ ] `CLAUDE.md` のルールは流用してよい（必要に応じてプロジェクト固有のルールを追記）

#### G. テンプレート更新の取り込み（任意）

将来テンプレート側の改善（CI/lint 設定、認証ライブラリ更新など）を取り込みたい場合:

```bash
git remote add template git@github.com:naozine/project_crud_with_auth_tmpl.git
git fetch template
git merge template/master --allow-unrelated-histories
```

詳細は本書末尾の「テンプレート更新の取り込み」を参照。

#### H. 動作確認

- [ ] `air` で起動 → ログインページにアクセスできること
- [ ] dev 環境で `.bypass_emails` または開発ログ経由で magic link を取得し、admin としてログインできること
- [ ] 自分が実装したビジネス機能の golden path 動作確認
- [ ] `make check` で fmt / vet / lint / test がすべて緑
- [ ] GitHub に push し、Actions の CI が緑（go build / test / golangci-lint / govulncheck）

ここまで通ればテンプレートの初期化は完了です。

### 2. ビジネスロジックの実装

既存の `projects` 機能を参考に、以下のファイルを編集・追加します。

- `cmd/server/routes_business.go` — アプリ名・リダイレクト設定
- `db/schema_business.sql`, `db/query_business.sql` — テーブル・クエリ定義
- `internal/handlers/business_*.go` — ハンドラー
- `internal/routes/business.go` — ルーティング
- `web/components/*.templ` — UI コンポーネント

### 3. 開発コマンド

| コマンド | 説明 |
|---|---|
| `air` | ホットリロードでサーバー起動 |
| `make generate` | コード生成（sqlc + templ + Tailwind CSS） |
| `make build` | ビルド |
| `make migrate-new NAME=create_xxx` | マイグレーションファイル作成 |

### 4. CSS の開発

Tailwind CSS はローカルビルドです（CDN 不使用）。

- 入力: `web/static/css/input.css`
- 出力: `web/static/css/style.css`（生成ファイル）
- `make generate` または `air` で自動ビルドされます
- Tailwind CLI のインストールが必要です

## デプロイ

### fly.io

```bash
# 初回セットアップ
make fly-setup

# シークレット設定
fly secrets set -a <アプリ名> \
  ADMIN_EMAIL=admin@example.com \
  SMTP_HOST=smtp.example.com \
  SMTP_PORT=587 \
  SMTP_USERNAME=user \
  SMTP_PASSWORD=pass \
  SMTP_FROM=noreply@example.com

# デプロイ
make fly-deploy

# カスタムドメイン設定（オプション）
make fly-dns-setup
```

その他: `make fly-status`, `make fly-logs`

### VPS (Docker)

```bash
# deploy.config を設定
cp deploy.config.example deploy.config

# 環境変数を設定
cp .env.production.example .env.production

# デプロイ
make docker-deploy

# Caddy 設定（初回のみ）
make caddy-setup
```

その他: `make docker-remote-logs`, `make docker-restart`

### Litestream バックアップ（オプション）

Cloudflare R2 への SQLite リアルタイムバックアップ。環境変数の有無で有効/無効が切り替わります。

```bash
# R2 バケット作成
make fly-litestream-setup

# シークレット設定
make fly-litestream-secrets

# 状態確認
make fly-litestream-status
```

ローカルからのリストア: `make ls-restore`, `make ls-restore-timestamp TIMESTAMP="2026-03-15T04:00:00Z"`

### ビルド時に自動設定される値

以下はデプロイ時に自動設定されます（.env での設定不要）:

- `SERVER_ADDR`: `https://$(PUBLIC_HOST)` から生成
- `WEBAUTHN_RP_ID`: `SERVER_ADDR` のホスト名部分
- `WEBAUTHN_ALLOWED_ORIGINS`: `SERVER_ADDR` と同じ

開発時（`air` 使用時）は `http://localhost:PORT` が自動で使用されます。

## テンプレート更新の取り込み

### リモートリポジトリの追加（初回のみ）

```bash
git remote add template https://github.com/naozine/project_crud_with_auth_tmpl.git
```

### 更新のマージ

```bash
git fetch template
git merge template/master --allow-unrelated-histories
```

コンフリクトが発生した場合は、ビジネスロジック領域（`*_business.*` 等）は自分の変更を優先し、その後 `go mod tidy && make generate` を実行します。

派生プロジェクトが大きく乖離している場合は、マージよりも AI エージェントに「この機能を参考に実装して」と指示する方が現実的なこともあります。

## ファイル構成

- `*_business.*` はビジネスロジック領域。自由に変更・リネーム可。
- `internal/database/*.go`（sqlc 自動生成）は `.gitignore` で除外。CI/CD ではビルド前に `make generate` を実行。
- `db/*.sql` に日本語を書かないこと（sqlc のコード生成がバグる）。
