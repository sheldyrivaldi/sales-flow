package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"salespilot/internal/hermes"
)

// stubHermesClient is a minimal hermes.Client stub for unit testing
// KeywordService without a real Hermes bridge. Only GenerateJSON is
// meaningful; the rest satisfy the interface.
type stubHermesClient struct {
	generateJSON func(ctx context.Context, prompt string, schema any, sk hermes.SessionKey) (json.RawMessage, error)
}

var _ hermes.Client = (*stubHermesClient)(nil)

func (s *stubHermesClient) Chat(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
	return hermes.ChatResponse{}, errors.New("not implemented")
}

func (s *stubHermesClient) ChatStream(_ context.Context, _ hermes.ChatRequest) (<-chan hermes.Chunk, error) {
	return nil, errors.New("not implemented")
}

func (s *stubHermesClient) GenerateJSON(ctx context.Context, prompt string, schema any, sk hermes.SessionKey) (json.RawMessage, error) {
	return s.generateJSON(ctx, prompt, schema, sk)
}

func (s *stubHermesClient) Health(_ context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{}, errors.New("not implemented")
}

func (s *stubHermesClient) Configure(_ context.Context, _ hermes.ProviderConfig) error {
	return errors.New("not implemented")
}

func TestKeywordService_Generate_Success(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"keywords":["pengadaan aplikasi","integrasi sistem"],"negative_keywords":["ATK","sewa Kendaraan"]}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}

	svc := NewKeywordService(stub, "sk-test")
	resp, err := svc.Generate(context.Background(), []string{"Web App", "AI/Automation"}, "")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	if resp.Degraded {
		t.Error("Degraded = true, want false on success")
	}
	if resp.Language != "id" {
		t.Errorf("Language = %q, want default 'id'", resp.Language)
	}
	if len(resp.Keywords) != 2 {
		t.Errorf("Keywords = %v, want 2 items", resp.Keywords)
	}

	// "sewa Kendaraan" from AI output should dedup case-insensitively against
	// the preset's "sewa kendaraan".
	seen := map[string]int{}
	for _, k := range resp.NegativeKeywords {
		seen[k]++
	}
	for k, count := range seen {
		if count > 1 {
			t.Errorf("NegativeKeywords has duplicate (case-insensitive) entry: %q", k)
		}
	}
	if len(resp.NegativeKeywords) != len(negativeKeywordPreset) {
		t.Errorf("NegativeKeywords length = %d, want %d (preset + 1 new AI keyword, 1 deduped)",
			len(resp.NegativeKeywords), len(negativeKeywordPreset))
	}

	foundATK := false
	for _, k := range resp.NegativeKeywords {
		if k == "ATK" {
			foundATK = true
		}
	}
	if !foundATK {
		t.Errorf("NegativeKeywords missing preset entry %q: %v", "ATK", resp.NegativeKeywords)
	}
}

func TestKeywordService_Generate_Degraded(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}

	svc := NewKeywordService(stub, "sk-test")
	resp, err := svc.Generate(context.Background(), []string{"Web App"}, "id")
	if err != nil {
		t.Fatalf("Generate should not return a hard error on AI failure, got: %v", err)
	}

	if !resp.Degraded {
		t.Error("Degraded = false, want true when Hermes fails")
	}
	if len(resp.Keywords) != 0 {
		t.Errorf("Keywords = %v, want empty on degrade", resp.Keywords)
	}
	if len(resp.NegativeKeywords) != len(negativeKeywordPreset) {
		t.Errorf("NegativeKeywords = %v, want preset-only fallback", resp.NegativeKeywords)
	}
}

func TestDedupCaseInsensitive(t *testing.T) {
	in := []string{"ATK", "atk", " ATK ", "Sewa Kendaraan", "sewa kendaraan", "", "Katering"}
	got := dedupCaseInsensitive(in)
	want := []string{"ATK", "Sewa Kendaraan", "Katering"}
	if len(got) != len(want) {
		t.Fatalf("dedupCaseInsensitive(%v) = %v, want %v", in, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("dedupCaseInsensitive[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
