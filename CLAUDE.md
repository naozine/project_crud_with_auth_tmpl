# ルール

- アプリやコマンドの実行は指示がない限り人間がやる（ビルド検証は可）
- UI 更新は Datastar の部分更新（patch）を主体にする（→「UI 更新の方針」）

## UI 更新の方針

Datastar の強み（ページ遷移なしの部分更新）を活かす。以前は「原則 PRG」だったが、
編集系はダイアログ/インライン更新を主体にする方針へ転換した。

- **編集・作成・小さな操作はダイアログ/インライン更新を主体にする**。SSE で該当
  要素だけ patch し、`window.location.reload()` や PRG リダイレクトは避ける。
  例: ダイアログで編集 → `@put` → サーバが該当行だけ `PatchElementTempl`（outer/inner）
  で差し替え、`ExecuteScript` でダイアログを閉じる（遷移も reload もしない）。
- **ページ遷移は URL が意味を持つ単位に限定する**（一覧↔詳細など、ブックマーク・
  共有・リロード復元・戻るボタンが要る場面）。この場合は通常の遷移や、必要なら PRG
  （`sse.Redirect` / `ExecuteScript` の location 操作）を使ってよい。
- 判断軸は「その状態を URL で表す必要があるか」。必要なら遷移、不要なら patch。
- お手本: `/datastar/recipes`（特にレシピ⑩ ダイアログ編集）と
  `docs/datastar/datastar-llm-guide.md`。

## UI / 見た目の規約（デザイントークン）

統一感のある見た目を「特に指示しなくても」保つための規約。配色・角丸・フォントは
すべて `web/static/css/input.css` のセマンティックトークンに集約し、テーマ
（Vercel / Notion など）で切り替える。

- **生のパレットを直書きしない**。`bg-blue-500` `text-gray-900` `bg-black`
  `rounded-lg` `indigo-600` などの素の Tailwind 色・任意の角丸は使わない。
  必ず下記のセマンティックなユーティリティを使う:
  - 色: `bg-canvas`（ページ背景）/ `bg-surface`（カード等の面）/ `border-border` /
    `text-ink`（主テキスト）/ `text-muted`（副テキスト）/ `text-faint`（淡いテキスト）/
    `bg-accent` `text-accent` `hover:bg-accent-hover` `text-accent-fg`（アクセント）/
    `bg-danger` `text-danger` `text-danger-fg` / `bg-success` `text-success`。
    淡い面・枠は `bg-accent/10` `ring-danger/30` のように透明度修飾子を使う。
  - 角丸: `rounded-ui`（ボタン・入力）/ `rounded-card`（カード・ダイアログ）。
  - フォント: `font-sans`（`<body>` で適用済み。個別指定は不要）。
- **ボタン・カード・フォーム・バッジ・ダイアログは既存コンポーネントを使う**。
  生のクラス文字列で再実装しない。場所:
  - `web/components/ui_button.templ`: `@PrimaryButton` `@PrimaryActionButton`
    `@PrimarySubmitButton` `@SecondaryButton` `@DangerButton` `@DangerActionButton`
    `@DangerTextButton` `@PrimaryLink` `@SecondaryLink` `@GhostIconButton`（控えめなアイコン操作）
    `@Fab`（右下固定の主アクション。アイコンは `@iconPlus`）ほか
  - `web/components/ui_page.templ`: `@PageHeader` `@SectionCard` `@EmptyState`
    `@Dialog` `@DialogHeader` `@DialogFooter` `@AlertSuccess` `@AlertError`
    `@RoleBadge` `@StatusBadge` `@BackLink`
  - `web/components/ui_form.templ`: `@FormField` `@TextInput` `@DataInput`
    `@RoleSelect` ほか
  - `web/components/ui_table.templ`: `@Table` `@TableHead` `@Th` `@ThRight`
    `@ThTools` `@TableRow` `@Td` `@TdMuted` `@TdRight`（ヘッダ固定のデータテーブル）
- **コンポーネントは外側のマージンを持たない**。要素間の縦の間隔はページ root の
  `space-y-*` で管理する（外側マージンは相殺の有無で効いたり効かなかったりするため）。
- 必要な部品が無い場合は、その場で生クラスを書かず **`ui_*.templ` にコンポーネントを
  追加してから使う**（トークンのみで実装する）。
- **新しいテーマを足したい**ときは `input.css` の `[data-theme="..."]` ブロックを
  1つ追加して値を埋めるだけ（`<html>` の `data-theme` で切り替わる。切替 UI は
  サイドバー下部、適用スクリプトは `web/layouts/head.templ`）。
