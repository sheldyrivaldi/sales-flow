package auth

import (
	"strings"

	"github.com/labstack/echo/v4"

	"salespilot/internal/http/httperr"
)

// JWTMiddleware verifies the Bearer token in the Authorization header,
// then stores AuthUser in the echo context for downstream handlers.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return httperr.Write(c, httperr.NewUnauthorized("token tidak ditemukan atau format salah"))
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := Parse(token, secret, TokenAccess)
			if err != nil {
				return httperr.Write(c, httperr.NewUnauthorized("token tidak valid atau sudah kedaluwarsa"))
			}

			SetUser(c, AuthUser{
				ID:   claims.Subject,
				Role: claims.Role,
			})
			return next(c)
		}
	}
}
