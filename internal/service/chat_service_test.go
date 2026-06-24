package service

import "testing"

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"halo dunia ini pesan pertama dari user", "halo dunia ini pesan pertama dari"},
		{"satu", "satu"},
		{"satu dua", "satu dua"},
		{"a b c d e f", "a b c d e f"},
		{"a b c d e f g h", "a b c d e f"},
		{"", "Percakapan baru"},
		{"   ", "Percakapan baru"},
	}
	for _, tt := range tests {
		got := deriveTitle(tt.input)
		if got != tt.want {
			t.Errorf("deriveTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