- **一覧画面の共通レイアウト（3画面で統一済み。新規一覧もこれに合わせる）**:
  - 幅 `max-w-6xl mx-auto`、縦間隔 `space-y-4`。先頭に必ず `@PageHeader(タイトル, 説明)`。
  - **作成系の主アクション（新規作成・追加）は右下の `@Fab` に統一**（アイコン `@iconPlus`）。
    専用の操作行は作らない。作成のないページ（例: アクセスログ）には FAB を置かない。
  - 副次操作・非作成系操作は見出し行の右、または対象近くに控えめに置く:
    ユーザー管理の「一括インポート」は見出し行の右（`flex items-start justify-between` で
    `@PageHeader` と横並び）。アクセスログの「更新」はテーブル見出し右端の
    `@ThTools` + `@GhostIconButton`（モバイルはカード上に右寄せ）。※FAB は構築系の
    主アクション専用とし、更新のような操作には使わない（ユーザー判断、2026-06-10）。
  - 一覧の SSE 更新は **コンテナごと inner 置換**（テーブル＝デスクトップ/カード＝モバイルの
    2系統を同期させるため、行単位 outer 置換にはしない）。reload・PRG はしない。
- お手本（主・テーブル型）: `web/components/access_logs.templ`（`/admin/access-logs`）。
  行が多い一覧の基準。デスクトップは `ui_table` のヘッダ固定テーブル（ページは動かず
  テーブル内だけスクロール、見出し sticky）、モバイルはカード。高さは Shell の fixed な
  `main` から flex で受ける（組み方は `ui_table.templ` 冒頭コメント参照。マジックナンバー禁止）。
  SSE は `#access-logs-list` を `internal/handlers/access_log.go` の `TableSSE` で inner 置換。
  CRUD 付きの同型は `admin_users_list.templ`（`patchUserList`）。
- お手本（カード型）: `web/components/project_list.templ` — URL を持つ（詳細ページのある）
  項目をカードで並べる場合の基準実装。テーブルではなくカードグリッド + ダイアログ作成。
  上記の共通レイアウト（見出し・FAB・幅・余白）はテーブル型と揃える。
- ロール識別色（`@RoleBadge` の admin=紫 / editor=青 / viewer=灰）はカテゴリ色
  として固定。これらはテーマに依存させない例外。

## テンプレートとの関係

このリポジトリはテンプレート (`github.com/naozine/project_crud_with_auth_tmpl`) から派生した可能性がある。テンプレートとの関係についての方針:

- 派生プロジェクトは独立進化が前提。テンプレ側との `git merge` 運用は行わない
- `*_business.*` のサフィックスは「派生で最初に書き換える場所」のヒントであり、Core / Business の境界を強制するものではない。改善のためならどのファイルも編集してよい
- テンプレ側の改善を取り込みたい場合、テンプレリポの `docs/migrations/` を参照して機能単位で移植する

## コード規約

- ロール（admin / editor / viewer）は `internal/roles` の定数（`roles.Admin` / `roles.Editor` / `roles.Viewer`）を使う。文字列リテラルでベタ書きしない。
  - 検証は `roles.IsValid()` を使う。ロールを増減するときは `internal/roles` だけを変更する
  - 違反は `make check-roles`（`make check` に含む）で検出される。テストコードは対象外

## コード調査

- 関数の呼び出し元・呼び出し先・インターフェースの実装などを調べるときは、LSP (gopls) を優先して使う
  - 例: `Find References`, `Goto Definition`, `Find Implementations`, `Hover`, `Workspace Symbols`
  - grep は文字列マッチで誤検知が多いので、型情報が必要な調査では LSP を選ぶ
  - LSP が使えない環境（gopls 未インストール等）の場合のみ grep にフォールバック

## GitHub 運用

- `gh` CLI を積極的に使う（PR の作成・レビュー、issue の操作、リポジトリ情報の確認など）
  - 可能な限り `gh pr`, `gh issue`, `gh diff` などを活用し、`curl` やブラウザ誘導に頼らない

## 作業完了時

- Go ファイルの変更を伴う作業の最後に `make vet` と `make lint` を実行し、結果を報告する
  - `make lint` で `golangci-lint: command not found` が出たらスキップして報告のみでよい
  - 既存コードに由来する警告（自分が変更していない箇所）は、その旨を明記して区別する
