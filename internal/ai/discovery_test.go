package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// --- Fakes (deterministic, no DB/Hermes needed) ---

type fakeCrawler struct {
	discover func(ctx context.Context, in DiscoverInput) ([]CandidateTender, error)
}

func (f *fakeCrawler) Discover(ctx context.Context, in DiscoverInput) ([]CandidateTender, error) {
	return f.discover(ctx, in)
}

type fakeProfileGetter struct {
	agg *domain.ProfileAggregate
	err error
}

func (f *fakeProfileGetter) GetCurrent(_ context.Context) (*domain.ProfileAggregate, error) {
	return f.agg, f.err
}

type fakeSourceRepo struct {
	items []domain.Source
}

func (r *fakeSourceRepo) Create(_ context.Context, s *domain.Source) error { return nil }
func (r *fakeSourceRepo) GetByID(_ context.Context, id string) (*domain.Source, error) {
	for _, s := range r.items {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeSourceRepo) GetByPresetKey(_ context.Context, _ string) (*domain.Source, error) {
	return nil, errors.New("not found")
}
func (r *fakeSourceRepo) ListPresetKeys(_ context.Context) ([]string, error) { return nil, nil }
func (r *fakeSourceRepo) List(_ context.Context, f domain.SourceFilter, _, _ int) ([]domain.Source, int64, error) {
	var out []domain.Source
	for _, s := range r.items {
		if f.Enabled != nil && s.Enabled != *f.Enabled {
			continue
		}
		out = append(out, s)
	}
	return out, int64(len(out)), nil
}
func (r *fakeSourceRepo) Update(_ context.Context, _ *domain.Source) error { return nil }
func (r *fakeSourceRepo) Delete(_ context.Context, _ string) error         { return nil }

type fakeAuditRepo struct {
	events []domain.AuditEvent
}

func (r *fakeAuditRepo) Create(_ context.Context, e *domain.AuditEvent) error {
	r.events = append(r.events, *e)
	return nil
}

func testProfileAggregate() *domain.ProfileAggregate {
	return &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{CompanyName: "PT Contoh"},
		Target:  &domain.TargetCriteria{Countries: []string{"Indonesia"}},
		Keywords: []domain.KeywordSet{
			{Keywords: []string{"pengadaan aplikasi", "integrasi sistem"}},
			{Keywords: []string{"web development"}},
		},
	}
}

func TestOrchestrator_CollectCandidates_PassesEnabledSourcesAndKeywords(t *testing.T) {
	var gotInput DiscoverInput
	crawler := &fakeCrawler{
		discover: func(_ context.Context, in DiscoverInput) ([]CandidateTender, error) {
			gotInput = in
			return []CandidateTender{{Title: "Tender A"}}, nil
		},
	}
	sources := &fakeSourceRepo{items: []domain.Source{
		{ID: "s1", Name: "SPSE", Enabled: true, Access: domain.SourceAccessPublik},
		{ID: "s2", Name: "Disabled", Enabled: false, Access: domain.SourceAccessPublik},
	}}
	profiles := &fakeProfileGetter{agg: testProfileAggregate()}

	orch := NewDiscoveryOrchestrator(crawler, sources, profiles, &fakeAuditRepo{})
	candidates, allSources, err := orch.CollectCandidates(context.Background())
	if err != nil {
		t.Fatalf("CollectCandidates error: %v", err)
	}

	if len(candidates) != 1 || candidates[0].Title != "Tender A" {
		t.Errorf("candidates = %+v, want 1 candidate 'Tender A'", candidates)
	}
	if len(allSources) != 1 {
		t.Errorf("allSources (List result) = %+v, want 1 enabled source", allSources)
	}
	if len(gotInput.Sources) != 1 || gotInput.Sources[0].ID != "s1" {
		t.Errorf("crawler received sources = %+v, want only enabled source s1", gotInput.Sources)
	}
	if len(gotInput.Keywords) != 3 {
		t.Errorf("crawler received keywords = %v, want 3 flattened keywords", gotInput.Keywords)
	}
	if gotInput.Target == nil || gotInput.Target.Countries[0] != "Indonesia" {
		t.Errorf("crawler received Target = %+v, want Indonesia", gotInput.Target)
	}
}

func TestOrchestrator_CollectCandidates_NoEnabledSources_SkipsCrawlerCall(t *testing.T) {
	called := false
	crawler := &fakeCrawler{
		discover: func(_ context.Context, _ DiscoverInput) ([]CandidateTender, error) {
			called = true
			return nil, nil
		},
	}
	sources := &fakeSourceRepo{items: []domain.Source{
		{ID: "s1", Enabled: false},
	}}
	orch := NewDiscoveryOrchestrator(crawler, sources, &fakeProfileGetter{agg: testProfileAggregate()}, &fakeAuditRepo{})

	candidates, _, err := orch.CollectCandidates(context.Background())
	if err != nil {
		t.Fatalf("CollectCandidates error: %v", err)
	}
	if candidates != nil {
		t.Errorf("candidates = %+v, want nil when no enabled sources", candidates)
	}
	if called {
		t.Error("crawler.Discover should not be called when there are no enabled sources")
	}
}

