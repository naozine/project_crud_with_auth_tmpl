package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/naozine/nz-magic-link/magiclink"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/naozine/project_crud_with_auth_tmpl/db"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/routes"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/version"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not be loaded, using OS environment variables.")
	}

	// Initialize Logger
	logConfig := logger.DefaultConfig()
	logConfig.LogDir = os.Getenv("LOG_DIR")
	if err := logger.Init(logConfig); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Close()

	// 1. Database Setup
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}
	conn, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(30000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		log.Fatal("Failed to connect to app.db:", err)
	}
	defer conn.Close()

	// Run Migrations
	goose.SetBaseFS(db.MigrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal("Failed to set goose dialect:", err)
	}
	if err := goose.Up(conn, "migrations"); err != nil {
		log.Fatal("Failed to apply migrations:", err)
	}

	// Initialize Admin User if needed
	if err := ensureAdminUser(conn); err != nil {
		log.Printf("Warning: Failed to initialize admin user: %v", err)
	}

	// 2. MagicLink Setup
	mlConfig := magiclink.DefaultConfig()
	mlConfig.DatabaseType = "sqlite"
	mlConfig.ServerAddr = resolveServerAddr()

	if os.Getenv("APP_ENV") == "dev" && strings.HasPrefix(mlConfig.ServerAddr, "http://") {
		mlConfig.CookieSecure = false
	}

	for _, path := range []string{".bypass_emails", "data/.bypass_emails"} {
		if _, err := os.Stat(path); err == nil {
			mlConfig.DevBypassEmailFilePath = path
			break
		}
	}

	mlConfig.RedirectURL = "/projects"
	mlConfig.ErrorRedirectURL = "/auth/login"
	mlConfig.LoginSuccessMessage = "ログイン用のメールを送信しました"
	mlConfig.CookieName = generateCookieName(version.ProjectName)

	mlConfig.AllowLogin = func(r *http.Request, email string) error {
		q := database.New(conn)
		user, err := q.GetUserByEmail(r.Context(), email)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.Warn("Login attempt with unregistered email", "email", email)
				return nil
			}
			logger.Error("Database error in AllowLogin", "error", err, "email", email)
			return fmt.Errorf("システムエラーが発生しました。")
		}
		if !user.IsActive {
			logger.Warn("Login attempt with inactive account", "email", email)
			return nil
		}
		return nil
	}

	// SMTP
	mlConfig.SMTPHost = os.Getenv("SMTP_HOST")
	mlConfig.SMTPPort = mustAtoi(os.Getenv("SMTP_PORT"), 587)
	mlConfig.SMTPUsername = os.Getenv("SMTP_USERNAME")
	mlConfig.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	mlConfig.SMTPFrom = os.Getenv("SMTP_FROM")
	mlConfig.SMTPFromName = os.Getenv("SMTP_FROM_NAME")

	// WebAuthn
	mlConfig.WebAuthnEnabled = true
	mlConfig.WebAuthnRPID = extractHost(mlConfig.ServerAddr)
	mlConfig.WebAuthnRPName = os.Getenv("WEBAUTHN_RP_NAME")
	if mlConfig.WebAuthnRPName == "" {
		mlConfig.WebAuthnRPName = "Project CRUD"
	}
	mlConfig.WebAuthnRedirectURL = "/projects"
	mlConfig.WebAuthnAllowedOrigins = []string{mlConfig.ServerAddr}

	ConfigureBusinessSettings(&mlConfig)

	ml, err := magiclink.NewWithDB(mlConfig, conn)
	if err != nil {
		log.Fatal("Failed to initialize MagicLink:", err)
	}

	// 3. Initialize Handlers
	queries := database.New(conn)
	authHandler := handlers.NewAuthHandler()
	profileHandler := handlers.NewProfileHandler(queries)
	setupHandler := handlers.NewSetupHandler(queries)

	// 4. Chi Router Setup
	r := chi.NewRouter()
	r.Use(appMiddleware.AccessLogMiddleware(logger.AccessWriter()))
	r.Use(chiMiddleware.Recoverer)
	r.Use(appMiddleware.UserContextMiddleware(ml, conn))

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fs))

	// 5. Routes
	r.Get("/health", handlers.HealthCheck)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	})
	r.Get("/auth/login", authHandler.LoginPage)
	r.Get("/setup", setupHandler.SetupPage)
	r.Post("/setup", setupHandler.CreateInitialAdmin)

	// MagicLink handlers (net/http ベース)
	mlHandler := ml.Handler()
	r.Mount("/auth", mlHandler)
	r.Mount("/webauthn", mlHandler)

	// Business & Admin Routes
	authMW := appMiddleware.RequireAuth("/auth/login")
	routes.RegisterBusinessRoutes(r, queries, authMW)
	routes.RegisterAdminRoutes(r, queries, authMW)
	routes.RegisterSSERoutes(r, queries, ml, authMW)

	// Profile Routes
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Get("/profile", profileHandler.ShowProfile)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("Starting server on :%s", port)
	log.Fatal(s.ListenAndServe())
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

func ensureAdminUser(conn *sql.DB) error {
	var count int
	err := conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		return nil
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		log.Println("No users found and ADMIN_EMAIL not set. Skipping admin creation.")
		return nil
	}

	adminName := os.Getenv("ADMIN_NAME")
	if adminName == "" {
		adminName = "Admin"
	}

	log.Printf("Creating initial admin user: %s (%s)", adminName, adminEmail)

	q := database.New(conn)
	_, err = q.CreateUser(context.Background(), database.CreateUserParams{
		Email:    adminEmail,
		Name:     adminName,
		Role:     "admin",
		IsActive: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Println("Initial admin user created successfully.")
	return nil
}

func generateCookieName(projectName string) string {
	name := strings.ToLower(projectName)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		name = "app"
	}
	return name + "_session"
}

func resolveServerAddr() string {
	if serverAddr := os.Getenv("SERVER_ADDR"); serverAddr != "" {
		return serverAddr
	}
	if os.Getenv("APP_ENV") == "dev" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		return "http://localhost:" + port
	}
	return version.ServerAddr
}

func extractHost(serverAddr string) string {
	u, err := url.Parse(serverAddr)
	if err != nil || u.Host == "" {
		return "localhost"
	}
	host := u.Hostname()
	if host == "" {
		return "localhost"
	}
	return host
}
