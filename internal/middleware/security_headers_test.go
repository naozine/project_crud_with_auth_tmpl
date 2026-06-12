package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// SecurityHeaders が各セキュリティヘッダを全レスポンスに付け、
// かつ後続ハンドラをそのまま呼ぶこと。
func TestSecurityHeaders_SetsHeaders(t *testing.T) {
	called := false
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next ハンドラが呼ばれていない")
	}
	want := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
