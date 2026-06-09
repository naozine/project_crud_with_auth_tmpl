package handlers

import (
	"fmt"
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/maintenance"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/starfederation/datastar-go/datastar"
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

// ToggleSSE は現在のメンテモードを反転させ、状態パネルだけを patch する。
// Datastar 経由 (POST /api/sse/admin/maintenance/toggle)。reload しない。
func (h *MaintenanceHandler) ToggleSSE(w http.ResponseWriter, r *http.Request) {
	cur := maintenance.IsEnabled(r.Context(), h.Queries)
	if err := maintenance.SetEnabled(r.Context(), h.Queries, !cur); err != nil {
		logger.Error("メンテナンスモード切替に失敗", "error", err, "next", !cur)
		http.Error(w, "切替に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	// 状態パネルだけを差し替える（reload しない）。
	if err := sse.PatchElementTempl(
		components.AdminMaintenancePanel(!cur),
		datastar.WithSelectorID("maintenance-panel"),
		datastar.WithModeOuter(),
		datastar.WithViewTransitions(),
	); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
	}
	state := "OFF"
	if !cur {
		state = "ON"
	}
	sendToast(sse, fmt.Sprintf("メンテナンスモードを%sにしました", state))
}
