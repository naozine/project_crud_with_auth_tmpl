# 2026-06-02: NoIndex ミドルウェア + robots.txt + meta robots（限定公開向け）

## Why

認証必須の限定公開サービスを、**検索結果に出したくない**。検索エンジン向けに「クロールするな／インデックスするな」を多層で伝える。

> 注意: これはアクセス制御ではなく「検索に載せない」仕組み。秘匿性は認証で担保する。**公開して検索で見つけてほしいサービスでは入れない**。

## What

新規ファイル:
- `internal/middleware/noindex_test.go`

既存ファイル変更:
- `internal/middleware/limits.go` (`NoIndex` ミドルウェア)
- `cmd/server/main.go` (`r.Use(NoIndex)` + `/robots.txt` ルート)
- `web/layouts/head.templ` (`<meta name="robots">`)

## How

3 つの層は役割が違う。`robots.txt` は「来るな（クロール禁止）」、`meta` / `X-Robots-Tag` は「載せるな（インデックス禁止）」。

### NoIndex ミドルウェア（X-Robots-Tag ヘッダ。HTML 以外にも効く）

```go
// internal/middleware/limits.go
func NoIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}
```

### main.go: ミドルウェア登録 + robots.txt

```go
r.Use(chiMiddleware.Recoverer)
r.Use(appMiddleware.NoIndex)
r.Use(appMiddleware.UserContextMiddleware(ml, conn))
// ...
r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
})
```

### head.templ: HTML への meta

```html
<meta name="robots" content="noindex, nofollow"/>
```

### 落とし穴（robots.txt と noindex の相互作用）

`robots.txt` で全 `Disallow` すると、クローラはページを取得できず **`noindex`（meta / ヘッダ）を読めない**。そのため外部からリンクされた URL が「タイトルなしの URL だけ」で稀にインデックスされることがある。

このテンプレは **ほぼ全ページが認証必須**でクローラが中身を取得できないため、両方入れる二重の保険で実害はほぼない。もし「URL すら載せたくない」を厳密に求めるなら、`robots.txt` ではブロックせず各ページの `noindex` を読ませる構成にする（Google 推奨）。

## 派生プロジェクトへの適用

- **公開サービスの派生では入れない**（検索流入が必要なため）。
- 一部だけ公開したい場合は、`NoIndex` を全体 `r.Use` ではなく該当ルートグループだけに付ける／`robots.txt` の `Disallow` を調整する。

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-06-02-noindex-robots.md を参照して、
このプロジェクト（限定公開）に検索エンジン除外（NoIndex + robots.txt + meta robots）を入れてください。
```

## 検証

- `go test ./internal/middleware/ -run TestNoIndex` 緑
- `make vet` / `make lint` 緑
- 起動後 `curl -I /` に `X-Robots-Tag: noindex, nofollow`、`curl /robots.txt` に `Disallow: /`

## 関連コミット

- `89ede15` 検索エンジン除外を追加（NoIndex ミドルウェア + robots.txt + meta robots）
