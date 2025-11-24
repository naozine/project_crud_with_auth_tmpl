package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/naozine/nz-magic-link/magiclink"

	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
	"github.com/naozine/project_crud_with_auth_tmpl/web/layouts"
)

type AuthHandler struct {
	ML *magiclink.MagicLink
}

func NewAuthHandler(ml *magiclink.MagicLink) *AuthHandler {
	return &AuthHandler{ML: ml}
}

func (h *AuthHandler) LoginPage(c echo.Context) error {
	_, isLoggedIn := h.ML.GetUserID(c)
	if isLoggedIn {
		return c.Redirect(http.StatusSeeOther, "/projects")
	}

	content := components.LoginForm()
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	if c.Request().Header.Get("HX-Request") == "true" {
		return content.Render(c.Request().Context(), c.Response().Writer)
	}
	return layouts.Base("ログイン", content).Render(c.Request().Context(), c.Response().Writer)
}
