package handlers

import (
	"net/http"

	"project_crud_with_auth_tmpl/components"
	"project_crud_with_auth_tmpl/database"
	"project_crud_with_auth_tmpl/layouts"

	"github.com/labstack/echo/v4"
)

type ProjectHandler struct {
	DB *database.Queries
}

func NewProjectHandler(db *database.Queries) *ProjectHandler {
	return &ProjectHandler{DB: db}
}

func (h *ProjectHandler) ListProjects(c echo.Context) error {
	ctx := c.Request().Context()

	// 1. Get Data
	projects, err := h.DB.ListProjects(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// 2. Prepare Component
	content := components.ProjectList(projects)

	// 3. Dual-Mode Rendering
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	if c.Request().Header.Get("HX-Request") == "true" {
		return content.Render(ctx, c.Response().Writer)
	}
	return layouts.Base("プロジェクト一覧", content).Render(ctx, c.Response().Writer)
}

func (h *ProjectHandler) NewProjectPage(c echo.Context) error {
	ctx := c.Request().Context()
	content := components.ProjectForm()

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	if c.Request().Header.Get("HX-Request") == "true" {
		return content.Render(ctx, c.Response().Writer)
	}
	return layouts.Base("新規プロジェクト作成", content).Render(ctx, c.Response().Writer)
}

func (h *ProjectHandler) CreateProject(c echo.Context) error {
	ctx := c.Request().Context()

	name := c.FormValue("name")

	_, err := h.DB.CreateProject(ctx, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return c.Redirect(http.StatusSeeOther, "/projects")
}
