package handlers

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/naozine/project_crud_with_auth_tmpl/web/layouts"
)

// renderShell は認証済みページをシェルレイアウトでラップして描画する。
func renderShell(c echo.Context, title string, content templ.Component) error {
	ctx := c.Request().Context()
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	currentPath := c.Request().URL.Path
	return layouts.Shell(title, currentPath, content).Render(ctx, c.Response().Writer)
}

// renderGuest はゲスト（未認証）ページを描画する。
func renderGuest(c echo.Context, title string, content templ.Component) error {
	ctx := c.Request().Context()
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return layouts.Guest(title, content).Render(ctx, c.Response().Writer)
}
