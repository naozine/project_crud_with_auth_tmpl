// Package integration はエンドツーエンドの統合テスト基盤を提供する。
// インメモリ SQLite + chi ルート登録により、magiclink に依存せず
// ハンドラの動作と権限マトリクスを検証できる。
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/handlers"
	appMiddleware "github.com/naozine/project_crud_with_auth_tmpl/internal/middleware"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/routes"
	"github.com/pressly/goose/v3"

	"github.com/naozine/project_crud_with_auth_tmpl/db"
	_ "modernc.org/sqlite"
)

// SeedData はテスト用の初期データを保持する
type SeedData struct {
	AdminUser     database.User
	EditorUser    database.User
	ViewerUser    database.User
	DeletableUser database.User // 削除テスト用の使い捨てユーザー
	Project       database.Project
}

// SetupTestDB はインメモリ SQLite を作成し、マイグレーションを適用する
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", "file::memory:?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("DB接続に失敗: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	goose.SetBaseFS(db.MigrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose dialect設定に失敗: %v", err)
	}
	if err := goose.Up(conn, "migrations"); err != nil {
		t.Fatalf("マイグレーションに失敗: %v", err)
	}
	return conn
}

// SeedTestData はテスト用のユーザーとプロジェクトを作成する
func SeedTestData(t *testing.T, conn *sql.DB) SeedData {
	t.Helper()
	q := database.New(conn)
	ctx := context.Background()

	adminUser, err := q.CreateUser(ctx, database.CreateUserParams{
		Email: "admin@test.com", Name: "Admin", Role: "admin", IsActive: true,
	})
	if err != nil {
		t.Fatalf("adminユーザー作成に失敗: %v", err)
	}

	editorUser, err := q.CreateUser(ctx, database.CreateUserParams{
		Email: "editor@test.com", Name: "Editor", Role: "editor", IsActive: true,
	})
	if err != nil {
		t.Fatalf("editorユーザー作成に失敗: %v", err)
	}

	viewerUser, err := q.CreateUser(ctx, database.CreateUserParams{
		Email: "viewer@test.com", Name: "Viewer", Role: "viewer", IsActive: true,
	})
	if err != nil {
		t.Fatalf("viewerユーザー作成に失敗: %v", err)
	}

	deletableUser, err := q.CreateUser(ctx, database.CreateUserParams{
		Email: "deletable@test.com", Name: "Deletable", Role: "viewer", IsActive: true,
	})
	if err != nil {
		t.Fatalf("削除用ユーザー作成に失敗: %v", err)
	}

	project, err := q.CreateProject(ctx, "テストプロジェクト")
	if err != nil {
		t.Fatalf("プロジェクト作成に失敗: %v", err)
	}

	return SeedData{
		AdminUser:     adminUser,
		EditorUser:    editorUser,
		ViewerUser:    viewerUser,
		DeletableUser: deletableUser,
		Project:       project,
	}
}

// testRequireAuth は magiclink を使わず appcontext からログイン状態を判定するテスト用ミドルウェア。
// 本番の middleware.RequireAuth と異なり、redirect クエリパラメータは付与しない（テストの期待値を単純化するため）。
func testRequireAuth(loginURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, isLoggedIn, _ := appcontext.GetUser(r.Context())
			if !isLoggedIn {
				http.Redirect(w, r, loginURL, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// testUserContextMiddleware は X-Test-User-ID ヘッダからユーザー情報を復元するテスト用ミドルウェア
func testUserContextMiddleware(queries *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userIDStr := r.Header.Get("X-Test-User-ID")
			if userIDStr != "" {
				var userID int64
				if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err == nil {
					u, err := queries.GetUserByID(r.Context(), userID)
					if err == nil {
						ctx := appcontext.WithUser(r.Context(), u.Email, true, false, u.Role, u.ID)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SetupTestServer は統合テスト用の chi ルーターを作成し、ルートを登録する。
// 本番と同じ routes.Register* を使い、認証ミドルウェアのみテスト用に差し替える。
// SSE 系のうち magiclink を必要としないルート（Project/Admin Users）は手動で登録する。
func SetupTestServer(t *testing.T, conn *sql.DB) http.Handler {
	t.Helper()
	queries := database.New(conn)

	r := chi.NewRouter()
	r.Use(testUserContextMiddleware(queries))

	authMW := testRequireAuth("/auth/login")
	routes.RegisterBusinessRoutes(r, queries, authMW)
	routes.RegisterAdminRoutes(r, queries, authMW)
	registerTestSSERoutes(r, queries, authMW)

	return r
}

// registerTestSSERoutes は magiclink に依存しない SSE ルートのみを登録する。
// 本番の routes.RegisterSSERoutes は magiclink を要求するため、テストでは独自に組む。
func registerTestSSERoutes(r chi.Router, queries *database.Queries, authMW func(http.Handler) http.Handler) {
	projectSSE := handlers.NewProjectSSEHandler(queries)
	adminSSE := handlers.NewAdminSSEHandler(queries)

	requireWrite := appMiddleware.RequireRole("admin", "editor")
	requireAdmin := appMiddleware.RequireRole("admin")

	r.Route("/api/sse", func(r chi.Router) {
		r.Use(authMW)

		r.Group(func(r chi.Router) {
			r.Use(requireWrite)
			r.Post("/projects/new", projectSSE.CreateProjectSSE)
			r.Put("/projects/{id}", projectSSE.UpdateProjectSSE)
			r.Delete("/projects/{id}", projectSSE.DeleteProjectSSE)
		})

		r.Group(func(r chi.Router) {
			r.Use(requireAdmin)
			r.Post("/admin/users/create", adminSSE.CreateUserDialogSSE)
			r.Get("/admin/users/{id}/edit", adminSSE.EditUserDialogSSE)
			r.Put("/admin/users/{id}", adminSSE.UpdateUserSSE)
			r.Delete("/admin/users/{id}", adminSSE.DeleteUserSSE)
		})
	})
}

// DoRequest は指定ロールで HTTP リクエストを実行し、レスポンスを返す。
// user が nil の場合は未認証リクエストとなる。
// body が指定された場合は application/x-www-form-urlencoded として送る。
func DoRequest(h http.Handler, method, path string, user *database.User, body ...string) *httptest.ResponseRecorder {
	var req *http.Request
	if len(body) > 0 && body[0] != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body[0]))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	if user != nil {
		req.Header.Set("X-Test-User-ID", fmt.Sprintf("%d", user.ID))
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// DoSSERequest は Datastar SSE エンドポイントへ JSON シグナルを送る。
// 成功時は 200 + SSE ストリーム、エラー時は 4xx/5xx + プレーンテキストが返る。
// jsonBody が空の場合は body 無しで送る（DELETE 用）。
func DoSSERequest(h http.Handler, method, path string, user *database.User, jsonBody string) *httptest.ResponseRecorder {
	var req *http.Request
	if jsonBody != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	if user != nil {
		req.Header.Set("X-Test-User-ID", fmt.Sprintf("%d", user.ID))
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// queryFromConn は sql.DB から Queries を生成するヘルパー
func queryFromConn(conn *sql.DB) *database.Queries {
	return database.New(conn)
}

// sprintf は fmt.Sprintf のエイリアス（テストコードの簡略化用）
var sprintf = fmt.Sprintf
