package middleware

import "net/http"

// SecurityHeaders は基本的なセキュリティヘッダを全レスポンスに付ける。
//   - X-Frame-Options: DENY — iframe 埋め込みを禁止（クリックジャッキング対策。
//     セッション Cookie の SameSite=Lax と二重の防御）
//   - X-Content-Type-Options: nosniff — Content-Type を無視した型推測を禁止
//     （MIME スニッフィングによるスクリプト実行事故の対策）
//   - Referrer-Policy: strict-origin-when-cross-origin — 外部サイトへの遷移時に
//     URL のパス・クエリを Referer に載せない（トークン入り URL の漏洩対策）
//
// HSTS は TLS を終端する前段のリバースプロキシ（nz-vps-ops の Caddy）で付与する
// 方針のため、ここでは付けない。CSP は templ / Datastar の inline script と
// 衝突するため未導入（導入する場合は nonce 化が必要）。
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
