# 2026-06-02: AccessLog を UserContextMiddleware の内側に移し user_id 欠落を修正

## Why

アクセスログ (`access.log`) の `user_id` が **常に空** になっていた。原因はミドルウェアの登録順:

```go
// 修正前 (cmd/server/main.go)
r.Use(appMiddleware.AccessLogMiddleware(logger.AccessWriter())) // 外側
r.Use(chiMiddleware.Recoverer)
r.Use(appMiddleware.UserContextMiddleware(ml, conn))            // 内側で ctx にユーザーを詰める
```

`AccessLogMiddleware` が `UserContextMiddleware` より **外側** にあると、ユーザー情報を拾えない。理由は `http.Request` がイミュータブルだから:

- `UserContextMiddleware` は `r = r.WithContext(ctx)` で **新しい Request** を作って内側へ渡す
- 外側にいる `AccessLogMiddleware` が握っている `r` は **元のまま**（ユーザー情報なしの ctx）
- 結果、`appcontext.GetUser(r.Context())` が空を返し、`user_id` が空でログされる

実害: アクセスログから「誰のリクエストか」が一切分からず、監査・障害調査で使い物にならない。

## What

変更ファイル:
- `cmd/server/main.go` (ミドルウェア登録順)

新規ファイル:
- `internal/middleware/access_log_test.go` (順序を固定する回帰テスト)

## How

### ミドルウェア順序の修正

```go
// 修正後 (cmd/server/main.go)
r := chi.NewRouter()
r.Use(chiMiddleware.Recoverer)
r.Use(appMiddleware.UserContextMiddleware(ml, conn))
// AccessLogMiddleware は UserContextMiddleware より内側に置く必要がある。
// http.Request は immutable で r.WithContext(...) は新しい Request を返すため、
// UserContextMiddleware より外側に置くと AccessLog 側が見る r.Context() に
// userEmail が反映されず、user_id が空のままログ出力されてしまう。
r.Use(appMiddleware.AccessLogMiddleware(logger.AccessWriter()))
```

ポイント: `ctx` を **読む** ミドルウェアは、`ctx` を **書く** ミドルウェアより内側に置く。

### 回帰テスト

「正しい順序なら user_id が乗る」「逆順なら空になる」の両方を仕様として固定する。順序を逆に戻すと `TestAccessLog_RecordsUserID_WhenAfterUserContext` が落ちる。

```go
func TestAccessLog_RecordsUserID_WhenAfterUserContext(t *testing.T) {
	var buf bytes.Buffer
	r := chi.NewRouter()
	r.Use(withTestUser("alice@example.invalid")) // ctx に詰める役
	r.Use(AccessLogMiddleware(&buf))             // ログに書く役は内側
	r.Get("/x", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// ... 出力 JSON の user_id == "alice@example.invalid" を検証
}
```

(全文は `internal/middleware/access_log_test.go` を参照)

## 派生プロジェクトへの適用

古い派生も `AccessLogMiddleware` を持っているなら **同じバグが残っている可能性が高い**。`cmd/server/main.go` のミドルウェア登録順を確認し、`AccessLog` が `UserContext` より内側にあるかを見る。

派生プロジェクトの Claude Code に投げるプロンプト例:

```
テンプレリポの docs/migrations/2026-06-02-accesslog-order-fix.md を参照して、
このプロジェクトの cmd/server/main.go のミドルウェア登録順を確認し、
AccessLog が user_id を取りこぼすバグがあれば修正して回帰テストを追加してください。
```

## 検証

- `go test ./internal/middleware/ -run TestAccessLog` 緑
- `make vet` / `make lint` 緑

## 関連コミット

- `330ab2b` AccessLog を UserContextMiddleware の内側に移動し user_id 欠落を修正
