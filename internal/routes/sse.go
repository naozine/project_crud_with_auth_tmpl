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
	adminSSE := handlers.NewAdminSSEHandler(queries)

	sseGroup := e.Group("/api/sse")
	sseGroup.Use(authMW)

	requireWrite := appMiddleware.RequireRole("admin", "editor")
	requireAdmin := appMiddleware.RequireRole("admin")

	// Projects
	sseGroup.POST("/projects/new", projectSSE.CreateProjectSSE, requireWrite)
	sseGroup.PUT("/projects/:id", projectSSE.UpdateProjectSSE, requireWrite)
	sseGroup.DELETE("/projects/:id", projectSSE.DeleteProjectSSE, requireWrite)

	// Admin Users
	sseGroup.POST("/admin/users/create", adminSSE.CreateUserDialogSSE, requireAdmin)
	sseGroup.GET("/admin/users/:id/edit", adminSSE.EditUserDialogSSE, requireAdmin)
	sseGroup.PUT("/admin/users/:id", adminSSE.UpdateUserSSE, requireAdmin)
	sseGroup.DELETE("/admin/users/:id", adminSSE.DeleteUserSSE, requireAdmin)
}
