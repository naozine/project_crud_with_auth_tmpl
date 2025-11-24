package main

import (
	"database/sql"
	"log"
	"net/http"

	"project_crud_with_auth_tmpl/db"
	"project_crud_with_auth_tmpl/internal/database"
	"project_crud_with_auth_tmpl/internal/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 1. Database Setup
	conn, err := sql.Open("sqlite3", "file:app.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Simple Migration
	if err := applySchema(conn); err != nil {
		log.Fatal(err)
	}

	// 2. Initialize Handlers
	queries := database.New(conn)
	projectHandler := handlers.NewProjectHandler(queries)

	// 3. Echo Setup
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 4. Routes
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/projects")
	})
	e.GET("/projects", projectHandler.ListProjects)
	e.GET("/projects/new", projectHandler.NewProjectPage)
	e.POST("/projects/new", projectHandler.CreateProject)
	e.GET("/projects/:id", projectHandler.ShowProject)
	e.GET("/projects/:id/edit", projectHandler.EditProjectPage)
	e.POST("/projects/:id/update", projectHandler.UpdateProject)
	e.POST("/projects/:id/delete", projectHandler.DeleteProject)

	// Start server
	log.Fatal(e.Start(":8080"))
}

func applySchema(conn *sql.DB) error {
	schema, err := db.SchemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(schema))
	return err
}
