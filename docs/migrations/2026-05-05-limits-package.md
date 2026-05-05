# 2026-05-05: body サイズ上限を internal/limits パッケージに集約

## Why

`MaxBodySize` ミドルウェア導入時 ([2026-05-02](./2026-05-02-maxbody-protection.md)) に各所で書いた `1 << 20`, `6 << 20` 等のマジックナンバーが、middleware / handler / test / route の **5 ファイルに散らばって** いた:

```
internal/routes/business.go         r.Use(MaxBodySize(1 << 20))
internal/routes/business.go         r.With(MaxBodySize(6 << 20)).Post(...)
internal/handlers/admin_user_import.go  r.ParseMultipartForm(6 << 20)
internal/integration/body_size_test.go  strings.Repeat("x", 2<<20)
internal/integration/body_size_test.go  doOversizedFileUpload(..., 7<<20)
```

実害:
- 上限を変更したい時、grep して回る必要がある
- AI に「上限を変えて」と頼んだ時、見落としリスク
- テスト値 (`2<<20` = 上限の 2 倍) が「なぜ 2MB？」と読み解きにくい

## What

新規ファイル:
- `internal/limits/limits.go`

既存ファイル変更:
- `internal/routes/business.go` (定数参照に置換)
- `internal/handlers/admin_user_import.go` (定数参照に置換)
- `internal/integration/body_size_test.go` (相対表現に置換)

## How

### `internal/limits/limits.go`

```go
// Package limits は受信リクエストのサイズ上限など、HTTP 層のリソース制約値を一元管理する。
//
// ここに集約することで、上限を変更する際に middleware / handler / test の
// 複数箇所を grep して回る必要がなくなる。
package limits

const (
    // ProjectFormBody は /projects/* の POST/PUT 受信 body 上限。
    ProjectFormBody = 1 << 20 // 1 MB

    // UserImportBody は /admin/users/import の multipart 受信 body 上限。
    // 5 MB の Excel ファイル + multipart オーバーヘッド分の余裕を見込む。
    UserImportBody = 6 << 20 // 6 MB
)
```

### 各ファイルの参照置換

```go
// routes/business.go
r.Use(appMiddleware.MaxBodySize(limits.ProjectFormBody))
r.With(appMiddleware.MaxBodySize(limits.UserImportBody)).Post(...)

// handlers/admin_user_import.go
if err := r.ParseMultipartForm(limits.UserImportBody); err != nil { ... }

// integration/body_size_test.go
body := "name=" + strings.Repeat("x", 2*limits.ProjectFormBody)
rec := doOversizedFileUpload(t, e, ..., limits.UserImportBody+(1<<20))
```

テスト値を相対表現にすることで、上限を変更した時にテストも自動追従するようになる。

## 派生プロジェクトへの適用

派生プロジェクトでは、業務ドメインに合わせて定数名を変える:

```
ProjectFormBody → TaskFormBody (例)
UserImportBody  → DocImportBody (例)
```

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-05-05-limits-package.md を参照して、
このプロジェクトの body サイズ上限値を internal/limits パッケージに集約してください。
派生プロジェクトの業務ドメインに合わせて定数名を調整してください。
```

## 検証

- `make lint` 緑
- `make test` 緑
- `grep -rn "<<\s*20\|\* 1024" --include='*.go' internal/` で `limits.go` 以外にヒットしない

## 関連コミット

- `c5221b7` body サイズ上限を internal/limits パッケージに集約
