package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
)

// RegisterSSERoutes は Datastar SSE 用のルートを登録する。
func RegisterSSERoutes(e *echo.Echo, queries *database.Queries, authMW echo.MiddlewareFunc) {
	projectSSE := handlers.NewProjectSSEHandler(queries)

	sseGroup := e.Group("/api/sse")
	sseGroup.Use(authMW)

	requireWrite := appMiddleware.RequireRole("admin", "editor")

	// Projects
	sseGroup.POST("/projects/new", projectSSE.CreateProjectSSE, requireWrite)
	sseGroup.PUT("/projects/:id", projectSSE.UpdateProjectSSE, requireWrite)
	sseGroup.DELETE("/projects/:id", projectSSE.DeleteProjectSSE, requireWrite)
}
