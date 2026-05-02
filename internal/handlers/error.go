package handlers

import (
	"errors"
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

// parseFormOr413 は r.ParseForm を呼び、body 上限超過なら 413 を返して true を返す。
// 戻り値が true の場合、呼び出し元は即座に return すること。
// 上限超過でないパースエラーは 400 を返す。
func parseFormOr413(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil { //nolint:gosec // body 上限は MaxBodySize ミドルウェアで設定済み
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httpError(w, r, http.StatusRequestEntityTooLarge, "リクエストが大きすぎます")
			return true
		}
		httpError(w, r, http.StatusBadRequest, "リクエストの解析に失敗しました")
		return true
	}
	return false
}

// httpError はHTMLエラーページを返す。
func httpError(w http.ResponseWriter, r *http.Request, code int, message string) {
	if message == "" || message == "Not Found" {
		switch code {
		case http.StatusNotFound:
			message = "お探しのページは存在しません"
		case http.StatusForbidden:
			message = "このページへのアクセス権限がありません"
		case http.StatusInternalServerError:
			message = "サーバーで問題が発生しました"
		}
	}

	if code >= 500 {
		logger.Error("HTTP error",
			"status", code,
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	w.WriteHeader(code)
	renderGuest(w, r, "エラー", components.ErrorPage(code, message))
}
