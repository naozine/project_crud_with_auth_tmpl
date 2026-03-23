package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/naozine/project_crud_with_auth_tmpl/web/layouts"
)

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	errorPage := components.ErrorPage(code, message)
	page := layouts.Base("エラー", errorPage)
	page.Render(r.Context(), w)
}
