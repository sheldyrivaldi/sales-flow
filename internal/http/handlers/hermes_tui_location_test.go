package handlers

import "testing"

func TestPrefixTuiLocation(t *testing.T) {
	const base = "/api/admin/hermes/tui"
	cases := []struct{ name, in, want string }{
		{"redirect login root-relative diprefiks", "/auth/login?provider=basic&next=%2F", base + "/auth/login?provider=basic&next=%2F"},
		{"root diprefiks", "/", base + "/"},
		{"sudah berprefix dibiarkan", base + "/dashboard", base + "/dashboard"},
		{"base persis dibiarkan", base, base},
		{"absolut http dibiarkan", "http://lain/auth/login", "http://lain/auth/login"},
		{"protocol-relative dibiarkan", "//lain/x", "//lain/x"},
		{"kosong dibiarkan", "", ""},
		{"non-root (relatif) dibiarkan", "next-page", "next-page"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := prefixTuiLocation(c.in, base); got != c.want {
				t.Errorf("prefixTuiLocation(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
