package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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
	"github.com/naozine/project_crud_with_auth_tmpl/internal/loginpolicy"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/roles"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/routes"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/version"
	"github.com/naozine/project_crud_with_auth_tmpl/web"
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
	defer func() { _ = conn.Close() }()

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

	// RedirectURL / WebAuthnRedirectURL は ConfigureBusinessSettings で
	// appconfig.LandingPath を参照して設定するため、ここでは ErrorRedirectURL のみ。
	mlConfig.ErrorRedirectURL = "/auth/login"
	mlConfig.LoginSuccessMessage = "ログイン用のメールを送信しました"
	mlConfig.CookieName = generateCookieName(version.ProjectName)

	mlConfig.AllowLogin = func(r *http.Request, email string) error {
		return loginpolicy.AllowLogin(r.Context(), database.New(conn), email, r.URL.Query().Get("hp"))
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
	mlConfig.WebAuthnAllowedOrigins = []string{mlConfig.ServerAddr}

	ConfigureBusinessSettings(&mlConfig)

	ml, err := magiclink.NewWithDB(mlConfig, conn)
	if err != nil {
		log.Fatal("Failed to initialize MagicLink:", err)
	}

	// 3. Initialize Handlers
	queries := database.New(conn)
	authHandler := handlers.NewAuthHandler(queries)
	profileHandler := handlers.NewProfileHandler(queries)
	setupHandler := handlers.NewSetupHandler(queries)

	// 4. Chi Router Setup
	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(appMiddleware.SecurityHeaders)
	r.Use(appMiddleware.NoIndex)
	r.Use(appMiddleware.UserContextMiddleware(ml, conn))
	// 直近リクエストのインメモリ保持（管理画面のアクセスログビューワ用。再起動で消える）。
	accessLogStore := appMiddleware.NewAccessLogStore(1000)
	// AccessLogMiddleware は UserContextMiddleware より内側に置く必要がある。
	// http.Request は immutable で r.WithContext(...) は新しい Request を返すため、
	// UserContextMiddleware より外側に置くと AccessLog 側が見る r.Context() に
	// userEmail が反映されず、user_id が空のままログ出力されてしまう。
	r.Use(appMiddleware.AccessLogMiddleware(logger.AccessWriter(), accessLogStore))

	// Static files
	staticSubFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		log.Fatal("Failed to create sub FS for static:", err)
	}
	fileServer := http.FileServer(http.FS(staticSubFS))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// 5. Routes
	r.Get("/health", handlers.HealthCheck)

	// robots.txt: 全クローラに全パスのクロールを禁止する。検索結果に
	// 出したくない限定公開サービス向け。公開サイトにする場合は外す。
	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	})
	r.Get("/auth/login", authHandler.LoginPage)
	r.Get("/setup", setupHandler.SetupPage)
	r.Post("/setup", setupHandler.CreateInitialAdmin)

	// Datastar リファレンス/回帰確認ページ（dev 限定。本番では登録しない）。
	// docs/datastar/datastar-llm-guide.md の各機能を実機で確認できる。
	if os.Getenv("APP_ENV") == "dev" {
		r.Get("/datastar-test", handlers.DatastarReferencePage)
	}

	// Datastar レシピ集（認証不要・本番でも公開。DB 非依存のインメモリデモ）。
	// docs/datastar/datastar-llm-guide.md のお手本。バックエンドは handlers/datastar_recipes.go。
	r.Get("/datastar/recipes", handlers.DatastarRecipesPage)
	r.Post("/datastar/recipes/api/counter", handlers.RecipeCounterInc)
	r.Post("/datastar/recipes/api/todos", handlers.RecipeTodoAdd)
	r.Delete("/datastar/recipes/api/todos/{id}", handlers.RecipeTodoRemove)
	r.Get("/datastar/recipes/api/search", handlers.RecipeSearch)
	r.Get("/datastar/recipes/api/slow", handlers.RecipeSlow)
	r.Get("/datastar/recipes/api/tick", handlers.RecipeTick)
	r.Get("/datastar/recipes/api/dialog", handlers.RecipeDialog)
	r.Get("/datastar/recipes/api/vrows", handlers.RecipeVRows)
	r.Get("/datastar/recipes/api/items/{id}/edit", handlers.RecipeItemEdit)
	r.Put("/datastar/recipes/api/items/{id}", handlers.RecipeItemUpdate)
	r.Post("/datastar/recipes/api/reset", handlers.RecipeReset)

	// MagicLink handlers (net/http ベース)
	// Handler() は /auth/login, /auth/verify, /auth/logout, /webauthn/* をフルパスで登録
	mlHandler := ml.Handler()
	r.Handle("/auth/*", mlHandler)
	r.Handle("/webauthn/*", mlHandler)

	// Business & Admin Routes
	authMW := appMiddleware.RequireAuth("/auth/login")
	routes.RegisterBusinessRoutes(r, queries, authMW)
	routes.RegisterAdminRoutes(r, queries, authMW, accessLogStore)
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

	// Graceful shutdown: SIGTERM（systemd の stop/restart）/ SIGINT（Ctrl+C）を受けたら
	// リスナーを閉じて新規接続を止め、処理中のリクエストの完了を待ってから終了する。
	// 排水タイムアウトは WriteTimeout(10s) より長い 15s。それでも残る接続があれば
	// 待たずに進む（さらに固まった場合は systemd が TimeoutStopSec で SIGKILL する）。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on :%s", port)
		serverErr <- s.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		// 起動失敗（ポート使用中など）。log.Fatal は defer を実行しないが、
		// この時点ではリクエストを受けていないため閉じ漏れの実害はない。
		log.Fatal("Server error:", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received, draining requests...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown incomplete: %v", err)
		} else {
			log.Println("Server stopped gracefully")
		}
	}
	// ここで main を抜けることで、defer 済みの conn.Close() / logger.Close() が
	// 「排水完了後」に実行される（Shutdown より先に DB を閉じてはいけない）。
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
		Role:     roles.Admin,
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
