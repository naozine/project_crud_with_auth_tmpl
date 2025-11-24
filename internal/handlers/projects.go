package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"project_crud_with_auth_tmpl/internal/database"
	"project_crud_with_auth_tmpl/web/components"
	"project_crud_with_auth_tmpl/web/layouts"

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
	projects, err := h.DB.ListProjects(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	content := components.ProjectList(projects)
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

func (h *ProjectHandler) ShowProject(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
	}

	project, err := h.DB.GetProject(ctx, int64(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	content := components.ProjectDetail(project)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	if c.Request().Header.Get("HX-Request") == "true" {
		return content.Render(ctx, c.Response().Writer)
	}
	return layouts.Base(project.Name, content).Render(ctx, c.Response().Writer)
}

func (h *ProjectHandler) EditProjectPage(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
	}

	project, err := h.DB.GetProject(ctx, int64(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	content := components.ProjectEdit(project)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	if c.Request().Header.Get("HX-Request") == "true" {
		return content.Render(ctx, c.Response().Writer)
	}
	return layouts.Base("編集: "+project.Name, content).Render(ctx, c.Response().Writer)
}

func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
	}

	name := c.FormValue("name")
	_, err = h.DB.UpdateProject(ctx, database.UpdateProjectParams{
		Name: name,
		ID:   int64(id),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/projects/%d", id))
}

func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
	}

	err = h.DB.DeleteProject(ctx, int64(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return c.Redirect(http.StatusSeeOther, "/projects")
}
