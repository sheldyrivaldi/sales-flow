package service

import (
	"context"
	"encoding/json"
	"testing"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
)

type fakeAISettingRepo struct {
	rows      []domain.AISetting
	upsertErr error // when non-nil, Upsert fails without persisting (simulates a DB error)
}

func (r *fakeAISettingRepo) GetActive(_ context.Context) (*domain.AISetting, error) {
	for _, s := range r.rows {
		if s.IsActive {
			cp := s
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeAISettingRepo) Upsert(_ context.Context, s *domain.AISetting) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	for i := range r.rows {
		r.rows[i].IsActive = false
	}
	s.IsActive = true
	if s.ID == "" {
		s.ID = "ai-setting-1"
	}
	r.rows = append(r.rows, *s)
	return nil
}

func testEncKey(t *testing.T) []byte {
	t.Helper()
	key, err := auth.DecodeEncKey("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	if err != nil {
		t.Fatalf("DecodeEncKey: %v", err)
	}
	return key
}

func TestAISettingService_Update_EncryptsAndConfiguresBridge(t *testing.T) {
	repo := &fakeAISettingRepo{}
	var configuredWith hermes.ProviderConfig
	stub := &stubHermesClient{
		configure: func(_ context.Context, cfg hermes.ProviderConfig) error {
			configuredWith = cfg
			return nil
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	apiKey := "sk-real-secret-key-abcd"
	resp, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   &apiKey,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if configuredWith.Provider != "openai" || configuredWith.Model != "gpt-4o" || configuredWith.APIKey != apiKey {
		t.Fatalf("hc.Configure called with %+v, want provider=openai model=gpt-4o api_key=%s", configuredWith, apiKey)
	}

	if len(repo.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(repo.rows))
	}
	if repo.rows[0].APIKeyEncrypted == apiKey {
		t.Fatal("stored key must be encrypted, not plaintext")
	}

	if resp.APIKeyMasked == apiKey || resp.APIKeyMasked == "" {
		t.Fatalf("APIKeyMasked = %q, want masked (not plaintext, not empty)", resp.APIKeyMasked)
	}
}

func TestAISettingService_Update_EmptyKeyPreservesExisting(t *testing.T) {
	repo := &fakeAISettingRepo{}
	stub := &stubHermesClient{
		configure: func(_ context.Context, _ hermes.ProviderConfig) error { return nil },
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	original := "sk-original-key-9999"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   &original,
	}); err != nil {
		t.Fatalf("first Update error: %v", err)
	}

	// Second update: change model, leave api_key nil — must keep the original key.
	empty := ""
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		APIKey:   &empty,
	}); err != nil {
		t.Fatalf("second Update error: %v", err)
	}

	if len(repo.rows) != 2 {
		t.Fatalf("rows = %d, want 2 (history preserved)", len(repo.rows))
	}
	latest := repo.rows[1]
	plain, err := auth.Decrypt(latest.APIKeyEncrypted, testEncKey(t))
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if plain != original {
		t.Fatalf("decrypted key = %q, want preserved original %q", plain, original)
	}
	if latest.Model != "gpt-4o-mini" {
		t.Fatalf("Model = %q, want gpt-4o-mini (should still update non-key fields)", latest.Model)
	}
}

func TestAISettingService_Update_NoExistingKey_RequiresAPIKey(t *testing.T) {
	repo := &fakeAISettingRepo{}
	svc := NewAISettingService(repo, &stubHermesClient{}, testEncKey(t))

	_, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
	})
	if err == nil {
		t.Fatal("expected error when api_key is empty and no config exists yet")
	}
	if len(repo.rows) != 0 {
		t.Errorf("rows = %d, want 0 (nothing should be persisted)", len(repo.rows))
	}
}

func TestAISettingService_Update_InvalidProvider(t *testing.T) {
	repo := &fakeAISettingRepo{}
	svc := NewAISettingService(repo, &stubHermesClient{}, testEncKey(t))

	key := "sk-x"
	_, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "not-a-real-provider",
		Model:    "x",
		APIKey:   &key,
	})
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
	if len(repo.rows) != 0 {
		t.Errorf("rows = %d, want 0", len(repo.rows))
	}
}

func TestAISettingService_Update_BridgePushFails_NoPartialSuccess(t *testing.T) {
	repo := &fakeAISettingRepo{}
	stub := &stubHermesClient{
		configure: func(_ context.Context, _ hermes.ProviderConfig) error {
			return context.DeadlineExceeded
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	key := "sk-x"
	_, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   &key,
	})
	if err == nil {
		t.Fatal("expected error when bridge push fails")
	}
	// Configure is called BEFORE persisting (see Update's ordering comment):
	// a rejected/unreachable bridge must leave the DB completely untouched,
	// so a previously-working config is never deactivated in favor of a
	// broken one.
	if len(repo.rows) != 0 {
		t.Fatalf("rows = %d, want 0 (nothing persisted when bridge push fails)", len(repo.rows))
	}
}

