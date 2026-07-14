package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
)

// testConnectionPrompt is the minimal prompt used by Test to verify the
// active provider/model/API-key actually works end-to-end, not just that
// the bridge process is reachable.
const testConnectionPrompt = "Balas dengan teks singkat \"ok\" untuk konfirmasi koneksi. Tidak perlu format khusus."

// AISettingService orchestrates EP-18 ST-18.4 AI Provider Config: DB
// (ai_provider_setting, api key always encrypted at rest) is the source of
// truth; every successful Update also pushes the plaintext config to
// hermes-bridge via hermes.Configure so the next chat/scoring/etc. call
// uses it immediately, without a bridge restart (rehydrate-on-boot is
// TK-18.4.4, for when the bridge itself restarts).
type AISettingService struct {
	repo   domain.AISettingRepository
	hc     hermes.Client
	encKey []byte // nil/empty when CONFIG_ENC_KEY isn't configured — feature unavailable, not a crash (PRD §8 non-blocking, extended to config).
}

func NewAISettingService(repo domain.AISettingRepository, hc hermes.Client, encKey []byte) *AISettingService {
	return &AISettingService{repo: repo, hc: hc, encKey: encKey}
}

func (s *AISettingService) unavailable() error {
	return httperr.NewBadRequest("AI_CONFIG_UNAVAILABLE", "Konfigurasi AI Provider tidak tersedia (CONFIG_ENC_KEY belum diset)")
}

// Get returns the active config, masked, or a zero-value response
// (is_active=false, provider="") if none has ever been configured — a
// normal state, not an error. Never returns the plaintext key.
func (s *AISettingService) Get(ctx context.Context) (*dto.AISettingResponse, error) {
	if len(s.encKey) == 0 {
		return nil, s.unavailable()
	}

	active, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai_setting.Get: %w", err)
	}
	if active == nil {
		return &dto.AISettingResponse{}, nil
	}

	plain, err := auth.Decrypt(active.APIKeyEncrypted, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("ai_setting.Get: decrypt: %w", err)
	}
	resp := dto.ToAISettingResponse(*active, maskAPIKey(plain))
	return &resp, nil
}

// Update validates req, pushes it to the bridge via hermes.Configure FIRST,
// and only persists + activates it in DB (encrypting the API key — or, if
// req.APIKey is empty, reusing the currently-stored key, per the write-only
// field contract) once the bridge has accepted it. This ordering matters: if
// DB activation happened first, a permanently-rejected config (bad
// base_url/api_key/provider — the bridge validates all three) would
// deactivate the previously-working config while leaving the broken one
// active, and Rehydrate would keep failing to push it after every boot.
// Pushing first means a rejected config leaves the previous config (DB and
// bridge) completely untouched.
func (s *AISettingService) Update(ctx context.Context, req *dto.AISettingUpdateRequest) (*dto.AISettingResponse, error) {
	if len(s.encKey) == 0 {
		return nil, s.unavailable()
	}

	provider := domain.AIProvider(req.Provider)
	if !provider.Valid() {
		return nil, httperr.NewBadRequest("INVALID_PROVIDER", "provider harus openai atau openrouter")
	}

	// Snapshot the currently-active config before touching the bridge, so we
	// can roll the bridge back to it if the DB write below fails after the
	// bridge already accepted the new config (see the Upsert error path).
	prev, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai_setting.Update: get current: %w", err)
	}

	apiKeyPlain, err := s.resolveAPIKeyFrom(prev, req.APIKey)
	if err != nil {
		return nil, err
	}

	toolSets := parseToolsets(req.EnabledToolsets)

	if err := s.hc.Configure(ctx, hermes.ProviderConfig{
		Provider: string(provider),
		Model:    req.Model,
		BaseURL:  req.BaseURL,
		APIKey:   apiKeyPlain,
		ToolSets: toolSets,
	}); err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Konfigurasi ditolak Hermes (periksa provider/model/API key/base URL) — belum disimpan")
	}

	encrypted, err := auth.Encrypt(apiKeyPlain, s.encKey)
	if err != nil {
		s.rollbackBridge(ctx, prev)
		return nil, fmt.Errorf("ai_setting.Update: encrypt: %w", err)
	}

	setting := &domain.AISetting{
		Provider:        provider,
		Model:           req.Model,
		BaseURL:         req.BaseURL,
		APIKeyEncrypted: encrypted,
		EnabledToolsets: req.EnabledToolsets,
	}
	if err := s.repo.Upsert(ctx, setting); err != nil {
		// The bridge already switched to the new config but the DB still
		// records `prev` as active. Roll the bridge back so the two don't
		// diverge (a later boot Rehydrate would push `prev` anyway, but that
		// could be far off — don't leave the bridge silently running an
		// un-persisted config until then).
		s.rollbackBridge(ctx, prev)
		return nil, fmt.Errorf("ai_setting.Update: %w", err)
	}

	resp := dto.ToAISettingResponse(*setting, maskAPIKey(apiKeyPlain))
	return &resp, nil
}

