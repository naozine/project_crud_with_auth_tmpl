package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	_, isLoggedIn, _ := appcontext.GetUser(r.Context())
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
