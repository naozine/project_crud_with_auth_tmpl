package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		userRole       string
		allowedRoles   []string
		wantStatusCode int
	}{
		{"admin は admin 専用ルートにアクセスできる", "admin", []string{"admin"}, http.StatusOK},
		{"editor は admin 専用ルートにアクセスできない", "editor", []string{"admin"}, http.StatusForbidden},
		{"viewer は admin 専用ルートにアクセスできない", "viewer", []string{"admin"}, http.StatusForbidden},
		{"admin は書き込みルートにアクセスできる", "admin", []string{"admin", "editor"}, http.StatusOK},
		{"editor は書き込みルートにアクセスできる", "editor", []string{"admin", "editor"}, http.StatusOK},
		{"viewer は書き込みルートにアクセスできない", "viewer", []string{"admin", "editor"}, http.StatusForbidden},
		{"ロール未設定はアクセスできない", "", []string{"admin"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := appcontext.WithUser(req.Context(), "test@example.com", true, false, tt.userRole, 1)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireRole(tt.allowedRoles...)(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("ステータスコード = %d, want %d", rec.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name         string
		isLoggedIn   bool
		requestPath  string
		wantCode     int
		wantRedirect string
	}{
		{
			name:         "認証済みはそのまま通過",
			isLoggedIn:   true,
			requestPath:  "/projects",
			wantCode:     http.StatusOK,
			wantRedirect: "",
		},
		{
			name:         "未認証はログインページにリダイレクト",
			isLoggedIn:   false,
			requestPath:  "/projects",
			wantCode:     http.StatusSeeOther,
			wantRedirect: "/auth/login?redirect=%2Fprojects",
		},
		{
			name:         "未認証時に元の URL パスとクエリが redirect に含まれる",
			isLoggedIn:   false,
			requestPath:  "/projects/5?tab=details",
			wantCode:     http.StatusSeeOther,
			wantRedirect: "/auth/login?redirect=%2Fprojects%2F5%3Ftab%3Ddetails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)

			email := ""
			if tt.isLoggedIn {
				email = "test@example.com"
			}
			ctx := appcontext.WithUser(req.Context(), email, tt.isLoggedIn, false, "admin", 1)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireAuth("/auth/login")(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("ステータスコード = %d, want %d", rec.Code, tt.wantCode)
			}

			if tt.wantRedirect != "" {
				got := rec.Header().Get("Location")
				if got != tt.wantRedirect {
					t.Errorf("リダイレクト先 = %q, want %q", got, tt.wantRedirect)
				}
			}
		})
	}
}