// rollbackBridge best-effort re-pushes the previously-active config to the
// bridge after a failed Update that had already called Configure, so the
// bridge's in-memory config matches what's still active in the DB. If prev is
// nil (no prior config) or decrypt fails, there's nothing safe to re-push —
// log loudly so the divergence is visible; the next boot Rehydrate is the
// backstop.
func (s *AISettingService) rollbackBridge(ctx context.Context, prev *domain.AISetting) {
	if prev == nil {
		log.Printf("ai_setting: WARNING bridge sudah menerima config baru tapi DB gagal disimpan, dan tidak ada config lama untuk rollback — bridge mungkin memakai config yang tidak tersimpan sampai restart berikutnya")
		return
	}
	plain, err := auth.Decrypt(prev.APIKeyEncrypted, s.encKey)
	if err != nil {
		log.Printf("ai_setting: WARNING gagal decrypt config lama untuk rollback bridge: %v", err)
		return
	}
	if err := s.hc.Configure(ctx, hermes.ProviderConfig{
		Provider: string(prev.Provider),
		Model:    prev.Model,
		BaseURL:  prev.BaseURL,
		APIKey:   plain,
		ToolSets: parseToolsets(prev.EnabledToolsets),
	}); err != nil {
		log.Printf("ai_setting: WARNING gagal rollback bridge ke config lama: %v — bridge dan DB bisa divergen sampai restart berikutnya", err)
	}
}

// Test verifies the currently active provider/model/API-key actually work
// by making one real, cheap round-trip through the bridge's deterministic
// /v1/responses mode (skip_memory=true, no toolsets — the same mode scoring
// uses) — unlike hermesStatus/Health, which only proves the bridge process
// itself is reachable and would report success even with an invalid key.
func (s *AISettingService) Test(ctx context.Context) bool {
	_, err := s.hc.GenerateJSON(ctx, testConnectionPrompt, nil, "")
	return err == nil
}

// parseToolsets decodes req.EnabledToolsets (arbitrary JSON from the
// request body) into a string slice for hermes.ProviderConfig.ToolSets. A
// missing/empty/malformed value returns nil (no override — bridge keeps its
// own default) rather than failing the whole Update, since toolsets is a
// secondary convenience field, not core to what makes a provider config
// valid.
func parseToolsets(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var toolsets []string
	if err := json.Unmarshal(raw, &toolsets); err != nil {
		log.Printf("ai_setting: enabled_toolsets bukan array string valid, diabaikan: %v", err)
		return nil
	}
	return toolsets
}

// resolveAPIKeyFrom returns the plaintext key to encrypt+store: the new one
// if provided, otherwise the currently-active config's key decrypted
// (write-only field semantics — an empty api_key in the request means "don't
// change it"). current is the already-fetched active config (nil when none
// exists), passed in so Update doesn't hit the repo twice.
func (s *AISettingService) resolveAPIKeyFrom(current *domain.AISetting, newKey *string) (string, error) {
	if newKey != nil && *newKey != "" {
		return *newKey, nil
	}

	if current == nil {
		return "", httperr.NewBadRequest("API_KEY_REQUIRED", "api_key wajib diisi untuk konfigurasi pertama")
	}

	plain, err := auth.Decrypt(current.APIKeyEncrypted, s.encKey)
	if err != nil {
		return "", fmt.Errorf("ai_setting.resolveAPIKey: decrypt current: %w", err)
	}
	return plain, nil
}

// Rehydrate re-pushes the currently active config to the bridge — called at
// API boot (TK-18.4.4) so a hermes-bridge restart (which loses its
// in-memory config, since Configure only stores it there) doesn't silently
// fall back to stale env-var defaults. pushed=false+err=nil means "nothing
// to do" (no active config yet, or CONFIG_ENC_KEY unavailable) — a normal
// boot state, not a failure; the caller should log accordingly but must
// never fail boot over this (best-effort, mirrors PRD §8 non-blocking).
func (s *AISettingService) Rehydrate(ctx context.Context) (pushed bool, err error) {
	if len(s.encKey) == 0 {
		return false, nil
	}

	active, err := s.repo.GetActive(ctx)
	if err != nil {
		return false, fmt.Errorf("ai_setting.Rehydrate: %w", err)
	}
	if active == nil {
		return false, nil
	}

	plain, err := auth.Decrypt(active.APIKeyEncrypted, s.encKey)
	if err != nil {
		return false, fmt.Errorf("ai_setting.Rehydrate: decrypt: %w", err)
	}

	if err := s.hc.Configure(ctx, hermes.ProviderConfig{
		Provider: string(active.Provider),
		Model:    active.Model,
		BaseURL:  active.BaseURL,
		APIKey:   plain,
		ToolSets: parseToolsets(active.EnabledToolsets),
	}); err != nil {
		return false, fmt.Errorf("ai_setting.Rehydrate: configure: %w", err)
	}
	return true, nil
}

// maskAPIKey returns a display-safe hint like "sk-...abcd" — never the full
// key. Short keys are fully starred rather than partially exposed.
func maskAPIKey(plain string) string {
	const headLen, tailLen = 3, 4
	if plain == "" {
		return ""
	}
	if len(plain) <= headLen+tailLen {
		return strings.Repeat("*", len(plain))
	}
	return plain[:headLen] + "..." + plain[len(plain)-tailLen:]
}
