package handlers

import (
	"net/http"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

// AccessLogHandler は「最近のリクエスト」一覧を表示する（管理者用）。
type AccessLogHandler struct {
	Store *appMiddleware.AccessLogStore
}

func NewAccessLogHandler(store *appMiddleware.AccessLogStore) *AccessLogHandler {
	return &AccessLogHandler{Store: store}
}

// accessLogViewLimit は一覧に表示する最大件数。
const accessLogViewLimit = 500

// Page は直近のアクセスログを新しい順に表示する。
func (h *AccessLogHandler) Page(w http.ResponseWriter, r *http.Request) {
	entries := h.Store.Recent(accessLogViewLimit)
	renderShell(w, r, "アクセスログ", components.AccessLogsPage(entries))
}

// TableSSE は一覧コンテナ #access-logs-list を最新エントリで inner 置換する（更新ボタン用）。
func (h *AccessLogHandler) TableSSE(w http.ResponseWriter, r *http.Request) {
	entries := h.Store.Recent(accessLogViewLimit)
	sse := newSSE(w, r)
	if err := sse.PatchElementTempl(
		components.AccessLogsTable(entries),
		datastar.WithSelectorID("access-logs-list"),
		datastar.WithModeInner(),
		datastar.WithViewTransitions(),
	); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
	}
}
