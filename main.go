package main

import (
	"log"

	"project_crud_with_auth_tmpl/components"
	"project_crud_with_auth_tmpl/layouts"
	"project_crud_with_auth_tmpl/models"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", helloWorldHandler)
	e.GET("/projects", listProjectsHandler)

	// Start server
	log.Fatal(e.Start(":8080"))
}

func helloWorldHandler(c echo.Context) error {
	// If request is from htmx, render only the component.
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		return components.HelloWorld().Render(c.Request().Context(), c.Response().Writer)
	}

	// Otherwise, render full layout
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	// For Hello World, we can use a simpler layout or just the component if no layout exists for it specifically
	// But assuming we want consistency, we might wrap it too, but for now keeping as is or wrapping in base.
	// Let's wrap it in Base for consistency.
	return layouts.Base("Hello", components.HelloWorld()).Render(c.Request().Context(), c.Response().Writer)
}

func listProjectsHandler(c echo.Context) error {
	// Mock Data
	projects := []models.Project{
		{ID: 1, Name: "Project Alpha", Description: "The first project", Status: "Active"},
		{ID: 2, Name: "Project Beta", Description: "The second project", Status: "Pending"},
		{ID: 3, Name: "Project Gamma", Description: "The third project", Status: "Completed"},
	}

	// 1. Prepare Component
	content := components.ProjectList(projects)

	// 2. Dual-Mode Rendering
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)

	if c.Request().Header.Get("HX-Request") == "true" {
		// Render only the list component (fragment)
		return content.Render(c.Request().Context(), c.Response().Writer)
	}

	// Render full layout wrapping the component
	return layouts.Base("Projects", content).Render(c.Request().Context(), c.Response().Writer)
}
