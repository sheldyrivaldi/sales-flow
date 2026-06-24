package config

import "testing"

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/sp")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("API_SERVER_KEY", "key")
	t.Setenv("WORKSPACE_SESSION_KEY", "ws")
	t.Setenv("SALES_MCP_TOKEN", "mcp")
	t.Setenv("SEED_ADMIN_EMAIL", "admin@local")
	t.Setenv("SEED_ADMIN_PASSWORD", "pw")
}

func TestLoad_Success_AndDefaults(t *testing.T) {
	setRequired(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port default = %q, want 8080", cfg.Port)
	}
	if cfg.HermesBaseURL != "http://localhost:8642" {
		t.Errorf("HermesBaseURL default = %q", cfg.HermesBaseURL)
	}
	if cfg.DatabaseURL != "postgres://localhost/sp" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	setRequired(t)
	t.Setenv("JWT_SECRET", "") // kosongkan satu yang wajib
	if _, err := Load(); err == nil {
		t.Fatal("expected error when JWT_SECRET empty")
	}
}
