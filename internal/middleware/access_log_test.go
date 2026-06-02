package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
)

// withTestUser は context にユーザー情報を詰めるテスト用 middleware。
// UserContextMiddleware の代わりに使い、AccessLogMiddleware が
// 「先行 middleware が context に詰めたユーザー情報をログに反映できるか」だけを
// 切り出して検証する。
func withTestUser(email string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := appcontext.WithUser(r.Context(), email, true, false, "viewer", 1)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func lastLogLine(buf *bytes.Buffer) []byte {
	b := bytes.TrimRight(buf.Bytes(), "\n")
	if i := bytes.LastIndex(b, []byte("\n")); i >= 0 {
		return b[i+1:]
	}
	return b
}

// AccessLogMiddleware を「ユーザーを詰める middleware より内側」に置けば、
// 出力 JSON の user_id にメアドが乗ること。
//
// 本番 (cmd/server/main.go) で middleware を `UserContext → AccessLog` の順で
// 並べたとき、access.log の user_id がきちんと記録されることのリグレッション防止。
// この順序を逆にすると本テストは失敗する（user_id が空になる）。
func TestAccessLog_RecordsUserID_WhenAfterUserContext(t *testing.T) {
	var buf bytes.Buffer

	r := chi.NewRouter()
	r.Use(withTestUser("alice@example.invalid")) // 「ユーザー情報を context に詰める」役
	r.Use(AccessLogMiddleware(&buf))             // 「ログに書く」役は内側に置くのが正解
	r.Get("/x", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rec, req)

	var entry AccessLogEntry
	if err := json.Unmarshal(lastLogLine(&buf), &entry); err != nil {
		t.Fatalf("アクセスログのパース失敗: %v\nraw=%q", err, buf.String())
	}
	if entry.UserID != "alice@example.invalid" {
		t.Errorf("user_id = %q, want alice@example.invalid（middleware 順を逆にすると空になるバグ）", entry.UserID)
	}
	if entry.Path != "/x" {
		t.Errorf("path = %q, want /x", entry.Path)
	}
	if entry.Status != http.StatusOK {
		t.Errorf("status = %d, want %d", entry.Status, http.StatusOK)
	}
}

// 「逆順（AccessLog が外側）に並べると user_id が空になる」という性質も
// 仕様として固定しておく。順序を入れ替えても気づかないバグを防ぐ。
//
// http.Request はイミュータブルで、内側で r.WithContext(...) しても
// 外側にいる middleware が見ている r は更新されないため、AccessLog 側は
// 元の context（ユーザー情報なし）を読むことになる。
func TestAccessLog_DropsUserID_WhenBeforeUserContext(t *testing.T) {
	var buf bytes.Buffer

	r := chi.NewRouter()
	r.Use(AccessLogMiddleware(&buf))             // ← 外側に置くと拾えない
	r.Use(withTestUser("alice@example.invalid")) // ← 内側
	r.Get("/x", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(rec, req)

	var entry AccessLogEntry
	if err := json.Unmarshal(lastLogLine(&buf), &entry); err != nil {
		t.Fatalf("アクセスログのパース失敗: %v\nraw=%q", err, buf.String())
	}
	if entry.UserID != "" {
		t.Errorf("user_id = %q, 逆順なら空であるべき（性質として固定）", entry.UserID)
	}
}
