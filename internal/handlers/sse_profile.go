package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
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

// UpdateProfileSSE はプロフィールを更新し、ページをリロードする。
func (h *ProfileSSEHandler) UpdateProfileSSE(c echo.Context) error {
	ctx := c.Request().Context()
	email, _, _ := appcontext.GetUser(ctx)

	currentUser, err := h.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません")
	}

	var signals struct {
		ProfileName string `json:"profileName"`
	}
	if err := datastar.ReadSignals(c.Request(), &signals); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "無効なリクエストです")
	}

	if signals.ProfileName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "名前は必須です")
	}

	if _, err := h.Queries.UpdateUser(ctx, database.UpdateUserParams{
		Name:     signals.ProfileName,
		Role:     currentUser.Role,
		IsActive: currentUser.IsActive,
		ID:       currentUser.ID,
	}); err != nil {
		logger.Error("プロフィール更新に失敗", "error", err, "email", email)
		return echo.NewHTTPError(http.StatusInternalServerError, "プロフィールの更新に失敗しました")
	}

	sse := newSSE(c)
	return sse.ExecuteScript("window.location.reload()")
}

// DeletePasskeysSSE は全パスキーを削除し、ページをリロードする。
func (h *ProfileSSEHandler) DeletePasskeysSSE(c echo.Context) error {
	ctx := c.Request().Context()
	email, _, _ := appcontext.GetUser(ctx)

	creds, err := h.ML.DB.GetPasskeyCredentialsByUserID(email)
	if err != nil {
		logger.Error("パスキーの取得に失敗", "error", err, "email", email)
		return echo.NewHTTPError(http.StatusInternalServerError, "パスキーの取得に失敗しました")
	}

	for _, cred := range creds {
		if err := h.ML.DB.DeletePasskeyCredential(cred.ID); err != nil {
			logger.Error("パスキーの削除に失敗", "error", err, "credentialID", cred.ID)
		}
	}

	sse := newSSE(c)
	return sse.ExecuteScript("window.location.reload()")
}