func TestOrchestrator_CollectCandidates_ProfileError(t *testing.T) {
	crawler := &fakeCrawler{discover: func(_ context.Context, _ DiscoverInput) ([]CandidateTender, error) {
		return nil, nil
	}}
	profiles := &fakeProfileGetter{err: errors.New("db down")}
	orch := NewDiscoveryOrchestrator(crawler, &fakeSourceRepo{}, profiles, &fakeAuditRepo{})

	_, _, err := orch.CollectCandidates(context.Background())
	if err == nil {
		t.Fatal("CollectCandidates should return an error when profile lookup fails")
	}
}

// --- Compliance guard (§9) ---

func TestFilterCrawlableSources_OnlyPublikCrawlable(t *testing.T) {
	sources := []domain.Source{
		{ID: "s-publik", Access: domain.SourceAccessPublik, Priority: 1},
		{ID: "s-login", Access: domain.SourceAccessLogin, Priority: 5},
		{ID: "s-manual", Access: domain.SourceAccessManual, Priority: 5},
	}
	crawlable, skipped := filterCrawlableSources(sources)

	if len(crawlable) != 1 || crawlable[0].ID != "s-publik" {
		t.Errorf("crawlable = %+v, want only s-publik", crawlable)
	}
	if len(skipped) != 2 {
		t.Errorf("skipped = %+v, want 2 (login+manual)", skipped)
	}
}

func TestFilterCrawlableSources_SortedByPriorityDesc(t *testing.T) {
	sources := []domain.Source{
		{ID: "low", Access: domain.SourceAccessPublik, Priority: 1},
		{ID: "high", Access: domain.SourceAccessPublik, Priority: 10},
		{ID: "mid", Access: domain.SourceAccessPublik, Priority: 5},
	}
	crawlable, _ := filterCrawlableSources(sources)

	if len(crawlable) != 3 || crawlable[0].ID != "high" || crawlable[1].ID != "mid" || crawlable[2].ID != "low" {
		t.Errorf("crawlable order = %+v, want [high, mid, low]", crawlable)
	}
}

func TestOrchestrator_CollectCandidates_NonPublikSourceNeverReachesCrawler(t *testing.T) {
	var gotInput DiscoverInput
	crawler := &fakeCrawler{
		discover: func(_ context.Context, in DiscoverInput) ([]CandidateTender, error) {
			gotInput = in
			return nil, nil
		},
	}
	sources := &fakeSourceRepo{items: []domain.Source{
		{ID: "s-publik", Enabled: true, Access: domain.SourceAccessPublik},
		{ID: "s-login", Enabled: true, Access: domain.SourceAccessLogin},
	}}
	audit := &fakeAuditRepo{}
	orch := NewDiscoveryOrchestrator(crawler, sources, &fakeProfileGetter{agg: testProfileAggregate()}, audit)

	if _, _, err := orch.CollectCandidates(context.Background()); err != nil {
		t.Fatalf("CollectCandidates error: %v", err)
	}

	for _, s := range gotInput.Sources {
		if s.Access != domain.SourceAccessPublik {
			t.Errorf("crawler received non-publik source %+v — compliance guard violated", s)
		}
	}

	var crawledAction, skippedAction bool
	for _, e := range audit.events {
		if e.Action == "crawl_source" && e.TargetID != nil && *e.TargetID == "s-publik" {
			crawledAction = true
		}
		if e.Action == "skip_source_noncompliant" && e.TargetID != nil && *e.TargetID == "s-login" {
			skippedAction = true
		}
	}
	if !crawledAction {
		t.Error("expected audit_log entry action=crawl_source for s-publik")
	}
	if !skippedAction {
		t.Error("expected audit_log entry action=skip_source_noncompliant for s-login")
	}
}

func TestOrchestrator_CollectCandidates_AuditFailure_DoesNotAbortRun(t *testing.T) {
	crawler := &fakeCrawler{
		discover: func(_ context.Context, _ DiscoverInput) ([]CandidateTender, error) {
			return []CandidateTender{{Title: "Still Works"}}, nil
		},
	}
	sources := &fakeSourceRepo{items: []domain.Source{
		{ID: "s1", Enabled: true, Access: domain.SourceAccessPublik},
	}}
	orch := NewDiscoveryOrchestrator(crawler, sources, &fakeProfileGetter{agg: testProfileAggregate()}, &failingAuditRepo{})

	candidates, _, err := orch.CollectCandidates(context.Background())
	if err != nil {
		t.Fatalf("CollectCandidates should not fail when audit write fails, got: %v", err)
	}
	if len(candidates) != 1 {
		t.Errorf("candidates = %+v, want 1 (run must proceed despite audit failure)", candidates)
	}
}

