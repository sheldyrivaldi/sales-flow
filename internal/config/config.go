package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	HermesBaseURL       string
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
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		HermesBaseURL:       getEnv("HERMES_BASE_URL", "http://localhost:8642"),
		APIServerKey:        os.Getenv("API_SERVER_KEY"),
		WorkspaceSessionKey: os.Getenv("WORKSPACE_SESSION_KEY"),
		SalesMCPToken:       os.Getenv("SALES_MCP_TOKEN"),
		SeedAdminEmail:      os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPassword:   os.Getenv("SEED_ADMIN_PASSWORD"),
		UploadDir:           getEnv("UPLOAD_DIR", "./data/uploads"),
		ConfigEncKey:        os.Getenv("CONFIG_ENC_KEY"),
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
