package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type ProjectHandler struct {
	Queries *database.Queries
}

func NewProjectHandler(queries *database.Queries) *ProjectHandler {
	return &ProjectHandler{Queries: queries}
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.Queries.ListProjects(r.Context())
	if err != nil {
		logger.Error("プロジェクト一覧の取得に失敗", "error", err)
		httpError(w, r, http.StatusInternalServerError, "プロジェクト一覧の取得に失敗しました")
		return
	}
	renderShell(w, r, "プロジェクト一覧", components.ProjectList(projects))
}

func (h *ProjectHandler) ShowProject(w http.ResponseWriter, r *http.Request) {
	// ページハンドラのため、SSE 用の parseIDOr400 ではなく HTML エラーページを返す。
	// 型は sqlc に合わせて int64。
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "無効なIDです")
		return
	}

	project, err := h.Queries.GetProject(r.Context(), id)
	if err != nil {
		httpError(w, r, http.StatusNotFound, "プロジェクトが見つかりません")
		return
	}

	renderShell(w, r, project.Name, components.ProjectDetail(project))
}
