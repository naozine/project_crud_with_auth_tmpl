package middleware

import "net/http"

// NoIndex は X-Robots-Tag ヘッダを全レスポンスに付け、検索エンジンに
// インデックス・追跡をさせないようにする。社内向け・限定公開など、
// 検索結果に出したくないサービス向け。HTML 側の <meta name="robots"> と
// 二重に効かせる。公開サイトにする場合はこのミドルウェアと meta を外す。
func NoIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		next.ServeHTTP(w, r)
	})
}

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