type failingAuditRepo struct{}

func (f *failingAuditRepo) Create(_ context.Context, _ *domain.AuditEvent) error {
	return errors.New("audit db down")
}

func TestFlattenKeywords(t *testing.T) {
	got := flattenKeywords([]domain.KeywordSet{
		{Keywords: []string{"a", "b"}},
		{Keywords: []string{"c"}},
		{Keywords: nil},
	})
	if len(got) != 3 {
		t.Errorf("flattenKeywords = %v, want 3 items", got)
	}
}

// --- Dedup (§ EP-12 ST-12.3) ---

func TestComputeDedupKey_Stable(t *testing.T) {
	deadline := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	k1 := ComputeDedupKey("Pemkot Bandung", "Pengembangan Portal Vendor", &deadline)
	k2 := ComputeDedupKey("Pemkot Bandung", "Pengembangan Portal Vendor", &deadline)
	if k1 != k2 {
		t.Errorf("ComputeDedupKey not stable: %q != %q", k1, k2)
	}
	if k1 == "" {
		t.Error("ComputeDedupKey returned empty for non-empty buyer+title")
	}
}

func TestComputeDedupKey_NormalizesCaseAndWhitespace(t *testing.T) {
	deadline := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	k1 := ComputeDedupKey("Pemkot Bandung", "Pengembangan Portal Vendor", &deadline)
	k2 := ComputeDedupKey("  pemkot bandung  ", "PENGEMBANGAN PORTAL VENDOR", &deadline)
	if k1 != k2 {
		t.Errorf("ComputeDedupKey should be case/whitespace-insensitive: %q != %q", k1, k2)
	}
}

func TestComputeDedupKey_DifferentInputsDifferentKeys(t *testing.T) {
	deadline := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	k1 := ComputeDedupKey("Buyer A", "Title A", &deadline)
	k2 := ComputeDedupKey("Buyer B", "Title A", &deadline)
	k3 := ComputeDedupKey("Buyer A", "Title B", &deadline)
	otherDeadline := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	k4 := ComputeDedupKey("Buyer A", "Title A", &otherDeadline)

	keys := map[string]bool{k1: true, k2: true, k3: true, k4: true}
	if len(keys) != 4 {
		t.Errorf("expected 4 distinct keys, got %d: %v", len(keys), keys)
	}
}

func TestComputeDedupKey_EmptyBuyerAndTitle_ReturnsEmpty(t *testing.T) {
	if got := ComputeDedupKey("", "", nil); got != "" {
		t.Errorf("ComputeDedupKey(\"\",\"\",nil) = %q, want empty", got)
	}
}

// --- hermesCrawler wire parsing (regression: CandidateTender itself has no
// json tags — GenerateJSON must unmarshal through candidateWire instead, or
// every snake_case field silently fails to bind since Go's default
// case-insensitive struct matching does not ignore underscores) ---

func TestHermesCrawler_Discover_ParsesSnakeCaseFieldsAndDeadline(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{Content: "Ditemukan 1 tender di SPSE."}, nil
		},
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"candidates":[{"title":"Portal Vendor","buyer_name":"Pemkot Bandung",` +
				`"buyer_country":"Indonesia","buyer_industry":"Pemerintahan","value_estimate":2500000000,` +
				`"submission_deadline":"2026-08-15","source_name":"SPSE","source_url":"https://spse.example/1",` +
				`"service_category":"Web App","scope_summary":"Ringkasan","eligibility_requirements":"NIB",` +
				`"technical_requirements":"React"}]}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}

	crawler := NewHermesCrawler(stub, "sk-test")
	candidates, err := crawler.Discover(context.Background(), DiscoverInput{
		Sources: []domain.Source{{Name: "SPSE", URL: "https://spse.example"}},
	})
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want 1", candidates)
	}

	c := candidates[0]
	if c.BuyerName != "Pemkot Bandung" {
		t.Errorf("BuyerName = %q, want %q (snake_case field must bind)", c.BuyerName, "Pemkot Bandung")
	}
	if c.ValueEstimate == nil || *c.ValueEstimate != 2500000000 {
		t.Errorf("ValueEstimate = %v, want 2500000000", c.ValueEstimate)
	}
	if c.SourceName != "SPSE" || c.SourceURL != "https://spse.example/1" {
		t.Errorf("SourceName/SourceURL = %q/%q, want SPSE/https://spse.example/1", c.SourceName, c.SourceURL)
	}
	if c.SubmissionDeadline == nil {
		t.Fatal("SubmissionDeadline is nil, want parsed from \"2026-08-15\"")
	}
	want := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if !c.SubmissionDeadline.Equal(want) {
		t.Errorf("SubmissionDeadline = %v, want %v", c.SubmissionDeadline, want)
	}
}

