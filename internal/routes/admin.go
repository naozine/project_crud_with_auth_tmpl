package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/roles"
)

// RegisterAdminRoutes は管理者用ルートを登録する。
func RegisterAdminRoutes(r chi.Router, queries *database.Queries, authMW func(http.Handler) http.Handler, accessLogStore *appMiddleware.AccessLogStore) {
	adminHandler := handlers.NewAdminHandler(queries)
	maintenanceHandler := handlers.NewMaintenanceHandler(queries)
	accessLogHandler := handlers.NewAccessLogHandler(accessLogStore)

	r.Route("/admin", func(r chi.Router) {
		r.Use(authMW)
		r.Use(appMiddleware.RequireRole(roles.Admin))
		r.Get("/users", adminHandler.ListUsers)
		r.Get("/access-logs", accessLogHandler.Page)
		r.Get("/access-logs/table", accessLogHandler.TableSSE)
		r.Get("/maintenance", maintenanceHandler.Page)
	})
}
