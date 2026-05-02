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
