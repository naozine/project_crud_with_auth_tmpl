# 2026-05-02: G120 (DoS) 対策の MaxBodySize ミドルウェア

## Why

`make lint` で gosec が以下の 2 種類の警告を出していた:

```
business_projects.go:69 G120: Parsing form data without limiting request body size
business_projects.go:85 G120: Parsing form data without limiting request body size
admin_user_import.go:61 G120: Parsing form data ... (multipart)
```

実害:
- `r.FormValue("name")` は body 全体をメモリに展開する。サイズ制限なしで巨大な POST を送ると **メモリ枯渇 DoS**
- `r.ParseMultipartForm(10<<20)` の `maxMemory` は「メモリに保持する上限」で **body 全体ではない**。残りは一時ファイルに書かれる → **ディスク枯渇 DoS**（こちらの方が地味にやばい）

## What

新規ファイル:
- `internal/middleware/limits.go` (`MaxBodySize` ミドルウェア)
- `internal/integration/body_size_test.go` (TDD ベースの 4 テスト)

既存ファイル変更:
- `internal/handlers/error.go` (`parseFormOr413` ヘルパー追加)
- `internal/handlers/business_projects.go` (Create/Update で `parseFormOr413` を呼ぶ)
- `internal/handlers/admin_user_import.go` (`ParseMultipartForm` のエラー判定で 413 返却)
- `internal/routes/business.go` (各ルートグループに `MaxBodySize` 適用)

## How

### `internal/middleware/limits.go`

```go
package middleware

import "net/http"

// MaxBodySize は受信 HTTP リクエスト body の上限を制限するミドルウェア。
// 上限を超えた場合、後続の r.ParseForm / r.ParseMultipartForm が
// *http.MaxBytesError を返すので、ハンドラ側でエラー型を判別して
// 413 Request Entity Too Large を返すこと。
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, limit)
            next.ServeHTTP(w, r)
        })
    }
}
```

### `parseFormOr413` ヘルパー

`r.FormValue` は内部で `ParseForm` のエラーを **握りつぶす** ので、明示的に判定する必要がある:

```go
func parseFormOr413(w http.ResponseWriter, r *http.Request) bool {
    if err := r.ParseForm(); err != nil {
        var maxBytesErr *http.MaxBytesError
        if errors.As(err, &maxBytesErr) {
            httpError(w, r, http.StatusRequestEntityTooLarge, "リクエストが大きすぎます")
            return true
        }
        httpError(w, r, http.StatusBadRequest, "リクエストの解析に失敗しました")
        return true
    }
    return false
}
```

ハンドラの先頭で:

```go
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
    if parseFormOr413(w, r) {
        return
    }
    name := r.FormValue("name") //nolint:gosec // 上限は MaxBodySize で設定済み
    ...
}
```

### `internal/routes/business.go` でのミドルウェア適用

```go
r.Group(func(r chi.Router) {
    r.Use(requireWrite)
    r.Use(appMiddleware.MaxBodySize(1 << 20))  // 1 MB
    r.Get("/new", projectHandler.NewProjectPage)
    r.Post("/new", projectHandler.CreateProject)
    ...
})

r.With(appMiddleware.MaxBodySize(6 << 20)).Post("/admin/users/import", importHandler.ExecuteImport)
```

注: 後で `2026-05-05-limits-package.md` でこの数値を `internal/limits` パッケージに集約。

### Excel インポート (multipart) の対処

```go
if err := r.ParseMultipartForm(6 << 20); err != nil {
    var maxBytesErr *http.MaxBytesError
    if errors.As(err, &maxBytesErr) {
        httpError(w, r, http.StatusRequestEntityTooLarge, "ファイルサイズが大きすぎます")
        return
    }
    httpError(w, r, http.StatusBadRequest, "リクエストの解析に失敗しました")
    return
}
```

## TDD アプローチ

実装前に **「壊れてる証拠」のテストを書いて赤確認** → 実装で緑化する流れ:

1. `body_size_test.go` で 4 ケース定義（通常サイズ ✓ / 超過 → 413 期待）
2. 実装前に `go test` で赤確認 (`303` や `200` が返り、`413` ではない)
3. 実装を入れる
4. 緑確認

これは派生プロジェクトでも有用。先にテストを置くと、Claude が自分で「今ここで赤、これで緑になった」と検証できる。

## 派生プロジェクトへの適用

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-05-02-maxbody-protection.md を参照して、
このプロジェクトの POST/PUT エンドポイントに MaxBodySize ミドルウェアを入れ、
G120 警告を解消してください。

業務ドメインに合わせて以下を調整してください:
- 通常フォームの上限: 1 MB
- ファイルアップロードがあるなら個別に上限指定
```

## 検証

- `make lint` で G120 警告が消えている
- `body_size_test.go` (派生では業務ドメインに合わせて書き換え) が緑
- 通常サイズで 303 or 200、超過で 413

## 関連コミット

- `ca2a2f3` body サイズ制限のテストを追加（G120 / TDD 赤確認）
- `4d8e9fa` body サイズ制限のミドルウェアを追加し、超過時に 413 を返す
