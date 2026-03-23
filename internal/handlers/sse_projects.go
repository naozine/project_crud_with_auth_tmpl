package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/starfederation/datastar-go/datastar"
)

type ProjectSSEHandler struct {
	Queries *database.Queries
}

func NewProjectSSEHandler(queries *database.Queries) *ProjectSSEHandler {
	return &ProjectSSEHandler{Queries: queries}
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
	sse.ExecuteScript("window.location.replace('/projects')")
}

func (h *ProjectSSEHandler) CreateProjectSSE(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		Name string `json:"name"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "無効なリクエストです", http.StatusBadRequest)
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
	sse.ExecuteScript("window.location.replace('/projects')")
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
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "無効なリクエストです", http.StatusBadRequest)
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

	sse := newSSE(w, r)
	sse.ExecuteScript(fmt.Sprintf("window.location.replace('/projects/%d')", id))
}
