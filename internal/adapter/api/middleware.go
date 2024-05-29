package api

import (
	"github.com/burenotti/go_health_backend/internal/app/auth"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

const KeyCurrentUser = "current_user"

func LoginRequired(authorizer *auth.Authorizer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			parts := strings.Split(header, " ")
			if len(parts) != 2 {
				return JsonError(c, http.StatusUnprocessableEntity, "Invalid Authorization header")
			}
			if parts[0] != "Bearer" {
				return JsonError(c, http.StatusUnprocessableEntity, "Invalid Authorization header")
			}
			user, err := authorizer.ValidateAccessToken(parts[1])
			if err != nil {
				return JsonError(c, http.StatusUnauthorized, err.Error())
			}
			c.Set(KeyCurrentUser, user)
			if err := next(c); err != nil {
				c.Error(err)
			}
			return nil
		}
	}
}
