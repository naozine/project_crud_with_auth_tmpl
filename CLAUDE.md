# ルール

- アプリやコマンドの実行は指示がない限り人間がやる（ビルド検証は可）
- 原則 PRG パターンに従う（フォーム送信 → リダイレクト → GET）

## コード調査

- 関数の呼び出し元・呼び出し先・インターフェースの実装などを調べるときは、LSP (gopls) を優先して使う
  - 例: `Find References`, `Goto Definition`, `Find Implementations`, `Hover`, `Workspace Symbols`
  - grep は文字列マッチで誤検知が多いので、型情報が必要な調査では LSP を選ぶ
  - LSP が使えない環境（gopls 未インストール等）の場合のみ grep にフォールバック

## 作業完了時

- Go ファイルの変更を伴う作業の最後に `make vet` と `make lint` を実行し、結果を報告する
  - `make lint` で `golangci-lint: command not found` が出たらスキップして報告のみでよい
  - 既存コードに由来する警告（自分が変更していない箇所）は、その旨を明記して区別する
