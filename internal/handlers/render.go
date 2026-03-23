package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/naozine/project_crud_with_auth_tmpl/web/layouts"
)

// renderShell は認証済みページをシェルレイアウトでラップして描画する。
func renderShell(w http.ResponseWriter, r *http.Request, title string, content templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	currentPath := r.URL.Path
	layouts.Shell(title, currentPath, content).Render(r.Context(), w)
}

// renderGuest はゲスト（未認証）ページを描画する。
func renderGuest(w http.ResponseWriter, r *http.Request, title string, content templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	layouts.Guest(title, content).Render(r.Context(), w)
}
