# 2026-05-01: Echo から chi/v5 への移行 (integration テスト)

## Why

本体は既に Echo → chi に移行済みだったが、`internal/integration/` パッケージが Echo のまま取り残されていた:

- `testhelper.go` が `*echo.Echo` を `chi.Router` 引数に渡そうとしてコンパイル不能
- `mlConfig.AllowLogin` の旧シグネチャ `func(c echo.Context, ...)` (現行は `*http.Request`)
- `ml.RegisterHandlers(e)` という存在しない API への参照

→ `make vet` も `make lint` も即赤になる状態で、CI を入れた瞬間に発覚した。

## What

修正対象:
- `internal/integration/testhelper.go` (主要、書き直し)
- `internal/integration/admin_crud_test.go` (admin CRUD は SSE 化されていたので全面書き直し)
- `internal/integration/admin_user_import_test.go` (echo 依存削除 + 文言更新)
- `internal/integration/permission_test.go` (admin 系を SSE ルートに差し替え + `bodyContentType` 導入)
- `internal/integration/login_http_bench_test.go` (Echo → chi への書き換え)
- `internal/integration/login_bench_test.go` は Echo 非依存 (DB ベンチ) で修正不要

加えて副作用で `go.mod` から不要依存が消える:
- `github.com/labstack/echo/v4`
- `github.com/labstack/gommon`
- `github.com/mattn/go-colorable`
- `github.com/valyala/fasttemplate`

## How

### `testhelper.go` の核心

旧 (Echo):
```go
e := echo.New()
e.HTTPErrorHandler = handlers.CustomHTTPErrorHandler  // ← 既に削除されたシンボル
routes.RegisterBusinessRoutes(e, queries, authMW)     // ← *echo.Echo を渡してた
```

新 (chi):
```go
r := chi.NewRouter()
r.Use(testUserContextMiddleware(queries))
authMW := testRequireAuth("/auth/login")
routes.RegisterBusinessRoutes(r, queries, authMW)
routes.RegisterAdminRoutes(r, queries, authMW)
registerTestSSERoutes(r, queries, authMW)  // ← magiclink 不要の SSE ルートだけ手動登録
return r  // http.Handler として返す
```

テスト用ミドルウェアも `func(http.Handler) http.Handler` シグネチャに変更:

```go
func testUserContextMiddleware(queries *database.Queries) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userIDStr := r.Header.Get("X-Test-User-ID")
            if userIDStr != "" {
                var userID int64
                if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err == nil {
                    if u, err := queries.GetUserByID(r.Context(), userID); err == nil {
                        ctx := appcontext.WithUser(r.Context(), u.Email, true, false, u.Role, u.ID)
                        r = r.WithContext(ctx)
                    }
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### `admin_crud_test.go` の書き直し（SSE 化）

旧: `POST /admin/users/new` 等の HTML フォームパス（既に廃止）
新: `POST /api/sse/admin/users/create` 等の Datastar SSE エンドポイントを JSON シグナル形式で叩く

```go
rec := DoSSERequest(e, http.MethodPost, "/api/sse/admin/users/create",
    &seed.AdminUser,
    `{"newName":"NewUser","newEmail":"newuser@test.com","newRole":"editor"}`)

if rec.Code != http.StatusOK {  // SSE はリダイレクトせず 200 で返す
    t.Fatalf("...")
}
```

期待ステータスは:
- 成功: `200`
- バリデーション/自己削除: `400`
- 権限なし: `403`
- 未認証: `303`

### `permission_test.go` の `bodyContentType` 導入

ルート別に Form / JSON を切り替えるための型を追加:

```go
type bodyContentType int

const (
    bodyForm bodyContentType = iota
    bodyJSON
)

type routeTestCase struct {
    Name         string
    Method       string
    Path         string
    Body         string
    BodyType     bodyContentType  // ← 新規
    AdminStatus  int
    EditorStatus int
    ViewerStatus int
    UnauthStatus int
}
```

## 派生プロジェクトへの適用

派生プロジェクトが Echo ベースのままなら、まず本体側を chi に移行してから integration を追従させる。

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-05-01-echo-to-chi.md を参照して、
このプロジェクトの integration テストを chi 化してください。
本体側のルーティングが既に chi なら、testhelper.go の書き直しと
各テストの URL/JSON ボディ更新だけで済みます。
```

## 検証

- `go vet ./...` が緑
- `go test ./...` が緑
- 不要依存削除後の `go mod tidy` で `go.sum` 差分なし

## 関連コミット

- `fc27ab1` integration テストを Echo から chi に移行し、admin CRUD を SSE 化
