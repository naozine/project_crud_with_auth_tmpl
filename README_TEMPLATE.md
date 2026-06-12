# GOTD Stack Base Template

**Go + templ + Tailwind + Datastar** (chi / sqlc / SQLite)

GOTH Stack (htmx 版) の Datastar 版という位置付け。reactive な UI を SSE で扱うため、フォーム送信→部分更新の体験を htmx より細かく作りたいケースに向く。

Web アプリケーション開発のベーステンプレートです。
認証（Magic Link / WebAuthn）、ユーザー管理、DB マイグレーション、ホットリロードを備えています。

## 技術スタック

| カテゴリ | 技術 |
|---|---|
| 言語 | Go 1.25+ |
| ルーター | chi/v5 |
| テンプレート | templ |
| スタイル | Tailwind CSS（CLI ローカルビルド） |
| データベース | SQLite (modernc.org/sqlite, CGO free) |
| SQL 生成 | sqlc |
| マイグレーション | goose |
| 認証 | nz-magic-link (Magic Link + WebAuthn/Passkey) |
| リアルタイム | Datastar (SSE) |

> 本番デプロイは別リポジトリ [`nz-vps-ops`](https://github.com/naozine/nz-vps-ops)（Ansible + systemd + Caddy）で管理しています。本リポジトリは「Go アプリのソース + ローカル開発」に責務を絞っています。Litestream（R2 バックアップ）等の運用構成も `nz-vps-ops` 側に集約しています。

## 派生プロジェクトの作成

このテンプレートは **Claude Code に初期化作業を委譲する前提** で設計されています。手動チェックリストもありますが、実用的には以下の流れを推奨します。

### 推奨フロー: Claude Code に丸投げ

1. **GitHub の "Use this template"** で新リポジトリを作成、もしくはローカルでクローン:

   ```bash
   git clone git@github.com:naozine/project_crud_with_auth_tmpl.git my-app
   cd my-app
   rm -rf .git && git init && git add . && git commit -m "initial commit"
   ```

2. Claude Code を起動し、初期化を依頼:

   ```
   このリポジトリは GOTD Stack のテンプレートです。docs/init-with-claude.md
   の手順に従って、以下の値で派生プロジェクトとして初期化してください。

   - モジュールパス: github.com/your-name/my-app
   - アプリ名 (UI 表示): タスク管理
   - WebAuthn RP Name: Task Manager
   - 業務ドメイン名 (projects → tasks にリネーム): tasks
   - 管理者メール: you@example.com
   ```

3. Claude が以下を自動実行:
   - モジュールパス置換 (`go.mod`, 全 import)
   - アプリ名・WebAuthn RP Name の差し替え
   - シェルレイアウトのロゴテキスト変更
   - 業務ドメイン (`projects` → `tasks`) のスキーマ・ハンドラ・テンプレ・テストのリネーム / 雛形再生成
   - `.env.example` 等のサンプル値更新
   - `make check` 緑確認
   - 初回コミット

詳細は [`docs/init-with-claude.md`](./docs/init-with-claude.md) を参照。

### 補足: `gonew` を使う場合

モジュールパス置換だけは Go 公式の `gonew` で行うとより安全です:

```bash
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/naozine/project_crud_with_auth_tmpl github.com/your-name/my-app
cd my-app
```

その後、上記の Claude Code 初期化フローのステップ 2 以降を実行してください。

### 補足: 完全手動でやる場合

Claude を使わず、手作業で初期化したい場合の詳細チェックリストを以下に残しておきます。各項目は Claude が裏で実行している内容と同じです。

<details>
<summary>手動チェックリスト（クリックで展開）</summary>

#### A. プロジェクト識別子

- [ ] **モジュールパスの置換**

  ```bash
  OLD=github.com/naozine/project_crud_with_auth_tmpl
  NEW=github.com/your-name/my-app
  go mod edit -module $NEW
  grep -rl $OLD --include='*.go' --include='*.templ' --include='Makefile' . \
    | xargs sed -i '' "s|$OLD|$NEW|g"   # macOS の sed は -i '' が必要
  go mod tidy
  make generate
  ```

- [ ] **フォルダ名**: `Makefile` は `notdir $(CURDIR)` から `PROJECT_NAME` を生成し、Cookie 名やバージョン情報の埋め込みに流用します（`_` は `-` に自動変換）。

#### B. ブランディング

- [ ] **アプリ名 (UI 表示)**: `cmd/server/routes_business.go` の `appconfig.AppName = "..."`
- [ ] **WebAuthn RP Name**:
  - デフォルト: `cmd/server/main.go` 内の `mlConfig.WebAuthnRPName = "Project CRUD"`
  - 本番環境変数 `WEBAUTHN_RP_NAME` は `nz-vps-ops` 側で注入
- [ ] **シェルレイアウト・ロゴ**: `web/layouts/shell.templ`, `web/layouts/guest.templ` 内のアプリ名表示と SVG アイコン
- [ ] **Favicon**: `web/static/favicon.svg`, `web/static/favicon.ico`

```bash
grep -rn "Project CRUD\|プロジェクト管理" --include='*.go' --include='*.templ' --include='*.yaml' .
```

#### C. 環境設定

- [ ] `cp .env.example .env` し、`ADMIN_EMAIL` 等を自分の値に書き換える（全環境変数の説明は `.env.example` 参照。コードに環境変数を追加したら `.env.example` にも記載する — `make check-env-docs` で検証される）
- [ ] 本番の環境変数を `nz-vps-ops` の `apps/<公開ホスト名>.yml`（`app_env_vars`）で設定する（変数の意味は `.env.example` が正）

#### D. ビジネスロジックの実装

既存の `projects` テーブル / ハンドラ / テンプレートを参考に、以下を独自モデルで置き換えます。

- [ ] `db/schema_business.sql` — 独自テーブル定義
- [ ] `db/query_business.sql` — sqlc 用クエリ
- [ ] `db/migrations/<timestamp>_xxx.sql` — `make migrate-new NAME=create_xxx` で雛形作成
- [ ] `internal/handlers/business_*.go` — ハンドラ実装
- [ ] `internal/routes/business.go` — ルート登録
- [ ] `web/components/*.templ` — UI コンポーネント
- [ ] `cmd/server/routes_business.go` の `RedirectURL` — ログイン後のランディング先
- [ ] `internal/integration/permission_test.go` の routes 配列 — 新ルートの権限期待値

#### E. ドキュメント整備

- [ ] `README.md` を派生プロジェクト用の説明に書き換える
- [ ] `CLAUDE.md` のルールは流用

#### F. 動作確認

- [ ] `air` で起動 → ログイン → 業務機能の動作確認
- [ ] `make check` で fmt / vet / lint / test がすべて緑
- [ ] GitHub に push し、CI が緑

</details>

## 開発環境セットアップ

Go 1.26+ が入っていれば、あとは2コマンドで開発を始められます:

```bash
make install   # 開発ツール一式をインストール（templ / goose / sqlc / air / golangci-lint / tailwindcss）
cp .env.example .env   # ADMIN_EMAIL 等を自分の値に書き換える
make dev       # ホットリロードでサーバー起動 → http://localhost:8080
```

- Go 製ツールは `go install`、tailwindcss はバイナリ直接ダウンロード（macOS / Linux 対応）
- バージョンは固定方針: templ / goose は go.mod に、tailwindcss は Makefile の `TAILWIND_VERSION` に、golangci-lint は CI に揃う
- `$(go env GOPATH)/bin` が PATH に含まれている必要があります

## 開発コマンド

| コマンド | 説明 |
|---|---|
| `make install` | 開発ツール一式をインストール |
| `make dev` (= `air`) | ホットリロードでサーバー起動 |
| `make generate` | コード生成（sqlc + templ + Tailwind CSS） |
| `make build` | ビルド |
| `make migrate-new NAME=create_xxx` | マイグレーションファイル作成 |
| `make fmt` / `make vet` / `make lint` / `make test` | 個別チェック |
| `make check` | 上記 + check-roles / check-env-docs をまとめて実行 |
| `make vuln` | 脆弱性スキャン (govulncheck) |

## CSS の開発

Tailwind CSS はローカルビルドです（CDN 不使用）。

- 入力: `web/static/css/input.css`
- 出力: `web/static/css/style.css`（生成ファイル）
- `make generate` または `air` で自動ビルド
- Tailwind CLI のインストールが必要

## Datastar の更新

`web/static/js/datastar.js` はベンダリングした Datastar 本体です（CDN 不使用。
バージョンはファイル先頭のコメント参照）。JS と Go SDK（`datastar-go`）は独立して
バージョニングされているため、**更新時は両者の同期が必要**です。

- 更新手順: `docs/migrations/2026-05-05-datastar-js-sync.md`
- 更新後の回帰確認: `APP_ENV=dev` で起動し `/datastar-test` の各機能を確認

## デプロイ

本番デプロイは別リポジトリ [`nz-vps-ops`](https://github.com/naozine/nz-vps-ops)（Ansible + systemd + Caddy）で管理しています。本リポジトリは Go アプリのソースとローカル開発環境のみを保持します。

- 本番ランタイム = Go 静的バイナリ + systemd unit（Docker は使わない）
- リバプロ = Caddy（VPS 上で共有）
- バックアップ = Litestream → Cloudflare R2（`nz-vps-ops` で systemd unit として管理）
- ビルド時のバージョン情報埋め込み（`-X 'version.Version=...'` 等）も `nz-vps-ops` の playbook 側で実施

ローカル開発時（`air` 使用時）は `http://localhost:PORT` が自動で使用されます。

## テンプレートとの関係

派生プロジェクトはテンプレートから生成された後、**独立して進化する**ことを前提にしています。`git merge` でテンプレ更新を取り込む運用は、過去の経験上ほぼ機能しませんでした（派生側の変更が大きくなりすぎ、コンフリクト解消が現実的でなくなる）。

代わりに、**改善の取り込みは AI エージェントに依頼する** 運用を取ります。

### 改善の取り込み手順

1. テンプレリポで気になる改善コミット (例: `c5221b7` のようなリファクタ) を特定
2. 派生プロジェクトで Claude Code を起動し、対応するハウツーを参照させる:
   ```
   テンプレリポの docs/migrations/2026-05-05-limits-package.md を参照して、
   この派生プロジェクトにも同じ改善を入れてください。
   ```
3. Claude がコミット差分を読み、派生プロジェクトの構造に合わせて適用

テンプレリポでは重要な改善があった際に [`docs/migrations/`](./docs/migrations/) にハウツーを残します（任意）。

### ライブラリ化の方針

複数の派生プロジェクトで再利用したい機能は、テンプレートに留めるのではなく **独立した Go モジュール** に切り出します:

- `github.com/naozine/nz-magic-link` — 認証 (既存)
- 将来的な候補: 共通ミドルウェア、SSE ヘルパー、ログ基盤など

ライブラリ化された機能は SemVer で管理され、派生プロジェクトでは `go get -u` で取り込めます。

## ファイル構成

### `*_business.*` の意味

`*_business.*` というサフィックスを持つファイルは、**派生プロジェクトで最初に書き換える場所のヒント** として残しています。Core 領域とビジネス領域の境界を厳密に強制するルールではありません（実運用では境界をまたぐ改善が必要になることが多いため）。

派生プロジェクトでは、`*_business.*` 以外のファイルも改善対象として自由に編集してください。

### その他

- `internal/database/*.go`（sqlc 自動生成）は `.gitignore` で除外。CI ではビルド前に `make generate` を実行
- `db/*.sql` に日本語を書かないこと（sqlc のコード生成がバグる）
- `web/components/*_templ.go` も生成ファイル（`.gitignore` で除外）
