package domain

import "testing"

func TestSourceAccess_Valid(t *testing.T) {
	tests := []struct {
		access SourceAccess
		want   bool
	}{
		{SourceAccessPublik, true},
		{SourceAccessLogin, true},
		{SourceAccessManual, true},
		{"INVALID", false},
		{"", false},
		{"PUBLIK", false},
	}
	for _, tt := range tests {
		if got := tt.access.Valid(); got != tt.want {
			t.Errorf("SourceAccess(%q).Valid() = %v, want %v", tt.access, got, tt.want)
		}
	}
}
