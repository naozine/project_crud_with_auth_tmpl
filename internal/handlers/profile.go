package handlers

import (
	"net/http"

	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type ProfileHandler struct {
	Queries *database.Queries
}

func NewProfileHandler(queries *database.Queries) *ProfileHandler {
	return &ProfileHandler{Queries: queries}
}

func (h *ProfileHandler) ShowProfile(w http.ResponseWriter, r *http.Request) {
	email, _, hasPasskey := appcontext.GetUser(r.Context())

	user, err := h.Queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		httpError(w, r, http.StatusNotFound, "ユーザーが見つかりません")
		return
	}

	renderShell(w, r, "マイページ", components.Profile(user, hasPasskey))
}
