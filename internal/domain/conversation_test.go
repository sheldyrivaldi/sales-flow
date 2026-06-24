package domain

import "testing"

func TestMessageRole_Valid(t *testing.T) {
	tests := []struct {
		role MessageRole
		want bool
	}{
		{RoleUser, true},
		{RoleAssistant, true},
		{RoleSystem, true},
		{RoleTool, true},
		{"x", false},
		{"", false},
		{"USER", false},
	}
	for _, tt := range tests {
		if got := tt.role.Valid(); got != tt.want {
			t.Errorf("MessageRole(%q).Valid() = %v, want %v", tt.role, got, tt.want)
		}
	}
}
