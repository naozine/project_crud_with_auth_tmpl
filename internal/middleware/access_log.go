package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
)

// AccessLogEntry はアクセスログの1エントリ
type AccessLogEntry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`

	TraceID    string `json:"trace_id,omitempty"`
	CFRay      string `json:"cf_ray,omitempty"`
	UpstreamID string `json:"upstream_id,omitempty"`

	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	IP        string `json:"ip"`
	UA        string `json:"ua,omitempty"`

	UserID string `json:"user_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

func getLogLevel(status int) string {
	if status >= 500 {
		return "ERROR"
	}
	if status >= 400 {
		return "WARN"
	}
	return "INFO"
}

func getTraceID(cfRay, upstreamID string) string {
	if cfRay != "" {
		return cfRay
	}
	return upstreamID
}

// statusResponseWriter はステータスコードをキャプチャする ResponseWriter ラッパー。
// http.Flusher も委譲して SSE (Datastar) と互換性を保つ。
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap は元の ResponseWriter を返す（http.ResponseController 対応）。
func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// AccessLogMiddleware はJSON形式のアクセスログを出力するミドルウェア
func AccessLogMiddleware(out io.Writer) func(http.Handler) http.Handler {
	encoder := json.NewEncoder(out)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			sw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			latency := time.Since(start)
			userEmail, _, _ := appcontext.GetUser(r.Context())
			cfRay := r.Header.Get("CF-Ray")
			upstreamID := r.Header.Get("X-Request-ID")

			entry := AccessLogEntry{
				Time:       start.UTC().Format(time.RFC3339),
				Level:      getLogLevel(sw.status),
				Msg:        "http_request",
				TraceID:    getTraceID(cfRay, upstreamID),
				CFRay:      cfRay,
				UpstreamID: upstreamID,
				Method:     r.Method,
				Path:       r.URL.Path,
				Status:     sw.status,
				LatencyMs:  latency.Milliseconds(),
				IP:         r.RemoteAddr,
				UA:         r.UserAgent(),
				UserID:     userEmail,
			}

			_ = encoder.Encode(entry)
		})
	}
}
