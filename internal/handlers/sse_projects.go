package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/starfederation/datastar-go/datastar"
)

type ProjectSSEHandler struct {
	Queries *database.Queries
}

func NewProjectSSEHandler(queries *database.Queries) *ProjectSSEHandler {
	return &ProjectSSEHandler{Queries: queries}
}

// patchGrid は一覧グリッドの中身を最新の全プロジェクトで inner 置換する。
// 作成・削除で「0件 ↔ あり」の表示が正しく切り替わるよう、グリッド全体を再描画する。
func (h *ProjectSSEHandler) patchGrid(sse *datastar.ServerSentEventGenerator, r *http.Request) error {
	projects, err := h.Queries.ListProjects(r.Context())
	if err != nil {
		return err
	}
	return sse.PatchElementTempl(
		components.ProjectCards(projects, true),
		datastar.WithSelectorID("projects-grid"),
		datastar.WithModeInner(),
		datastar.WithViewTransitions(),
	)
}

func (h *ProjectSSEHandler) CreateProjectSSE(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		Name string `json:"name"`
	}
	if !readSignalsOr413(w, r, &signals) {
		return
	}
	if signals.Name == "" {
		http.Error(w, "プロジェクト名は必須です", http.StatusBadRequest)
		return
	}

	if _, err := h.Queries.CreateProject(r.Context(), signals.Name); err != nil {
		logger.Error("プロジェクト作成に失敗", "error", err)
		http.Error(w, "プロジェクトの作成に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	if err := h.patchGrid(sse, r); err != nil {
		logger.Error("SSE patchGrid failed", "error", err)
		return
	}
	sse.ExecuteScript("document.getElementById('project-add-dialog')?.close()")
	sendToast(sse, "プロジェクトを作成しました")
}

// EditProjectDialogSSE は編集ダイアログを挿入して開く（@get）。
func (h *ProjectSSEHandler) EditProjectDialogSSE(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "無効なIDです", http.StatusBadRequest)
		return
	}
	project, err := h.Queries.GetProject(r.Context(), int64(id))
	if err != nil {
		http.Error(w, "プロジェクトが見つかりません", http.StatusNotFound)
		return
	}
	sse := newSSE(w, r)
	if err := sse.PatchElementTempl(
		components.ProjectEditDialog(project),
		datastar.WithSelectorID("project-dialog-container"),
		datastar.WithModeInner(),
	); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
		return
	}
	sse.ExecuteScript("document.getElementById('project-edit-dialog')?.showModal()")
}

func (h *ProjectSSEHandler) UpdateProjectSSE(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "無効なIDです", http.StatusBadRequest)
		return
	}

	var signals struct {
		Name string `json:"name"`
	}
	if !readSignalsOr413(w, r, &signals) {
		return
	}

	if _, err := h.Queries.UpdateProject(r.Context(), database.UpdateProjectParams{
		Name: signals.Name,
		ID:   int64(id),
	}); err != nil {
		logger.Error("プロジェクト更新に失敗", "error", err, "id", id)
		http.Error(w, "プロジェクトの更新に失敗しました", http.StatusInternalServerError)
		return
	}

	project, err := h.Queries.GetProject(r.Context(), int64(id))
	if err != nil {
		http.Error(w, "プロジェクトが見つかりません", http.StatusNotFound)
		return
	}

	sse := newSSE(w, r)
	// 該当カードだけを outer 置換（reload しない）。
	if err := sse.PatchElementTempl(
		components.ProjectCard(project, true),
		datastar.WithSelectorID(fmt.Sprintf("project-%d", id)),
		datastar.WithModeOuter(),
		datastar.WithViewTransitions(),
	); err != nil {
		logger.Error("SSE PatchElementTempl failed", "error", err)
		return
	}
	sse.ExecuteScript("document.getElementById('project-edit-dialog')?.close()")
	sendToast(sse, "プロジェクトを更新しました")
}

func (h *ProjectSSEHandler) DeleteProjectSSE(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "無効なIDです", http.StatusBadRequest)
		return
	}

	if err := h.Queries.DeleteProject(r.Context(), int64(id)); err != nil {
		logger.Error("プロジェクト削除に失敗", "error", err, "id", id)
		http.Error(w, "プロジェクトの削除に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	// グリッドを再描画（最後の1件削除時に空表示へ正しく切り替わる）。
	if err := h.patchGrid(sse, r); err != nil {
		logger.Error("SSE patchGrid failed", "error", err)
	}
	sendToast(sse, "プロジェクトを削除しました")
}
