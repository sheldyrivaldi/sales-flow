package auth

import (
	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

type Capability string

const (
	CapViewData      Capability = "ViewData"
	CapCRUDData      Capability = "CRUDData"
	CapEditProfile   Capability = "EditProfile"
	CapRunDiscovery  Capability = "RunDiscovery"
	CapUseAI         Capability = "UseAI"
	CapMakeDecision  Capability = "MakeDecision"
	CapManageUsers   Capability = "ManageUsers"
)

// capabilityRoles encodes the permission matrix from PRD §3.1.
var capabilityRoles = map[Capability][]domain.Role{
	CapViewData:     {domain.RoleSales, domain.RoleOps, domain.RoleManager, domain.RoleAdmin},
	CapCRUDData:     {domain.RoleSales, domain.RoleOps, domain.RoleManager, domain.RoleAdmin},
	CapEditProfile:  {domain.RoleOps, domain.RoleManager, domain.RoleAdmin},
	CapRunDiscovery: {domain.RoleOps, domain.RoleManager, domain.RoleAdmin},
	CapUseAI:        {domain.RoleSales, domain.RoleOps, domain.RoleManager, domain.RoleAdmin},
	CapMakeDecision: {domain.RoleSales, domain.RoleOps, domain.RoleManager, domain.RoleAdmin},
	CapManageUsers:  {domain.RoleAdmin},
}

// Can reports whether role has the given capability.
func Can(role domain.Role, cap Capability) bool {
	for _, r := range capabilityRoles[cap] {
		if r == role {
			return true
		}
	}
	return false
}

// RequireCapability is an Echo middleware that denies access when the
// authenticated user lacks the given capability.
// Must be used after JWTMiddleware (requires AuthUser in context).
func RequireCapability(cap Capability) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := UserFromContext(c)
			if !ok {
				return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
			}
			if !Can(user.Role, cap) {
				return httperr.Write(c, httperr.NewForbidden("akses ditolak: capability tidak mencukupi"))
			}
			return next(c)
		}
	}
}
