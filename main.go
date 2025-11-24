package main

import (
	"log"

	"project_crud_with_auth_tmpl/components" // Add this line

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

	// Start server
	log.Fatal(e.Start(":8080"))
}

func helloWorldHandler(c echo.Context) error {
	// If request is from htmx, render only the component.
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		return components.HelloWorld().Render(c.Request().Context(), c.Response().Writer)
	}

	// Otherwise, render full layout (for now, just the component directly)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return components.HelloWorld().Render(c.Request().Context(), c.Response().Writer)
}
