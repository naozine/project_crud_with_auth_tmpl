package main

import (
	"database/sql"
	"fmt"
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
	// Use existing SQLite connection, so DatabasePath is not used for connection but kept for config consistency
	mlConfig.DatabaseType = "sqlite"

	mlConfig.ServerAddr = os.Getenv("SERVER_ADDR")
	if mlConfig.ServerAddr == "" {
		mlConfig.ServerAddr = "http://localhost:8080"
	}
	mlConfig.DevBypassEmailFilePath = ".bypass_emails" // For development: return magic link in response
	mlConfig.RedirectURL = "/projects"                 // Redirect to projects list after login
	mlConfig.ErrorRedirectURL = "/auth/login"          // Redirect to login page on error
	mlConfig.LoginSuccessMessage = "ログイン用のメールを送信しました"

	// AllowLogin callback to check against users table
	mlConfig.AllowLogin = func(c echo.Context, email string) error {
		// We need to access the database here. Since we can't easily pass the queries object directly
		// into this config function definition before it's created, we'll create a new queries instance
		// or rely on a closure if we restructure.
		// However, `queries` is created *after* this config.
		// To fix this dependency cycle, we can defer the actual DB check or restructure initialization.
		// A simple way is to use the `conn` we already have.

		q := database.New(conn)
		user, err := q.GetUserByEmail(c.Request().Context(), email)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("このメールアドレスは登録されていません。")
			}
			log.Printf("Database error in AllowLogin: %v", err)
			return fmt.Errorf("システムエラーが発生しました。")
		}

		if !user.IsActive {
			return fmt.Errorf("このアカウントは無効化されています。")
		}

		return nil
	}

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
	mlConfig.WebAuthnRedirectURL = "/projects" // Redirect to projects list after passkey login

	allowedOrigins := os.Getenv("WEBAUTHN_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		mlConfig.WebAuthnAllowedOrigins = []string{allowedOrigins}
	} else {
		mlConfig.WebAuthnAllowedOrigins = []string{"http://localhost:8080"}
	}

	// Initialize MagicLink with existing DB connection
	ml, err := magiclink.NewWithDB(mlConfig, conn)
	if err != nil {
		log.Fatal("Failed to initialize MagicLink:", err)
	}

	// 3. Initialize Handlers
	queries := database.New(conn)
	projectHandler := handlers.NewProjectHandler(queries)
	authHandler := handlers.NewAuthHandler(ml)
	adminHandler := handlers.NewAdminHandler(queries)
	profileHandler := handlers.NewProfileHandler(queries, ml)

	// 4. Echo Setup
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(appMiddleware.UserContextMiddleware(ml, conn)) // Add UserContext middleware globally

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

	// Admin Routes
	adminGroup := e.Group("/admin")
	adminGroup.Use(ml.AuthMiddleware()) // Require login
	// In a real app, we'd add a RequireRole("admin") middleware here too,
	// but the handler checks it internally for now.

	adminGroup.GET("/users", adminHandler.ListUsers)
	adminGroup.GET("/users/new", adminHandler.NewUserPage)
	adminGroup.POST("/users", adminHandler.CreateUser)
	adminGroup.GET("/users/:id/edit", adminHandler.EditUserPage)
	adminGroup.POST("/users/:id", adminHandler.UpdateUser)
	adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)

	// Profile Routes
	e.GET("/profile", profileHandler.ShowProfile, ml.AuthMiddleware())
	e.POST("/profile", profileHandler.UpdateProfile, ml.AuthMiddleware())
	e.DELETE("/profile/passkeys", profileHandler.DeletePasskeys, ml.AuthMiddleware())

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
