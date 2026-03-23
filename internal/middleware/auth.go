package middleware

import (
	"database/sql"
	"net/http"

	"github.com/naozine/nz-magic-link/magiclink"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
)

func UserContextMiddleware(ml *magiclink.MagicLink, dbConn *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userEmail, isLoggedIn := ml.ValidateSession(r)

			var hasPasskey bool
			var role string
			var userID int64

			if isLoggedIn {
				creds, err := ml.DB.GetPasskeyCredentialsByUserID(userEmail)
				if err == nil && len(creds) > 0 {
					hasPasskey = true
				}

				q := database.New(dbConn)
				user, err := q.GetUserByEmail(r.Context(), userEmail)
				if err == nil {
					role = user.Role
					userID = user.ID
				}
			}

			ctx := appcontext.WithUser(r.Context(), userEmail, isLoggedIn, hasPasskey, role, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole は指定されたロールのいずれかを持つユーザーのみアクセスを許可するミドルウェア。
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := appcontext.GetUserRole(r.Context())
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "アクセス権限がありません", http.StatusForbidden)
		})
	}
}

// RequireAuth は認証を必須とするミドルウェア。
// 未認証の場合はログインページへリダイレクトする。
func RequireAuth(loginURL string) func(http.Handler) http.Handler {
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
