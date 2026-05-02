package handlers

import (
	"fmt"
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

func (h *ProjectHandler) NewProjectPage(w http.ResponseWriter, r *http.Request) {
	renderShell(w, r, "新規プロジェクト作成", components.ProjectForm())
}

func (h *ProjectHandler) ShowProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "無効なIDです")
		return
	}

	project, err := h.Queries.GetProject(r.Context(), int64(id))
	if err != nil {
		httpError(w, r, http.StatusNotFound, "プロジェクトが見つかりません")
		return
	}

	renderShell(w, r, project.Name, components.ProjectDetail(project))
}

func (h *ProjectHandler) EditProjectPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "無効なIDです")
		return
	}

	project, err := h.Queries.GetProject(r.Context(), int64(id))
	if err != nil {
		httpError(w, r, http.StatusNotFound, "プロジェクトが見つかりません")
		return
	}

	renderShell(w, r, "編集: "+project.Name, components.ProjectEdit(project))
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if parseFormOr413(w, r) {
		return
	}
	name := r.FormValue("name") //nolint:gosec // body 上限は MaxBodySize ミドルウェアで設定済み
	if _, err := h.Queries.CreateProject(r.Context(), name); err != nil {
		logger.Error("プロジェクト作成に失敗", "error", err)
		httpError(w, r, http.StatusInternalServerError, "プロジェクトの作成に失敗しました")
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "無効なIDです")
		return
	}

	if parseFormOr413(w, r) {
		return
	}
	name := r.FormValue("name") //nolint:gosec // body 上限は MaxBodySize ミドルウェアで設定済み
	if _, err := h.Queries.UpdateProject(r.Context(), database.UpdateProjectParams{
		Name: name,
		ID:   int64(id),
	}); err != nil {
		logger.Error("プロジェクト更新に失敗", "error", err, "id", id)
		httpError(w, r, http.StatusInternalServerError, "プロジェクトの更新に失敗しました")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", id), http.StatusSeeOther)
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, r, http.StatusBadRequest, "無効なIDです")
		return
	}

	if err := h.Queries.DeleteProject(r.Context(), int64(id)); err != nil {
		logger.Error("プロジェクト削除に失敗", "error", err, "id", id)
		httpError(w, r, http.StatusInternalServerError, "プロジェクトの削除に失敗しました")
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}
