package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/naozine/project_crud_with_auth_tmpl/db"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/naozine/nz-magic-link/magiclink"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not be loaded, using OS environment variables.")
	}

	// 1. Database Setup (for projects)
	conn, err := sql.Open("sqlite3", "file:app.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal("Failed to connect to app.db:", err)
	}
	defer conn.Close()

	// Simple Migration
	if err := applySchema(conn); err != nil {
		log.Fatal("Failed to apply app schema:", err)
	}

	// 2. MagicLink Setup
	mlConfig := magiclink.DefaultConfig()
	mlConfig.DatabasePath = "magiclink.db" // Uses a separate database for magic links
	mlConfig.DatabaseType = "leveldb"
	mlConfig.DatabaseOptions = map[string]string{
		"block_cache_capacity": "33554432", // 32MB
		"write_buffer":         "16777216", // 16MB
	}

	mlConfig.ServerAddr = os.Getenv("SERVER_ADDR")
	if mlConfig.ServerAddr == "" {
		mlConfig.ServerAddr = "http://localhost:8080"
	}
	mlConfig.DevBypassEmailFilePath = ".bypass_emails" // For development: return magic link in response
	mlConfig.RedirectURL = "/projects"                 // Redirect to projects list after login

	// Configure SMTP
	mlConfig.SMTPHost = os.Getenv("SMTP_HOST")
	mlConfig.SMTPPort = mustAtoi(os.Getenv("SMTP_PORT"), 587)
	mlConfig.SMTPUsername = os.Getenv("SMTP_USERNAME")
	mlConfig.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	mlConfig.SMTPFrom = os.Getenv("SMTP_FROM")
	mlConfig.SMTPFromName = os.Getenv("SMTP_FROM_NAME")

	// WebAuthn Configuration
	mlConfig.WebAuthnEnabled = true
	mlConfig.WebAuthnRPID = os.Getenv("WEBAUTHN_RP_ID")
	if mlConfig.WebAuthnRPID == "" {
		mlConfig.WebAuthnRPID = "localhost"
	}
	mlConfig.WebAuthnRPName = os.Getenv("WEBAUTHN_RP_NAME")
	if mlConfig.WebAuthnRPName == "" {
		mlConfig.WebAuthnRPName = "Project CRUD"
	}
	allowedOrigins := os.Getenv("WEBAUTHN_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		mlConfig.WebAuthnAllowedOrigins = []string{allowedOrigins}
	} else {
		mlConfig.WebAuthnAllowedOrigins = []string{"http://localhost:8080"}
	}

	ml, err := magiclink.New(mlConfig)
	if err != nil {
		log.Fatal("Failed to initialize MagicLink:", err)
	}

	// 3. Initialize Handlers
	queries := database.New(conn)
	projectHandler := handlers.NewProjectHandler(queries)
	authHandler := handlers.NewAuthHandler(ml)

	// 4. Echo Setup
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(appMiddleware.UserContextMiddleware(ml)) // Add UserContext middleware globally

	e.Static("/static", "web/static")

	// 5. Routes
	// Public Routes
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/auth/login")
	})
	e.GET("/auth/login", authHandler.LoginPage)

	// MagicLink internal handlers
	ml.RegisterHandlers(e)

	// Protected Routes
	// All project routes require authentication
	projectGroup := e.Group("/projects")
	projectGroup.Use(ml.AuthMiddleware()) // Apply auth middleware to this group

	projectGroup.GET("", projectHandler.ListProjects)
	projectGroup.GET("/new", projectHandler.NewProjectPage)
	projectGroup.POST("/new", projectHandler.CreateProject)
	projectGroup.GET("/:id", projectHandler.ShowProject)
	projectGroup.GET("/:id/edit", projectHandler.EditProjectPage)
	projectGroup.POST("/:id/update", projectHandler.UpdateProject)
	projectGroup.POST("/:id/delete", projectHandler.DeleteProject)

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

func mustAtoi(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Warning: Could not parse %q as int, using default %d. Error: %v", s, defaultValue, err)
		return defaultValue
	}
	return i
}