// TestAISettingService_Update_RejectedConfig_LeavesPreviousActive verifies
// the fix for the ordering bug: if a SECOND Update's bridge push fails, the
// FIRST (working) config must remain active in DB — not deactivated.
func TestAISettingService_Update_RejectedConfig_LeavesPreviousActive(t *testing.T) {
	repo := &fakeAISettingRepo{}
	configureShouldFail := false
	stub := &stubHermesClient{
		configure: func(_ context.Context, _ hermes.ProviderConfig) error {
			if configureShouldFail {
				return context.DeadlineExceeded
			}
			return nil
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	goodKey := "sk-good-key"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   &goodKey,
	}); err != nil {
		t.Fatalf("first Update error: %v", err)
	}

	configureShouldFail = true
	badKey := "sk-bad-key"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openrouter",
		Model:    "anthropic/claude-sonnet-4-6",
		APIKey:   &badKey,
	}); err == nil {
		t.Fatal("expected error when second Update's bridge push fails")
	}

	if len(repo.rows) != 1 {
		t.Fatalf("rows = %d, want 1 (rejected second config must not be persisted)", len(repo.rows))
	}
	if !repo.rows[0].IsActive {
		t.Error("first config must still be active — a rejected update must not deactivate it")
	}
	if repo.rows[0].Provider != domain.AIProviderOpenAI {
		t.Errorf("active provider = %q, want openai (first config preserved)", repo.rows[0].Provider)
	}
}

func TestAISettingService_Get_NoneConfigured(t *testing.T) {
	repo := &fakeAISettingRepo{}
	svc := NewAISettingService(repo, &stubHermesClient{}, testEncKey(t))

	resp, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.IsActive {
		t.Error("IsActive should be false when nothing configured")
	}
	if resp.APIKeyMasked != "" {
		t.Errorf("APIKeyMasked = %q, want empty", resp.APIKeyMasked)
	}
}

func TestAISettingService_Get_MaskedNeverPlaintext(t *testing.T) {
	repo := &fakeAISettingRepo{}
	stub := &stubHermesClient{configure: func(_ context.Context, _ hermes.ProviderConfig) error { return nil }}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	apiKey := "sk-super-secret-value-1234"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openrouter",
		Model:    "anthropic/claude-sonnet-4-6",
		APIKey:   &apiKey,
	}); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	resp, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.APIKeyMasked == apiKey {
		t.Fatal("Get must never return the plaintext key")
	}
	if resp.Provider != "openrouter" || resp.Model != "anthropic/claude-sonnet-4-6" {
		t.Errorf("Get returned wrong config: %+v", resp)
	}
}

func TestAISettingService_Unavailable_WithoutEncKey(t *testing.T) {
	repo := &fakeAISettingRepo{}
	svc := NewAISettingService(repo, &stubHermesClient{}, nil)

	if _, err := svc.Get(context.Background()); err == nil {
		t.Fatal("Get should error when encKey is nil")
	}
	key := "sk-x"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{Provider: "openai", Model: "x", APIKey: &key}); err == nil {
		t.Fatal("Update should error when encKey is nil")
	}
}

func TestAISettingService_Rehydrate_NoConfig(t *testing.T) {
	repo := &fakeAISettingRepo{}
	svc := NewAISettingService(repo, &stubHermesClient{}, testEncKey(t))

	pushed, err := svc.Rehydrate(context.Background())
	if err != nil {
		t.Fatalf("Rehydrate error: %v", err)
	}
	if pushed {
		t.Error("pushed should be false when nothing is configured")
	}
}

func TestAISettingService_Rehydrate_NoEncKey(t *testing.T) {
	repo := &fakeAISettingRepo{}
	svc := NewAISettingService(repo, &stubHermesClient{}, nil)

	pushed, err := svc.Rehydrate(context.Background())
	if err != nil {
		t.Fatalf("Rehydrate error: %v", err)
	}
	if pushed {
		t.Error("pushed should be false when encKey is nil")
	}
}

func TestAISettingService_Rehydrate_PushesActiveConfig(t *testing.T) {
	repo := &fakeAISettingRepo{}
	var configuredWith hermes.ProviderConfig
	stub := &stubHermesClient{
		configure: func(_ context.Context, cfg hermes.ProviderConfig) error {
			configuredWith = cfg
			return nil
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	apiKey := "sk-rehydrate-test-key"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   &apiKey,
	}); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	// Reset to prove Rehydrate makes its own Configure call, independent
	// of Update's.
	configuredWith = hermes.ProviderConfig{}

	pushed, err := svc.Rehydrate(context.Background())
	if err != nil {
		t.Fatalf("Rehydrate error: %v", err)
	}
	if !pushed {
		t.Fatal("pushed should be true when a config is active")
	}
	if configuredWith.APIKey != apiKey || configuredWith.Model != "gpt-4o" {
		t.Fatalf("Rehydrate pushed %+v, want plaintext key %q restored", configuredWith, apiKey)
	}
}

