package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/telemetry"
)

// stubHermesClient is a minimal hermes.Client stub for unit testing
// KeywordService/ReportService without a real Hermes bridge. generateJSON is
// meaningful for JSON-schema callers (ScoreService/PlaybookService/
// KeywordService); chat is meaningful for ReportService (Chat, not
// GenerateJSON, per ai.ReportGenerator). generateJSONFromDocument additionally
// makes this stub satisfy hermes.DocumentExtractor (profile_service_ingest_test.go's
// PDF ingest tests) when set — left nil, GenerateJSONFromDocument reports
// "not implemented", same non-support signal a real client lacking the
// capability would never actually give (it always implements it), but keeps
// this stub usable for tests that don't care about ingest at all. Any field
// may be left nil if unused by a given test.
type stubHermesClient struct {
	generateJSON             func(ctx context.Context, prompt string, schema any, sk hermes.SessionKey) (json.RawMessage, error)
	generateJSONFromDocument func(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk hermes.SessionKey) (json.RawMessage, error)
	chat                     func(ctx context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error)
	configure                func(ctx context.Context, cfg hermes.ProviderConfig) error
}

var _ hermes.Client = (*stubHermesClient)(nil)
var _ hermes.DocumentExtractor = (*stubHermesClient)(nil)

func (s *stubHermesClient) Chat(ctx context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error) {
	if s.chat != nil {
		return s.chat(ctx, req)
	}
	return hermes.ChatResponse{}, errors.New("not implemented")
}

func (s *stubHermesClient) ChatStream(_ context.Context, _ hermes.ChatRequest) (<-chan hermes.Chunk, error) {
	return nil, errors.New("not implemented")
}

func (s *stubHermesClient) GenerateJSON(ctx context.Context, prompt string, schema any, sk hermes.SessionKey) (json.RawMessage, error) {
	return s.generateJSON(ctx, prompt, schema, sk)
}

func (s *stubHermesClient) GenerateJSONFromDocument(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk hermes.SessionKey) (json.RawMessage, error) {
	if s.generateJSONFromDocument != nil {
		return s.generateJSONFromDocument(ctx, prompt, filename, fileBytes, schema, sk)
	}
	return nil, errors.New("not implemented")
}

func (s *stubHermesClient) Health(_ context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{}, errors.New("not implemented")
}

func (s *stubHermesClient) Configure(ctx context.Context, cfg hermes.ProviderConfig) error {
	if s.configure != nil {
		return s.configure(ctx, cfg)
	}
	return errors.New("not implemented")
}

func (s *stubHermesClient) ResetMemory(_ context.Context, _ hermes.SessionKey) error {
	return errors.New("not implemented")
}

// fakeTelemetryRepo is a shared in-memory domain.TelemetryRepository for
// tests verifying telemetry.Emitter.Emit is actually invoked (EP-17
// ST-17.1 TK-17.1.2). Emit is async, so tests must poll (see waitForEvents)
// rather than assert immediately after the call that triggers it.
type fakeTelemetryRepo struct {
	mu     sync.Mutex
	events []*domain.TelemetryEvent
}

func (f *fakeTelemetryRepo) Create(_ context.Context, e *domain.TelemetryEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
	return nil
}

func (f *fakeTelemetryRepo) CountByEvent(_ context.Context, event string, since time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, e := range f.events {
		if e.Event == event {
			n++
		}
	}
	return n, nil
}

func (f *fakeTelemetryRepo) snapshot() []*domain.TelemetryEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*domain.TelemetryEvent, len(f.events))
	copy(out, f.events)
	return out
}

// waitForEvents polls until n events named event have been recorded, or
// fails the test after a short timeout — needed because Emit fires in its
// own goroutine.
func waitForEvents(t *testing.T, repo *fakeTelemetryRepo, event string, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		count := 0
		for _, e := range repo.snapshot() {
			if e.Event == event {
				count++
			}
		}
		if count >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout menunggu %d event %q", n, event)
}

func newTestEmitter() (*telemetry.Emitter, *fakeTelemetryRepo) {
	repo := &fakeTelemetryRepo{}
	return telemetry.NewEmitter(repo), repo
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
