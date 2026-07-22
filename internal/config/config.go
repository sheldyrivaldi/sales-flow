package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

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
	// AutoMigrate controls whether RunMigrations (see
	// internal/repository/migrate.go) executes on boot. Defaults to true —
	// it's what makes the API self-sufficient against a fresh/company
	// database with no separate migrate step. Set AUTO_MIGRATE=false when
	// migrations are deliberately managed out-of-band (e.g. a DBA-controlled
	// company database where schema changes must go through a review
	// process instead of running automatically on every pod boot).
	AutoMigrate bool
	// ConfigEncKey encrypts ai_provider_setting.api_key_encrypted (EP-18
	// ST-18.4, AES-256-GCM — see internal/auth/crypto.go). Optional: the
	// AI Provider Config feature is simply unavailable without it, the
	// rest of the app works fine (PRD §8 non-blocking principle extended
	// to configuration, not just AI output).
	ConfigEncKey string

	// SMTP* mengirim undangan event terjadwal dari server. SEMUA opsional dan
	// dibaca lewat env yang disetel operator sendiri — aplikasi tidak pernah
	// menyimpan kredensial ini. Tanpa SMTPHost, fitur kirim-undangan degrade:
	// penjadwalan tetap tercatat tapi pengiriman ditandai gagal dengan pesan
	// jelas (mengikuti prinsip non-blocking PRD §8). SMTPFrom adalah alamat
	// pengirim amplop (akun relay); nama/alamat pengundang tetap dipasang di
	// header From/Reply-To agar undangan tampak datang dari yang mengundang.
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	// InviteTimezone menentukan arti "jam 07:00" pada jadwal non-custom (mis.
	// "Asia/Jakarta"). Default ke waktu lokal server bila kosong/invalid.
	InviteTimezone string
}

// SMTPConfigured melaporkan apakah pengiriman email siap dipakai.
func (c *Config) SMTPConfigured() bool {
	return c.SMTPHost != "" && c.SMTPPort != 0 && c.SMTPFrom != ""
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
		AutoMigrate:           getEnvBool("AUTO_MIGRATE", true),
		SMTPHost:              os.Getenv("SMTP_HOST"),
		SMTPPort:              getEnvInt("SMTP_PORT", 587),
		SMTPUsername:          os.Getenv("SMTP_USERNAME"),
		SMTPPassword:          os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:              os.Getenv("SMTP_FROM"),
		InviteTimezone:        getEnv("INVITE_TIMEZONE", "Asia/Jakarta"),
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

// getEnvInt parses an integer env var, falling back to def when unset/invalid.
func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("config: %s=%q invalid, pakai default %d", key, v, def)
		return def
	}
	return n
}

// getEnvBool parses a boolean env var (strconv.ParseBool: "1"/"0",
// "true"/"false", "TRUE"/"FALSE", etc.), falling back to def when unset or
// unparsable (with a warning in the latter case so a typo isn't silently
// ignored).
func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("config: %s=%q invalid, pakai default %t", key, v, def)
		return def
	}
	return b
}
