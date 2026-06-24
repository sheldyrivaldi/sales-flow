package auth

import (
	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
)

type contextKey string

const ctxUserKey contextKey = "auth_user"

// AuthUser carries the authenticated user's identity extracted from a JWT.
type AuthUser struct {
	ID   string
	Role domain.Role
}

// SetUser stores AuthUser in the echo context (called by JWTMiddleware).
func SetUser(c echo.Context, u AuthUser) {
	c.Set(string(ctxUserKey), u)
}

// UserFromContext retrieves AuthUser from the echo context.
// Returns (AuthUser{}, false) if not set.
func UserFromContext(c echo.Context) (AuthUser, bool) {
	v := c.Get(string(ctxUserKey))
	if v == nil {
		return AuthUser{}, false
	}
	u, ok := v.(AuthUser)
	return u, ok
}
