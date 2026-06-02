# 2026-06-02: ロール定数を internal/roles に一元管理し、ベタ書きを禁止する

## Why

ロール文字列 ("admin" / "editor" / "viewer") がハンドラ・ルート・templ の各所にベタ書きされていた（プロダクションコードで約20箇所）。

実害・リスク:
- **typo が静かに権限を壊す**: `"editor"` を `"editer"` と書いてもコンパイルは通り、権限チェックだけが効かなくなる
- **検証ロジックの重複**: 「3ロールのどれか」チェックが `admin_user_import.go` と `sse_admin.go` に別々に存在
- **追加時の漏れ**: ロールを増やすとき修正箇所を grep で追う必要

テンプレは「手本」なので、今後追加する機能がこのパターンを真似てしまう。定数化に加えて、**再発を防ぐガードレール**を入れるのが本マイグレーションの主眼。

## What

新規:
- `internal/roles/roles.go`（定数 + `All` + `IsValid()`）
- Makefile `check-roles` ターゲット（`make check` に組込み）
- CLAUDE.md「コード規約」セクション

変更（ベタ書き → 定数）:
- routes: business / sse / admin の `RequireRole`
- handlers: admin_user_import / setup / sse_admin、cmd/server/main.go
- templ: project_detail / project_list / ui_page / ui_form / shell

テストコードは**対象外**。JSON ペイロード（`{"newRole":"editor"}`）や期待値は具体値の方が読みやすく、権限が静かに壊れるリスクもないため。

## How

### internal/roles/roles.go
```go
package roles

const (
    Admin  = "admin"
    Editor = "editor"
    Viewer = "viewer"
)

var All = []string{Viewer, Editor, Admin}

func IsValid(r string) bool {
    switch r {
    case Admin, Editor, Viewer:
        return true
    default:
        return false
    }
}
```

### 置換例
```go
// routes
appMiddleware.RequireRole(roles.Admin, roles.Editor)

// 検証（重複していたロジックを集約）
if !roles.IsValid(role) { ... }
```
```templ
// templ（比較・セレクト）。属性値にも式を使える
if userRole == roles.Admin { ... }
<option value={ roles.Viewer }>Viewer（閲覧のみ）</option>
```

### 再発防止ガードレール（ここが肝）
1. **`make check-roles`**: roles パッケージ・生成物・テスト以外でロール文字列リテラルが出たら fail。`make check` に組込み。
   ```makefile
   check-roles:
       @if grep -rn '"admin"\|"editor"\|"viewer"' internal/ web/ cmd/ --include="*.go" --include="*.templ" \
           | grep -v '_templ.go' | grep -v '_test.go' \
           | grep -v 'internal/roles/' | grep -v 'internal/integration/'; then \
           echo "ERROR: ロール文字列はベタ書きせず internal/roles の定数を使ってください"; \
           exit 1; \
       fi
   ```
2. **CLAUDE.md のコード規約**: 「ロールは roles 定数を使う」を明記。Claude 中心開発なので **AI が手本に従う**仕組みとして効く。

> 定数化だけでは「今あるコードがきれいになる」だけ。grep チェック + CLAUDE.md を足して「今後ベタ書きしたら**機械と AI の両方が止める**」状態にするのが目的。

## 派生プロジェクトへの適用

派生で独自ロール（例: "owner"）がある場合は `internal/roles` に足し、`check-roles` の grep パターンも派生のロールに合わせる。

プロンプト例:
```
テンプレリポの docs/migrations/2026-06-02-role-constants.md を参照して、
このプロジェクトのロール文字列を internal/roles 定数に統一し、
make check-roles ガードレールと CLAUDE.md ルールを追加してください。
```

## 検証
- `make generate && make vet && make lint` 緑
- `make check-roles` 緑（ベタ書きなし）
- `go test ./...` 緑
