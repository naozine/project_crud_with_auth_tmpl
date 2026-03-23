package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type AdminHandler struct {
	Queries *database.Queries
}

func NewAdminHandler(queries *database.Queries) *AdminHandler {
	return &AdminHandler{Queries: queries}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Queries.ListUsers(r.Context())
	if err != nil {
		logger.Error("ユーザー一覧の取得に失敗", "error", err)
		httpError(w, r, http.StatusInternalServerError, "ユーザー一覧の取得に失敗しました")
		return
	}
	renderShell(w, r, "ユーザー管理", components.AdminUserList(users))
}
