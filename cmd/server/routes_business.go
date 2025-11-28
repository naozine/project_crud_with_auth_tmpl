package main

import (
	"github.com/labstack/echo/v4"
	"github.com/naozine/nz-magic-link/magiclink"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appconfig"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
)

// ConfigureBusinessSettings allows customization of MagicLink config and App Name
func ConfigureBusinessSettings(config *magiclink.Config) {
	config.RedirectURL = "/projects"         // Redirect to projects list after login
	config.WebAuthnRedirectURL = "/projects" // Redirect to projects list after passkey login

	// Set Application Name
	appconfig.AppName = "プロジェクト管理"
}

// RegisterBusinessRoutes registers routes for business logic features

func RegisterBusinessRoutes(e *echo.Echo, queries *database.Queries, ml *magiclink.MagicLink) {
	// Handlers for business logic
	projectHandler := handlers.NewProjectHandler(queries) // Note: This is now business_projects.go
	// This handler name is generic, but actually points to the business logic handler.
	// This allows project_crud_with_auth_tmpl to act as a sample for business logic.

	// Protected Routes (Business Logic - projects)
	projectGroup := e.Group("/projects")
	projectGroup.Use(appMiddleware.RequireAuth(ml, "/auth/login")) // 未認証時はログインページへリダイレクト

	projectGroup.GET("", projectHandler.ListProjects)
	projectGroup.GET("/new", projectHandler.NewProjectPage)
	projectGroup.POST("/new", projectHandler.CreateProject)
	projectGroup.GET("/:id", projectHandler.ShowProject)
	projectGroup.GET("/:id/edit", projectHandler.EditProjectPage)
	projectGroup.POST("/:id/update", projectHandler.UpdateProject)
	projectGroup.POST("/:id/delete", projectHandler.DeleteProject)

	// Other business logic routes can be added here in derived projects
}
