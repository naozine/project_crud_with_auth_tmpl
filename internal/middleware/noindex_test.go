package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NoIndex が X-Robots-Tag: noindex, nofollow を全レスポンスに付け、
// かつ後続ハンドラをそのまま呼ぶこと。
func TestNoIndex_SetsXRobotsTag(t *testing.T) {
	called := false
	h := NoIndex(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next ハンドラが呼ばれていない")
	}
	if got := rec.Header().Get("X-Robots-Tag"); got != "noindex, nofollow" {
		t.Errorf("X-Robots-Tag = %q, want %q", got, "noindex, nofollow")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
