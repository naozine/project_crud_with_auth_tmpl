package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/maintenance"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type MaintenanceHandler struct {
	Queries *database.Queries
}

func NewMaintenanceHandler(q *database.Queries) *MaintenanceHandler {
	return &MaintenanceHandler{Queries: q}
}

// Page はメンテナンスモードの状態と切替ボタンを表示する。admin 限定。
func (h *MaintenanceHandler) Page(w http.ResponseWriter, r *http.Request) {
	enabled := maintenance.IsEnabled(r.Context(), h.Queries)
	renderShell(w, r, "メンテナンスモード", components.AdminMaintenance(enabled))
}

// ToggleSSE は現在のメンテモードを反転させて、画面リロードを指示する。
// Datastar 経由 (POST /api/sse/admin/maintenance/toggle)。
func (h *MaintenanceHandler) ToggleSSE(w http.ResponseWriter, r *http.Request) {
	cur := maintenance.IsEnabled(r.Context(), h.Queries)
	if err := maintenance.SetEnabled(r.Context(), h.Queries, !cur); err != nil {
		logger.Error("メンテナンスモード切替に失敗", "error", err, "next", !cur)
		http.Error(w, "切替に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	sse.ExecuteScript("window.location.reload()")
}
