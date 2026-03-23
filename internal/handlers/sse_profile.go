package handlers

import (
	"net/http"

	"github.com/naozine/nz-magic-link/magiclink"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/appcontext"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/starfederation/datastar-go/datastar"
)

type ProfileSSEHandler struct {
	Queries *database.Queries
	ML      *magiclink.MagicLink
}

func NewProfileSSEHandler(queries *database.Queries, ml *magiclink.MagicLink) *ProfileSSEHandler {
	return &ProfileSSEHandler{Queries: queries, ML: ml}
}

func (h *ProfileSSEHandler) UpdateProfileSSE(w http.ResponseWriter, r *http.Request) {
	email, _, _ := appcontext.GetUser(r.Context())

	currentUser, err := h.Queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}

	var signals struct {
		ProfileName string `json:"profileName"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "無効なリクエストです", http.StatusBadRequest)
		return
	}

	if signals.ProfileName == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}

	if _, err := h.Queries.UpdateUser(r.Context(), database.UpdateUserParams{
		Name:     signals.ProfileName,
		Role:     currentUser.Role,
		IsActive: currentUser.IsActive,
		ID:       currentUser.ID,
	}); err != nil {
		logger.Error("プロフィール更新に失敗", "error", err, "email", email)
		http.Error(w, "プロフィールの更新に失敗しました", http.StatusInternalServerError)
		return
	}

	sse := newSSE(w, r)
	sse.ExecuteScript("window.location.reload()")
}

func (h *ProfileSSEHandler) DeletePasskeysSSE(w http.ResponseWriter, r *http.Request) {
	email, _, _ := appcontext.GetUser(r.Context())

	creds, err := h.ML.DB.GetPasskeyCredentialsByUserID(email)
	if err != nil {
		logger.Error("パスキーの取得に失敗", "error", err, "email", email)
		http.Error(w, "パスキーの取得に失敗しました", http.StatusInternalServerError)
		return
	}

	for _, cred := range creds {
		if err := h.ML.DB.DeletePasskeyCredential(cred.ID); err != nil {
			logger.Error("パスキーの削除に失敗", "error", err, "credentialID", cred.ID)
		}
	}

	sse := newSSE(w, r)
	sse.ExecuteScript("window.location.reload()")
}
