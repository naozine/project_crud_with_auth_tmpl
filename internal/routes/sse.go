package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/nz-magic-link/magiclink"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/roles"
)

// RegisterSSERoutes は Datastar SSE 用のルートを登録する。
func RegisterSSERoutes(r chi.Router, queries *database.Queries, ml *magiclink.MagicLink, authMW func(http.Handler) http.Handler) {
	projectSSE := handlers.NewProjectSSEHandler(queries)
	adminSSE := handlers.NewAdminSSEHandler(queries)
	maintenanceHandler := handlers.NewMaintenanceHandler(queries)
	profileSSE := handlers.NewProfileSSEHandler(queries, ml)

	requireWrite := appMiddleware.RequireRole(roles.Admin, roles.Editor)
	requireAdmin := appMiddleware.RequireRole(roles.Admin)

	r.Route("/api/sse", func(r chi.Router) {
		r.Use(authMW)

		// Projects
		r.Group(func(r chi.Router) {
			r.Use(requireWrite)
			r.Post("/projects/new", projectSSE.CreateProjectSSE)
			r.Get("/projects/{id}/edit", projectSSE.EditProjectDialogSSE)
			r.Put("/projects/{id}", projectSSE.UpdateProjectSSE)
			r.Delete("/projects/{id}", projectSSE.DeleteProjectSSE)
		})

		// Admin Users
		r.Group(func(r chi.Router) {
			r.Use(requireAdmin)
			r.Post("/admin/users/create", adminSSE.CreateUserDialogSSE)
			r.Get("/admin/users/{id}/edit", adminSSE.EditUserDialogSSE)
			r.Put("/admin/users/{id}", adminSSE.UpdateUserSSE)
			r.Delete("/admin/users/{id}", adminSSE.DeleteUserSSE)

			r.Post("/admin/maintenance/toggle", maintenanceHandler.ToggleSSE)
		})

		// Profile
		r.Put("/profile", profileSSE.UpdateProfileSSE)
		r.Delete("/profile/passkeys", profileSSE.DeletePasskeysSSE)
	})
}
