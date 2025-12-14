# GOTH Stack Base Template

**Go + Echo + templ + htmx + sqlc + SQLite**

Webアプリケーション開発のための、堅牢かつ拡張性の高いベーステンプレートです。
認証（MagicLink/WebAuthn）、ユーザー管理、DBマイグレーション、ホットリロードなどの必須機能を完備しています。

## コンセプト: コアとビジネスロジックの分離

このテンプレートは、**「不変のコア機能」** と **「可変のビジネスロジック」** を明確に分離するように設計されています。
これにより、将来的にベーステンプレートの機能（認証ロジックの改善など）がアップデートされた際、あなたのプロジェクトへの取り込み（マージ）を最小限のコンフリクトで行うことができます。

### 構成

*   **Core (Do Not Edit):** 認証、ユーザー管理、基本設定など。
    *   `cmd/server/main.go`
    *   `db/schema.sql`, `db/query.sql`
    *   `internal/handlers/auth.go`, `admin.go` 等
*   **Business Logic (Edit Freely):** あなたのアプリケーション固有の機能。
    *   `cmd/server/routes_business.go`: ルーティングとアプリ設定。
    *   `db/schema_business.sql`, `db/query_business.sql`: 独自のDB定義。
    *   `internal/handlers/business_*.go`: 独自のハンドラー。
    *   `web/components/`: 独自のUIコンポーネント。

## 使い方 (Getting Started)

### 1. プロジェクトの作成
このリポジトリをテンプレートとして使用するか、クローンして新しいリポジトリを作成します。

### 2. ビジネスロジックの実装
以下のファイルを編集・追加して、あなたのアプリケーションを構築します。

*   **設定:** `cmd/server/routes_business.go` の `ConfigureBusinessSettings` でアプリ名やリダイレクト先を設定します。
*   **データベース:** `db/schema_business.sql` にテーブルを定義し、`db/query_business.sql` にクエリを書きます。
*   **ロジック:** `internal/handlers/` にハンドラーを作成します（既存の `business_projects.go` を参考にしてください）。
*   **UI:** `web/components/` に `templ` ファイルを作成します。

### 3. 開発コマンド

*   **サーバー起動 (ホットリロード):** `air` または `make run`
*   **コード生成 (sqlc & templ):** `make generate`
*   **ビルド:** `make build`
*   **マイグレーション作成:** `make migrate-new NAME=create_my_table`

## ファイル構成ルール

*   **`*_business.*`**: これらのファイル名は「ビジネスロジック領域」であることを示しています。自由に変更・リネームして構いません。
*   **`.gitignore`**: `internal/database/*.go` (自動生成コード) はバージョン管理から除外されています。CI/CD環境ではビルド前に `make generate` を実行してください。

## 技術スタック

*   **Language:** Go 1.23+
*   **Framework:** Echo v4
*   **Template:** templ
*   **Frontend:** htmx, Alpine.js, Tailwind CSS
*   **Database:** SQLite (modernc.org/sqlite - CGO free)
*   **SQL Generator:** sqlc
*   **Migration:** goose
*   **Authentication:** Magic Link (Email), WebAuthn (Passkey)

---

## ベーステンプレートの更新を取り込む方法

このプロジェクトは `project_crud_with_auth_tmpl` をベースにしています。
ベーステンプレートにセキュリティ修正や新機能が追加された場合、以下の手順であなたのプロジェクトに取り込むことができます。

### 1. リモートリポジトリとしてテンプレートを追加
(初回のみ - 既に設定済みか確認するには `git remote -v` を実行してください)

```bash
git remote add template https://github.com/naozine/project_crud_with_auth_tmpl.git
```

### 2. 更新を取得してマージ

```bash
git fetch template
git merge template/main --allow-unrelated-histories
```

*   **コンフリクトが発生した場合:**
    *   コア機能（`cmd/server/main.go`, `db/schema.sql` 等）のコンフリクトは、基本的に**テンプレート側の変更**を採用してください。
    *   ビジネスロジック領域（`routes_business.go`, `*_business.*` 等）のコンフリクトは、**あなたのプロジェクトの変更**を優先してください。

### 3. 依存関係の更新

```bash
go mod tidy
make generate
```

---

## デプロイ

このテンプレートは **VPS (Docker)** と **fly.io** の2つのデプロイ方法をサポートしています。

### 共通設定

`deploy.config.example` を `deploy.config` にコピーして編集:

```bash
cp deploy.config.example deploy.config
```

主な設定項目:
- `PUBLIC_HOST`: 公開ドメイン名（例: `myapp.example.com`）
- `CF_API_TOKEN`, `CF_ZONE_ID`: Cloudflare DNS 自動設定用（オプション）

### fly.io へのデプロイ

軽量で自動スケーリング対応。スケールトゥゼロでコスト削減可能。

```bash
# 1. 初回セットアップ（アプリ作成 + ボリューム作成 + fly.toml生成）
make fly-setup

# 2. シークレット設定
fly secrets set -a <フォルダ名> \
  ADMIN_EMAIL=admin@example.com \
  SMTP_HOST=smtp.example.com \
  SMTP_PORT=587 \
  SMTP_USERNAME=user \
  SMTP_PASSWORD=pass \
  SMTP_FROM=noreply@example.com

# 3. デプロイ
make fly-deploy

# 4. カスタムドメイン設定（オプション、Cloudflare使用時）
make fly-dns-setup
# → 完了後、Cloudflare でProxy ON (オレンジ雲) に切り替え
```

その他のコマンド:
- `make fly-status`: ステータス確認
- `make fly-logs`: ログ表示

### VPS (Docker) へのデプロイ

自前のVPSにDockerでデプロイ。Caddyをリバースプロキシとして使用。

```bash
# 1. deploy.config を設定（VPS_USER, VPS_HOST, PUBLIC_HOST等）

# 2. 本番用環境変数を設定
cp .env.example .env.production
# .env.production を編集（SMTP設定等）

# 3. DNS設定（Cloudflare使用時、オプション）
make dns-setup

# 4. デプロイ
make docker-deploy

# 5. Caddy設定（初回のみ）
make caddy-setup
```

その他のコマンド:
- `make docker-remote-logs`: コンテナログ表示
- `make docker-restart`: コンテナ再起動

### ビルド時に自動設定される値

以下の値はデプロイ時に自動設定されます（.env での設定不要）:

- `SERVER_ADDR`: `https://$(PUBLIC_HOST)` から生成
- `WEBAUTHN_RP_ID`: `SERVER_ADDR` のホスト名部分
- `WEBAUTHN_ALLOWED_ORIGINS`: `SERVER_ADDR` と同じ

開発時（`air` 使用時）は `http://localhost:PORT` が自動で使用されます。