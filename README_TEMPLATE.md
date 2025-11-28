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