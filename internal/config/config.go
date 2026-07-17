package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	HermesBaseURL string
	// HermesTuiBaseURL is the hermes-tui sidecar (official nousresearch/
	// hermes-agent image running `hermes dashboard`) — a separate service
	// from hermes-bridge/HermesBaseURL, see deploy/docker-compose.yml. Only
	// the Go API talks to it; it has no published host port.
	HermesTuiBaseURL string
	// HermesTuiSessionToken authenticates outbound calls to hermes-tui's
	// dashboard REST API (e.g. /api/cron/jobs, EP-12 crawl automation
	// upsert) — sent as "Authorization: Bearer <token>". Must equal
	// hermes-tui's own HERMES_DASHBOARD_SESSION_TOKEN env var (see
	// deploy/docker-compose.yml); optional like ConfigEncKey — without it,
	// the crawl-automation-in-Hermes feature just degrades to "not synced"
	// rather than blocking anything.
	HermesTuiSessionToken string
	// CronTriggerSecret gates POST /internal/discovery/trigger — the
	// callback the Hermes cron job (upserted on profile save, EP-12) hits to
	// ask "run discovery now if due". A dedicated secret rather than reusing
	// APIServerKey: this one gets embedded verbatim in the cron job's prompt
	// text (persisted to a file in Hermes's own workspace), so a leak there
	// shouldn't also compromise hermes-bridge's inbound auth.
	CronTriggerSecret string
	// InternalAPIBaseURL is how the Hermes cron job's callback (see
	// CronTriggerSecret) reaches this API from inside hermes-tui's
	// container — the compose-network DNS name, not the host-facing one.
	InternalAPIBaseURL  string
	APIServerKey        string
	WorkspaceSessionKey string
	SalesMCPToken       string
	SeedAdminEmail      string
	SeedAdminPassword   string
	UploadDir           string
	// ConfigEncKey encrypts ai_provider_setting.api_key_encrypted (EP-18
	// ST-18.4, AES-256-GCM — see internal/auth/crypto.go). Optional: the
	// AI Provider Config feature is simply unavailable without it, the
	// rest of the app works fine (PRD §8 non-blocking principle extended
	// to configuration, not just AI output).
	ConfigEncKey string
}

// Load reads config from environment (and .env if present). Returns error
// if any required variable is empty.
func Load() (*Config, error) {
	_ = godotenv.Load() // .env opsional; abaikan bila tidak ada

	cfg := &Config{
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		HermesBaseURL:         getEnv("HERMES_BASE_URL", "http://localhost:8642"),
		HermesTuiBaseURL:      getEnv("HERMES_TUI_BASE_URL", "http://hermes-tui:9119"),
		HermesTuiSessionToken: os.Getenv("HERMES_DASHBOARD_SESSION_TOKEN"),
		CronTriggerSecret:     os.Getenv("CRON_TRIGGER_SECRET"),
		InternalAPIBaseURL:    getEnv("INTERNAL_API_BASE_URL", "http://api:8080"),
		APIServerKey:          os.Getenv("API_SERVER_KEY"),
		WorkspaceSessionKey:   os.Getenv("WORKSPACE_SESSION_KEY"),
		SalesMCPToken:         os.Getenv("SALES_MCP_TOKEN"),
		SeedAdminEmail:        os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPassword:     os.Getenv("SEED_ADMIN_PASSWORD"),
		UploadDir:             getEnv("UPLOAD_DIR", "./data/uploads"),
		ConfigEncKey:          os.Getenv("CONFIG_ENC_KEY"),
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// MustLoad is Load but terminates the process on error.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	return cfg
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":          c.DatabaseURL,
		"JWT_SECRET":            c.JWTSecret,
		"API_SERVER_KEY":        c.APIServerKey,
		"WORKSPACE_SESSION_KEY": c.WorkspaceSessionKey,
		"SALES_MCP_TOKEN":       c.SalesMCPToken,
		"SEED_ADMIN_EMAIL":      c.SeedAdminEmail,
		"SEED_ADMIN_PASSWORD":   c.SeedAdminPassword,
	}
	for k, v := range required {
		if v == "" {
			return fmt.Errorf("required env %s is empty", k)
		}
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
