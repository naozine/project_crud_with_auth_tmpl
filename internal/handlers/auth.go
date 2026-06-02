package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/maintenance"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type AuthHandler struct {
	Queries *database.Queries
}

func NewAuthHandler(queries *database.Queries) *AuthHandler {
	return &AuthHandler{Queries: queries}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	_, isLoggedIn, _ := appcontext.GetUser(r.Context())

	// メンテナンス中は一般ログインのフォームを描画せず、案内のみ表示する。
	// 既にログイン中のユーザーはセッションが維持されるのでそのままホームへ通す。
	if maintenance.IsEnabled(r.Context(), h.Queries) {
		if isLoggedIn {
			http.Redirect(w, r, "/projects", http.StatusSeeOther)
			return
		}
		renderGuest(w, r, "メンテナンス中", components.LoginMaintenance())
		return
	}

	if isLoggedIn {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	errorMessage := r.URL.Query().Get("error_description")
	if r.URL.Query().Get("error") == "token_used" {
		errorMessage = "このログインリンクは既に使用されています。もう一度メールアドレスを入力して新しいリンクを取得してください。"
	} else if errorMessage == "token has expired" {
		errorMessage = "ログインリンクの有効期限が切れています。もう一度お試しください。"
	} else if errorMessage == "invalid token" {
		errorMessage = "無効なログインリンクです。"
	}

	renderGuest(w, r, "ログイン", components.LoginForm(errorMessage))
}
