package middleware

import (
	"encoding/json"
	"io"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
)

// AccessLogEntry はアクセスログの1エントリ
type AccessLogEntry struct {
	Time      string `json:"time"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	IP        string `json:"ip"`
	User      string `json:"user,omitempty"`
	UA        string `json:"ua,omitempty"`
}

// AccessLogMiddleware はJSON形式のアクセスログを出力するミドルウェア
func AccessLogMiddleware(out io.Writer) echo.MiddlewareFunc {
	encoder := json.NewEncoder(out)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// 次のハンドラを実行
			err := next(c)

			// レイテンシ計算
			latency := time.Since(start)

			// ユーザー情報を取得
			userEmail, _, _ := appcontext.GetUser(c.Request().Context())

			// ログエントリを作成
			entry := AccessLogEntry{
				Time:      start.UTC().Format(time.RFC3339),
				Method:    c.Request().Method,
				Path:      c.Request().URL.Path,
				Status:    c.Response().Status,
				LatencyMs: latency.Milliseconds(),
				IP:        c.RealIP(),
				User:      userEmail,
				UA:        c.Request().UserAgent(),
			}

			// JSON出力
			_ = encoder.Encode(entry)

			return err
		}
	}
}