func TestAISettingService_Rehydrate_BridgeDownReturnsError(t *testing.T) {
	repo := &fakeAISettingRepo{}
	stub := &stubHermesClient{
		configure: func(_ context.Context, _ hermes.ProviderConfig) error { return context.DeadlineExceeded },
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	// Seed a config directly in the repo (bypassing Update, which would now
	// refuse to persist anything given a bridge that always errors) so
	// Rehydrate has an active config to attempt pushing at boot — the exact
	// scenario Rehydrate exists for: the bridge was reachable when this row
	// was saved, but is down again on this particular boot.
	encrypted, err := auth.Encrypt("sk-x", testEncKey(t))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if err := repo.Upsert(context.Background(), &domain.AISetting{
		Provider:        domain.AIProviderOpenAI,
		Model:           "x",
		APIKeyEncrypted: encrypted,
	}); err != nil {
		t.Fatalf("seed Upsert: %v", err)
	}

	pushed, err := svc.Rehydrate(context.Background())
	if err == nil {
		t.Fatal("Rehydrate should error when the bridge push fails")
	}
	if pushed {
		t.Error("pushed should be false on error")
	}
}

// TestAISettingService_Update_UpsertFails_RollsBackBridge verifies that when
// the bridge accepts the new config but the DB write then fails, the bridge is
// rolled back to the previously-active config so the two don't diverge.
func TestAISettingService_Update_UpsertFails_RollsBackBridge(t *testing.T) {
	repo := &fakeAISettingRepo{}
	var configured []hermes.ProviderConfig
	stub := &stubHermesClient{
		configure: func(_ context.Context, cfg hermes.ProviderConfig) error {
			configured = append(configured, cfg)
			return nil
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	// First update succeeds and becomes the active config.
	goodKey := "sk-good-key"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   &goodKey,
	}); err != nil {
		t.Fatalf("first Update error: %v", err)
	}

	// Second update: the bridge accepts it, but the DB write fails.
	repo.upsertErr = context.DeadlineExceeded
	configured = nil
	newKey := "sk-new-key"
	if _, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider: "openrouter",
		Model:    "anthropic/claude-sonnet-4-6",
		APIKey:   &newKey,
	}); err == nil {
		t.Fatal("expected error when Upsert fails")
	}

	// Two Configure calls this round: the new (failed) config, then a
	// rollback to the previous good config.
	if len(configured) != 2 {
		t.Fatalf("Configure calls = %d, want 2 (push + rollback)", len(configured))
	}
	rollback := configured[len(configured)-1]
	if rollback.Provider != "openai" || rollback.Model != "gpt-4o" || rollback.APIKey != goodKey {
		t.Fatalf("rollback pushed %+v, want the previous openai/gpt-4o/%s config", rollback, goodKey)
	}

	// DB still has exactly the first (good) config, active.
	if len(repo.rows) != 1 || !repo.rows[0].IsActive || repo.rows[0].Provider != domain.AIProviderOpenAI {
		t.Fatalf("DB state = %+v, want the original openai config still active", repo.rows)
	}
}

func TestAISettingService_Update_PassesToolsetsToConfigure(t *testing.T) {
	repo := &fakeAISettingRepo{}
	var configuredWith hermes.ProviderConfig
	stub := &stubHermesClient{
		configure: func(_ context.Context, cfg hermes.ProviderConfig) error {
			configuredWith = cfg
			return nil
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	key := "sk-x"
	_, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider:        "openai",
		Model:           "gpt-4o",
		APIKey:          &key,
		EnabledToolsets: json.RawMessage(`["web","search"]`),
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if len(configuredWith.ToolSets) != 2 || configuredWith.ToolSets[0] != "web" || configuredWith.ToolSets[1] != "search" {
		t.Fatalf("ToolSets = %v, want [web search]", configuredWith.ToolSets)
	}
}

func TestAISettingService_Update_EmptyToolsetsOmitted(t *testing.T) {
	repo := &fakeAISettingRepo{}
	var configuredWith hermes.ProviderConfig
	stub := &stubHermesClient{
		configure: func(_ context.Context, cfg hermes.ProviderConfig) error {
			configuredWith = cfg
			return nil
		},
	}
	svc := NewAISettingService(repo, stub, testEncKey(t))

	key := "sk-x"
	_, err := svc.Update(context.Background(), &dto.AISettingUpdateRequest{
		Provider:        "openai",
		Model:           "gpt-4o",
		APIKey:          &key,
		EnabledToolsets: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if len(configuredWith.ToolSets) != 0 {
		t.Fatalf("ToolSets = %v, want empty/nil (no override)", configuredWith.ToolSets)
	}
}

func TestAISettingService_Test_UsesRealProviderRoundTrip(t *testing.T) {
	repo := &fakeAISettingRepo{}

	okStub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return json.RawMessage(`"ok"`), nil
		},
	}
	svc := NewAISettingService(repo, okStub, testEncKey(t))
	if !svc.Test(context.Background()) {
		t.Error("Test should return true when GenerateJSON succeeds")
	}

	failStub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, context.DeadlineExceeded
		},
	}
	svc2 := NewAISettingService(repo, failStub, testEncKey(t))
	if svc2.Test(context.Background()) {
		t.Error("Test should return false when GenerateJSON fails (e.g. invalid API key)")
	}
}
