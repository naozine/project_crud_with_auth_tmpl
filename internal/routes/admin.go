package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
)

// RegisterAdminRoutes は管理者用ルートを登録する。
func RegisterAdminRoutes(r chi.Router, queries *database.Queries, authMW func(http.Handler) http.Handler) {
	adminHandler := handlers.NewAdminHandler(queries)

	r.Route("/admin", func(r chi.Router) {
		r.Use(authMW)
		r.Use(appMiddleware.RequireRole("admin"))
		r.Get("/users", adminHandler.ListUsers)
	})
}
