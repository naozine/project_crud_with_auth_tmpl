package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
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

// DeleteProjectSSE はプロジェクトを削除し、一覧を SSE で再描画する。
func (h *ProjectSSEHandler) DeleteProjectSSE(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なIDです")
	}

	if err := h.Queries.DeleteProject(ctx, int64(id)); err != nil {
		logger.Error("プロジェクト削除に失敗", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "プロジェクトの削除に失敗しました")
	}

	sse := newSSE(c)
	return sse.ExecuteScript("window.location.replace('/projects')")
}

// CreateProjectSSE はプロジェクトを作成し、一覧にリダイレクトする。
func (h *ProjectSSEHandler) CreateProjectSSE(c echo.Context) error {
	ctx := c.Request().Context()

	var signals struct {
		Name string `json:"name"`
	}
	if err := datastar.ReadSignals(c.Request(), &signals); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なリクエストです")
	}

	if signals.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "プロジェクト名は必須です")
	}

	if _, err := h.Queries.CreateProject(ctx, signals.Name); err != nil {
		logger.Error("プロジェクト作成に失敗", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "プロジェクトの作成に失敗しました")
	}

	sse := newSSE(c)
	return sse.ExecuteScript("window.location.replace('/projects')")
}

// UpdateProjectSSE はプロジェクトを更新し、詳細ページにリダイレクトする。
func (h *ProjectSSEHandler) UpdateProjectSSE(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なIDです")
	}

	var signals struct {
		Name string `json:"name"`
	}
	if err := datastar.ReadSignals(c.Request(), &signals); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なリクエストです")
	}

	if _, err := h.Queries.UpdateProject(ctx, database.UpdateProjectParams{
		Name: signals.Name,
		ID:   int64(id),
	}); err != nil {
		logger.Error("プロジェクト更新に失敗", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "プロジェクトの更新に失敗しました")
	}

	sse := newSSE(c)
	return sse.ExecuteScript(fmt.Sprintf("window.location.replace('/projects/%d')", id))
}
