package routes

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/limits"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/roles"
)

// RegisterBusinessRoutes はビジネスロジックのルートを登録する。
// db はトランザクションを使うハンドラ（一括インポート等）に渡す。
func RegisterBusinessRoutes(r chi.Router, db *sql.DB, queries *database.Queries, authMW func(http.Handler) http.Handler) {
	projectHandler := handlers.NewProjectHandler(queries)
	importHandler := handlers.NewUserImportHandler(db, queries)

	requireAdmin := appMiddleware.RequireRole(roles.Admin)

	// プロジェクトの作成・編集・削除は Datastar SSE（/api/sse/projects/*）で行う。
	// 通常ルートは一覧・詳細の表示のみ。
	r.Route("/projects", func(r chi.Router) {
		r.Use(authMW)
		r.Get("/", projectHandler.ListProjects)
		r.Get("/{id}", projectHandler.ShowProject)
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
