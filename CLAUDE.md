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
