package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/limits"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
)

// RegisterBusinessRoutes はビジネスロジックのルートを登録する。
func RegisterBusinessRoutes(r chi.Router, queries *database.Queries, authMW func(http.Handler) http.Handler) {
	projectHandler := handlers.NewProjectHandler(queries)
	importHandler := handlers.NewUserImportHandler(queries)

	requireWrite := appMiddleware.RequireRole("admin", "editor")
	requireAdmin := appMiddleware.RequireRole("admin")

	r.Route("/projects", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", projectHandler.ListProjects)
		r.Get("/{id}", projectHandler.ShowProject)

		r.Group(func(r chi.Router) {
			r.Use(requireWrite)
			r.Use(appMiddleware.MaxBodySize(limits.ProjectFormBody))
			r.Get("/new", projectHandler.NewProjectPage)
			r.Post("/new", projectHandler.CreateProject)
			r.Get("/{id}/edit", projectHandler.EditProjectPage)
			r.Post("/{id}/update", projectHandler.UpdateProject)
			r.Post("/{id}/delete", projectHandler.DeleteProject)
		})
	})

	// ユーザー一括インポート（admin のみ）
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Use(requireAdmin)
		r.Get("/admin/users/import", importHandler.ImportPage)
		r.With(appMiddleware.MaxBodySize(limits.UserImportBody)).Post("/admin/users/import", importHandler.ExecuteImport)
		r.Get("/admin/users/import/template", importHandler.TemplateDownload)
	})
}