func TestHermesCrawler_Discover_NoSources_SkipsChatCall(t *testing.T) {
	called := false
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			called = true
			return hermes.ChatResponse{}, nil
		},
	}
	crawler := NewHermesCrawler(stub, "sk-test")
	candidates, err := crawler.Discover(context.Background(), DiscoverInput{})
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if candidates != nil {
		t.Errorf("candidates = %+v, want nil", candidates)
	}
	if called {
		t.Error("Chat should not be called when there are no sources")
	}
}

// --- Rate limit + backoff (EP-12 ST-12.4.2) ---

func emptyExtractStub() *stubHermesClient {
	return &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{Content: "tidak ada temuan"}, nil
		},
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"candidates":[]}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				return nil, err
			}
			return raw, nil
		},
	}
}

func TestHermesCrawler_Discover_RespectsMinIntervalBetweenSources(t *testing.T) {
	var chatTimes []time.Time
	stub := emptyExtractStub()
	stub.chat = func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
		chatTimes = append(chatTimes, time.Now())
		return hermes.ChatResponse{Content: "tidak ada temuan"}, nil
	}

	interval := 30 * time.Millisecond
	crawler := &hermesCrawler{hc: stub, sk: "sk-test", minInterval: interval, maxRetries: 0}

	sources := []domain.Source{
		{ID: "s1", Name: "Sumber 1"},
		{ID: "s2", Name: "Sumber 2"},
		{ID: "s3", Name: "Sumber 3"},
	}
	if _, err := crawler.Discover(context.Background(), DiscoverInput{Sources: sources}); err != nil {
		t.Fatalf("Discover error: %v", err)
	}

	if len(chatTimes) != 3 {
		t.Fatalf("chat called %d times, want 3 (one per source)", len(chatTimes))
	}
	for i := 1; i < len(chatTimes); i++ {
		gap := chatTimes[i].Sub(chatTimes[i-1])
		if gap < interval {
			t.Errorf("gap between source %d and %d = %v, want >= %v (minInterval)", i-1, i, gap, interval)
		}
	}
}

func TestHermesCrawler_Discover_RetriesTransientErrorThenSucceeds(t *testing.T) {
	callCount := 0
	stub := emptyExtractStub()
	stub.chat = func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
		callCount++
		if callCount == 1 {
			return hermes.ChatResponse{}, errors.New("transient network error")
		}
		return hermes.ChatResponse{Content: "ditemukan"}, nil
	}

	crawler := &hermesCrawler{hc: stub, sk: "sk-test", minInterval: time.Millisecond, maxRetries: 2}
	candidates, err := crawler.Discover(context.Background(), DiscoverInput{
		Sources: []domain.Source{{ID: "s1", Name: "Sumber 1"}},
	})
	if err != nil {
		t.Fatalf("Discover should succeed after a retry, got error: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("candidates = %+v, want empty (stub extracts none)", candidates)
	}
	if callCount != 2 {
		t.Errorf("chat called %d times, want 2 (1 failure + 1 successful retry)", callCount)
	}
}

func TestHermesCrawler_Discover_GivesUpAfterMaxRetries(t *testing.T) {
	callCount := 0
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			callCount++
			return hermes.ChatResponse{}, errors.New("persistent failure")
		},
	}

	crawler := &hermesCrawler{hc: stub, sk: "sk-test", minInterval: time.Millisecond, maxRetries: 2}
	_, err := crawler.Discover(context.Background(), DiscoverInput{
		Sources: []domain.Source{{ID: "s1", Name: "Sumber 1"}},
	})
	if err == nil {
		t.Fatal("Discover should return an error once a source exhausts all retries")
	}
	if callCount != 3 {
		t.Errorf("chat called %d times, want 3 (1 initial + 2 retries = maxRetries+1)", callCount)
	}
}

func TestHermesCrawler_Discover_CtxCanceledDuringWait_StopsPromptly(t *testing.T) {
	stub := emptyExtractStub()
	crawler := &hermesCrawler{hc: stub, sk: "sk-test", minInterval: time.Hour, maxRetries: 0}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := crawler.Discover(ctx, DiscoverInput{
		Sources: []domain.Source{{ID: "s1"}, {ID: "s2"}},
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Discover should return an error when ctx is canceled mid-wait")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Discover took %v after ctx cancellation, want it to stop promptly (well under minInterval=1h)", elapsed)
	}
}
