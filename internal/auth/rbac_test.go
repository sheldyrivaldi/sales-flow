package auth

import (
	"testing"

	"salespilot/internal/domain"
)

func TestCan_AllCapabilitiesAllRoles(t *testing.T) {
	tests := []struct {
		cap  Capability
		role domain.Role
		want bool
	}{
		// ViewData — semua role bisa
		{CapViewData, domain.RoleSales, true},
		{CapViewData, domain.RoleOps, true},
		{CapViewData, domain.RoleManager, true},
		{CapViewData, domain.RoleAdmin, true},

		// CRUDData — semua role bisa
		{CapCRUDData, domain.RoleSales, true},
		{CapCRUDData, domain.RoleOps, true},
		{CapCRUDData, domain.RoleManager, true},
		{CapCRUDData, domain.RoleAdmin, true},

		// EditProfile — OPS, MANAGER, ADMIN; bukan SALES
		{CapEditProfile, domain.RoleSales, false},
		{CapEditProfile, domain.RoleOps, true},
		{CapEditProfile, domain.RoleManager, true},
		{CapEditProfile, domain.RoleAdmin, true},

		// RunDiscovery — OPS, MANAGER, ADMIN; bukan SALES
		{CapRunDiscovery, domain.RoleSales, false},
		{CapRunDiscovery, domain.RoleOps, true},
		{CapRunDiscovery, domain.RoleManager, true},
		{CapRunDiscovery, domain.RoleAdmin, true},

		// UseAI — semua role bisa
		{CapUseAI, domain.RoleSales, true},
		{CapUseAI, domain.RoleOps, true},
		{CapUseAI, domain.RoleManager, true},
		{CapUseAI, domain.RoleAdmin, true},

		// MakeDecision — semua role bisa (ownership enforcement ada di service layer)
		{CapMakeDecision, domain.RoleSales, true},
		{CapMakeDecision, domain.RoleOps, true},
		{CapMakeDecision, domain.RoleManager, true},
		{CapMakeDecision, domain.RoleAdmin, true},

		// ManageUsers — hanya ADMIN
		{CapManageUsers, domain.RoleSales, false},
		{CapManageUsers, domain.RoleOps, false},
		{CapManageUsers, domain.RoleManager, false},
		{CapManageUsers, domain.RoleAdmin, true},

		// ViewUsers — semua role bisa (read-only, untuk resolve nama owner)
		{CapViewUsers, domain.RoleSales, true},
		{CapViewUsers, domain.RoleOps, true},
		{CapViewUsers, domain.RoleManager, true},
		{CapViewUsers, domain.RoleAdmin, true},
	}

	for _, tt := range tests {
		got := Can(tt.role, tt.cap)
		if got != tt.want {
			t.Errorf("Can(%q, %q) = %v; want %v", tt.role, tt.cap, got, tt.want)
		}
	}
}
