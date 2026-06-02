package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// magiclink ライブラリ既定の IP rate limit は同一 IP で 1 分 10 リクエスト。
// 本番は ConfigureBusinessSettings (cmd/server/routes_business.go)、テストは
// setupHTTPBench が、いずれも mlConfig.DisableRateLimiting = true を設定している。
// その「無効化が効いている」ことを検証する回帰テスト。無効化を外すと 11 件目
// 以降で 429 が返り、本テストが落ちる。
//
// 同一 IP から 12 回連続でログインリクエストを投げて、429 が一度も返らない
// （= 11 件目以降も通る）ことを確認する。
func TestRateLimitDisabled_AllowsBurstFromSameIP(t *testing.T) {
	h, _, _ := setupHTTPBench(t, 1)

	const burst = 12
	const remoteAddr = "192.0.2.10:12345"
	email := "user0@test.com" // setupHTTPBench が作る最初のユーザー

	for i := 0; i < burst; i++ {
		body := fmt.Sprintf(`{"email":"%s"}`, email)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("リクエスト #%d で 429 が返った（rate limit が有効になっている）。"+
				"DisableRateLimiting を確認すること。 body=%s", i+1, rec.Body.String())
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("リクエスト #%d: status %d, body=%s", i+1, rec.Code, rec.Body.String())
		}
	}
}

// 同一 IP から異なるメアドで連投しても 429 が返らないこと。
// 「会場 Wi-Fi で大勢が同時にログイン」を簡易再現するシナリオ。
func TestRateLimitDisabled_AllowsBurstAcrossUsers(t *testing.T) {
	const numUsers = 15
	h, _, _ := setupHTTPBench(t, numUsers)

	const remoteAddr = "192.0.2.20:12345"

	for i := 0; i < numUsers; i++ {
		email := fmt.Sprintf("user%d@test.com", i)
		body := fmt.Sprintf(`{"email":"%s"}`, email)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("リクエスト #%d (%s) で 429。同一 IP からの異メアド連投が止められている", i+1, email)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("リクエスト #%d (%s): status %d, body=%s", i+1, email, rec.Code, rec.Body.String())
		}
	}
}
