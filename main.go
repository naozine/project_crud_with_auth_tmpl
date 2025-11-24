package main

import (
	"database/sql"
	"embed"
	"log"
	"net/http"
	_ "strings"

	"project_crud_with_auth_tmpl/components"
	"project_crud_with_auth_tmpl/database"
	"project_crud_with_auth_tmpl/layouts"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed db/schema.sql
var schemaFS embed.FS

type Handler struct {
	DB *database.Queries
}

func main() {
	// 1. Database Setup
	db, err := sql.Open("sqlite3", "file:app.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Simple Migration
	if err := applySchema(db); err != nil {
		log.Fatal(err)
	}

	// 2. Initialize Handler
	h := &Handler{
		DB: database.New(db),
	}

	// 3. Echo Setup
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 4. Routes
	e.GET("/", h.HelloWorld)
	e.GET("/projects", h.ListProjects)
	e.GET("/projects/new", h.NewProjectPage)
	e.POST("/projects/new", h.CreateProject)

	// Start server
	log.Fatal(e.Start(":8080"))
}

func applySchema(db *sql.DB) error {
	schema, err := schemaFS.ReadFile("db/schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(schema))
	return err
}

func (h *Handler) HelloWorld(c echo.Context) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		return components.HelloWorld().Render(c.Request().Context(), c.Response().Writer)
	}
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return layouts.Base("Hello", components.HelloWorld()).Render(c.Request().Context(), c.Response().Writer)
}

func (h *Handler) ListProjects(c echo.Context) error {
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

func (h *Handler) NewProjectPage(c echo.Context) error {
	ctx := c.Request().Context()
	content := components.ProjectForm()

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	if c.Request().Header.Get("HX-Request") == "true" {
		return content.Render(ctx, c.Response().Writer)
	}
	return layouts.Base("新規プロジェクト作成", content).Render(ctx, c.Response().Writer)
}

func (h *Handler) CreateProject(c echo.Context) error {
	ctx := c.Request().Context()

	name := c.FormValue("name")

	_, err := h.DB.CreateProject(ctx, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return c.Redirect(http.StatusSeeOther, "/projects")
}
