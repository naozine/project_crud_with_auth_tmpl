package auth

import (
	"project_crud_with_auth_tmpl/internal/appcontext"

	"github.com/labstack/echo/v4"
	"github.com/naozine/nz-magic-link/magiclink"
)

func UserContextMiddleware(ml *magiclink.MagicLink) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userEmail, isLoggedIn := ml.GetUserID(c)

			// Set user info to request context
			ctx := c.Request().Context()
			ctx = appcontext.WithUser(ctx, userEmail, isLoggedIn)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
