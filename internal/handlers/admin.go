package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/logger"
	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

type AdminHandler struct {
	Queries *database.Queries
}

func NewAdminHandler(queries *database.Queries) *AdminHandler {
	return &AdminHandler{Queries: queries}
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.Queries.ListUsers(c.Request().Context())
	if err != nil {
		logger.Error("ユーザー一覧の取得に失敗", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー一覧の取得に失敗しました")
	}
	return renderShell(c, "ユーザー管理", components.AdminUserList(users))
}
