package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

type fakeCandidateCollector struct {
	candidates []ai.CandidateTender
	err        error
}

func (c *fakeCandidateCollector) CollectCandidates(_ context.Context) ([]ai.CandidateTender, []domain.Source, error) {
	return c.candidates, nil, c.err
}

func successScoreJSON(fitScore int) []byte {
	return []byte(`{"fit_score":` + strconv.Itoa(fitScore) + `,"recommended_action":"PURSUE","confidence":0.8,` +
		`"reasoning":"cocok","evidence":[],"risk_flags":[],"no_go_triggered":false,"need_partner":false}`)
}

// fakeDiscoveryRunRepo is an in-memory domain.DiscoveryRunRepository — mirrors
// the fake-repo pattern used throughout internal/service (e.g.
// fakeProspectScoreRepo) so DiscoveryService's idempotency/lifecycle logic is
// testable without Postgres.
type fakeDiscoveryRunRepo struct {
	items     map[string]domain.DiscoveryRun
	createErr error // if set, the next Create fails with this error then is cleared
	nextID    int
}

func (r *fakeDiscoveryRunRepo) Create(_ context.Context, run *domain.DiscoveryRun) error {
	if r.createErr != nil {
		err := r.createErr
		r.createErr = nil
		return err
	}
	if run.ID == "" {
		r.nextID++
		run.ID = fmt.Sprintf("run-%d", r.nextID)
	}
	if r.items == nil {
		r.items = map[string]domain.DiscoveryRun{}
	}
	r.items[run.ID] = *run
	return nil
}

func (r *fakeDiscoveryRunRepo) Update(_ context.Context, run *domain.DiscoveryRun) error {
	r.items[run.ID] = *run
	return nil
}

func (r *fakeDiscoveryRunRepo) GetByID(_ context.Context, id string) (*domain.DiscoveryRun, error) {
	run, ok := r.items[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &run, nil
}

func (r *fakeDiscoveryRunRepo) List(_ context.Context, _, _ int) ([]domain.DiscoveryRun, int64, error) {
	var out []domain.DiscoveryRun
	for _, run := range r.items {
		out = append(out, run)
	}
	return out, int64(len(out)), nil
}

func (r *fakeDiscoveryRunRepo) GetByCorrelationKey(_ context.Context, key string) (*domain.DiscoveryRun, error) {
	for _, run := range r.items {
		if run.CorrelationKey != nil && *run.CorrelationKey == key {
			return &run, nil
		}
	}
	return nil, nil
}

func newTestDiscoveryService(collector candidateCollector, stub *stubHermesClient) (*DiscoveryService, *fakeScoreTenderRepo, *fakeDiscoveryRunRepo) {
	tenderRepo := &fakeScoreTenderRepo{items: map[string]domain.Tender{}}
	profileRepo := &fakeScoreProfileRepo{}
	scoreRepo := &fakeProspectScoreRepo{}
	runRepo := &fakeDiscoveryRunRepo{items: map[string]domain.DiscoveryRun{}}

	tenderSvc := NewTenderService(tenderRepo, &fakeOutcomeRepo{}, NoopLearningHook())
	prospectSvc := NewProspectService(&fakeScoreProspectRepo{items: map[string]domain.Prospect{}}, &fakeOutcomeRepo{}, NoopLearningHook())
	profileSvc := NewProfileService(profileRepo, "", nil)
	scorer := ai.NewScorer(stub, "sk-test")
	scoreSvc := NewScoreService(scorer, scoreRepo, tenderSvc, prospectSvc, profileSvc)

	return NewDiscoveryService(collector, tenderSvc, scoreSvc, runRepo), tenderRepo, runRepo
}

func TestDiscoveryService_RunOnce_Success(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Tender A", BuyerName: "Buyer A"},
		{Title: "Tender B", BuyerName: "Buyer B"},
	}}
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := successScoreJSON(85)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, stub)

	saved, _, errs := svc.runOnce(context.Background())
	if saved != 2 {
		t.Errorf("saved = %d, want 2", saved)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v, want none", errs)
	}
	if len(tenderRepo.items) != 2 {
		t.Fatalf("tenderRepo has %d items, want 2", len(tenderRepo.items))
	}
	for _, tender := range tenderRepo.items {
		if tender.Origin != domain.OriginDiscovery {
			t.Errorf("tender.Origin = %q, want %q", tender.Origin, domain.OriginDiscovery)
		}
		if tender.Status != domain.TenderStatusIdentified {
			t.Errorf("tender.Status = %q, want %q", tender.Status, domain.TenderStatusIdentified)
		}
		if tender.FitScore == nil || *tender.FitScore != 85 {
			t.Errorf("tender.FitScore = %v, want 85", tender.FitScore)
		}
	}
}

func TestDiscoveryService_RunOnce_ScoreFailure_TenderStillSaved(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Tender A"},
	}}
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, stub)

	saved, _, errs := svc.runOnce(context.Background())
	if saved != 1 {
		t.Errorf("saved = %d, want 1 (tender persists even if scoring fails)", saved)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want 1 scoring error", errs)
	}
	if len(tenderRepo.items) != 1 {
		t.Fatalf("tenderRepo has %d items, want 1", len(tenderRepo.items))
	}
	for _, tender := range tenderRepo.items {
		if tender.FitScore != nil {
			t.Errorf("tender.FitScore = %v, want nil (unscored)", tender.FitScore)
		}
		if tender.Origin != domain.OriginDiscovery {
			t.Errorf("tender.Origin = %q, want %q even when unscored", tender.Origin, domain.OriginDiscovery)
		}
	}
}

func TestDiscoveryService_RunOnce_PartialFailure_OtherCandidatesUnaffected(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Fails"},
		{Title: "Succeeds"},
	}}
	callCount := 0
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("hermes transient error")
			}
			raw := successScoreJSON(70)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, stub)

	saved, _, errs := svc.runOnce(context.Background())
	if saved != 2 {
		t.Errorf("saved = %d, want 2 (both tenders persisted)", saved)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want exactly 1 (only the first candidate's scoring failed)", errs)
	}

	var scoredCount int
	for _, tender := range tenderRepo.items {
		if tender.FitScore != nil {
			scoredCount++
		}
	}
	if scoredCount != 1 {
		t.Errorf("scoredCount = %d, want 1 (only 'Succeeds' should have a score)", scoredCount)
	}
}

func TestDiscoveryService_RunOnce_CollectError(t *testing.T) {
	collector := &fakeCandidateCollector{err: errors.New("crawl failed")}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, &stubHermesClient{})

	saved, _, errs := svc.runOnce(context.Background())
	if saved != 0 {
		t.Errorf("saved = %d, want 0", saved)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want 1 collect error", errs)
	}
	if len(tenderRepo.items) != 0 {
		t.Errorf("tenderRepo has %d items, want 0", len(tenderRepo.items))
	}
}

// --- Dedup (EP-12 ST-12.3) ---

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("mustParseDate(%q): %v", s, err)
	}
	return parsed
}

func TestDiscoveryService_RunOnce_DedupsIdenticalCandidates(t *testing.T) {
	deadline := mustParseDate(t, "2026-08-15")
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Portal Vendor", BuyerName: "Pemkot Bandung", SubmissionDeadline: &deadline, SourceName: "SPSE"},
		{Title: "Portal Vendor", BuyerName: "Pemkot Bandung", SubmissionDeadline: &deadline, SourceName: "Inaproc"},
	}}
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := successScoreJSON(80)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, stub)

	saved, duplicates, errs := svc.runOnce(context.Background())
	if saved != 1 {
		t.Errorf("saved = %d, want 1 (second candidate is a duplicate)", saved)
	}
	if duplicates != 1 {
		t.Errorf("duplicates = %d, want 1", duplicates)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v, want none (a duplicate is not an error)", errs)
	}
	if len(tenderRepo.items) != 1 {
		t.Errorf("tenderRepo has %d items, want 1 (no double insert)", len(tenderRepo.items))
	}
}

func TestDiscoveryService_RunOnce_DifferentCandidatesNotDeduped(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Portal Vendor A", BuyerName: "Pemkot Bandung"},
		{Title: "Portal Vendor B", BuyerName: "Pemkot Surabaya"},
	}}
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := successScoreJSON(80)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, stub)

	saved, duplicates, _ := svc.runOnce(context.Background())
	if saved != 2 {
		t.Errorf("saved = %d, want 2 (distinct tenders)", saved)
	}
	if duplicates != 0 {
		t.Errorf("duplicates = %d, want 0", duplicates)
	}
	if len(tenderRepo.items) != 2 {
		t.Errorf("tenderRepo has %d items, want 2", len(tenderRepo.items))
	}
}

func TestDiscoveryService_RunOnce_DedupsAgainstPreExistingTender(t *testing.T) {
	// A tender created by a previous run (already has the dedup key) must
	// also be recognized by a fresh run, not just within the same batch.
	deadline := mustParseDate(t, "2026-08-15")
	key := ai.ComputeDedupKey("Pemkot Bandung", "Portal Vendor", &deadline)
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Portal Vendor", BuyerName: "Pemkot Bandung", SubmissionDeadline: &deadline},
	}}
	svc, tenderRepo, _ := newTestDiscoveryService(collector, &stubHermesClient{})
	tenderRepo.items["existing"] = domain.Tender{ID: "existing", Title: "Portal Vendor", DedupKey: &key}

	saved, duplicates, errs := svc.runOnce(context.Background())
	if saved != 0 {
		t.Errorf("saved = %d, want 0", saved)
	}
	if duplicates != 1 {
		t.Errorf("duplicates = %d, want 1", duplicates)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v, want none", errs)
	}
	if len(tenderRepo.items) != 1 {
		t.Errorf("tenderRepo has %d items, want 1 (unchanged)", len(tenderRepo.items))
	}
}

func TestDiscoveryService_RunOnce_RaceUniqueViolation_TreatedAsDuplicate(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Race Candidate", BuyerName: "Buyer"},
	}}
	tenderRepo := &raceUniqueViolationTenderRepo{fakeScoreTenderRepo: fakeScoreTenderRepo{items: map[string]domain.Tender{}}}
	profileRepo := &fakeScoreProfileRepo{}
	scoreRepo := &fakeProspectScoreRepo{}
	tenderSvc := NewTenderService(tenderRepo, &fakeOutcomeRepo{}, NoopLearningHook())
	prospectSvc := NewProspectService(&fakeScoreProspectRepo{items: map[string]domain.Prospect{}}, &fakeOutcomeRepo{}, NoopLearningHook())
	profileSvc := NewProfileService(profileRepo, "", nil)
	scoreSvc := NewScoreService(ai.NewScorer(&stubHermesClient{}, "sk-test"), scoreRepo, tenderSvc, prospectSvc, profileSvc)
	svc := NewDiscoveryService(collector, tenderSvc, scoreSvc, &fakeDiscoveryRunRepo{items: map[string]domain.DiscoveryRun{}})

	saved, duplicates, errs := svc.runOnce(context.Background())
	if saved != 0 {
		t.Errorf("saved = %d, want 0", saved)
	}
	if duplicates != 1 {
		t.Errorf("duplicates = %d, want 1 (unique-violation race treated as duplicate)", duplicates)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v, want none — a race duplicate must not surface as a hard error", errs)
	}
}

// raceUniqueViolationTenderRepo simulates the check-then-create race: the
// dedup lookup finds nothing (no prior duplicate), but Create still fails
// with a Postgres unique-violation because another concurrent run just
// inserted the same dedup_key.
type raceUniqueViolationTenderRepo struct {
	fakeScoreTenderRepo
}

func (r *raceUniqueViolationTenderRepo) Create(_ context.Context, _ *domain.Tender) error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
}

// --- Idempotent run (EP-12 ST-12.3.2) ---

func strPtr(s string) *string { return &s }

func TestDiscoveryService_StartRun_NoKey_AlwaysCreatesNew(t *testing.T) {
	svc, _, runRepo := newTestDiscoveryService(&fakeCandidateCollector{}, &stubHermesClient{})

	run1, started1, err := svc.StartRun(context.Background(), nil)
	if err != nil {
		t.Fatalf("StartRun error: %v", err)
	}
	if !started1 {
		t.Error("started1 = false, want true (first run)")
	}

	run2, started2, err := svc.StartRun(context.Background(), nil)
	if err != nil {
		t.Fatalf("StartRun error: %v", err)
	}
	if !started2 {
		t.Error("started2 = false, want true (no key means always a new run)")
	}
	if run1.ID == run2.ID {
		t.Error("run1.ID == run2.ID, want distinct runs when no correlation key is given")
	}
	if len(runRepo.items) != 2 {
		t.Errorf("runRepo has %d items, want 2", len(runRepo.items))
	}
}

func TestDiscoveryService_StartRun_SameKey_ReturnsExistingRun(t *testing.T) {
	svc, _, runRepo := newTestDiscoveryService(&fakeCandidateCollector{}, &stubHermesClient{})
	key := strPtr("sched-2026-07-10")

	run1, started1, err := svc.StartRun(context.Background(), key)
	if err != nil {
		t.Fatalf("first StartRun error: %v", err)
	}
	if !started1 {
		t.Error("started1 = false, want true (first call with this key)")
	}

	run2, started2, err := svc.StartRun(context.Background(), key)
	if err != nil {
		t.Fatalf("second StartRun error: %v", err)
	}
	if started2 {
		t.Error("started2 = true, want false (second call must return the existing run, not start a new one)")
	}
	if run1.ID != run2.ID {
		t.Errorf("run1.ID = %q, run2.ID = %q, want same run returned", run1.ID, run2.ID)
	}
	if len(runRepo.items) != 1 {
		t.Errorf("runRepo has %d items, want 1 (idempotent — no second run created)", len(runRepo.items))
	}
}

func TestDiscoveryService_StartRun_RaceUniqueViolation_ReturnsExisting(t *testing.T) {
	svc, _, runRepo := newTestDiscoveryService(&fakeCandidateCollector{}, &stubHermesClient{})
	key := strPtr("sched-2026-07-10")

	// Simulate a concurrent request that already created the run between our
	// lookup and our own create attempt.
	existing := domain.DiscoveryRun{ID: "existing-run", CorrelationKey: key, Status: domain.DiscoveryStatusPending}
	runRepo.items[existing.ID] = existing
	runRepo.createErr = &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}

	run, started, err := svc.StartRun(context.Background(), key)
	if err != nil {
		t.Fatalf("StartRun error: %v", err)
	}
	if started {
		t.Error("started = true, want false (race must resolve to the existing run)")
	}
	if run.ID != existing.ID {
		t.Errorf("run.ID = %q, want %q", run.ID, existing.ID)
	}
}

func TestDiscoveryService_ExecuteRun_Success_UpdatesLifecycleAndFoundCount(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Tender A", BuyerName: "Buyer A"},
	}}
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := successScoreJSON(90)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, _, runRepo := newTestDiscoveryService(collector, stub)

	run, _, err := svc.StartRun(context.Background(), nil)
	if err != nil {
		t.Fatalf("StartRun error: %v", err)
	}

	if err := svc.ExecuteRun(context.Background(), run.ID); err != nil {
		t.Fatalf("ExecuteRun error: %v", err)
	}

	final := runRepo.items[run.ID]
	if final.Status != domain.DiscoveryStatusSuccess {
		t.Errorf("final.Status = %q, want %q", final.Status, domain.DiscoveryStatusSuccess)
	}
	if final.FoundCount != 1 {
		t.Errorf("final.FoundCount = %d, want 1", final.FoundCount)
	}
	if final.FinishedAt == nil {
		t.Error("final.FinishedAt is nil, want set after execution")
	}
	if final.Summary == nil || *final.Summary == "" {
		t.Error("final.Summary is empty, want a human-readable summary")
	}
}

func TestDiscoveryService_ExecuteRun_CollectFails_MarksFailed(t *testing.T) {
	collector := &fakeCandidateCollector{err: errors.New("crawl unreachable")}
	svc, _, runRepo := newTestDiscoveryService(collector, &stubHermesClient{})

	run, _, err := svc.StartRun(context.Background(), nil)
	if err != nil {
		t.Fatalf("StartRun error: %v", err)
	}

	if err := svc.ExecuteRun(context.Background(), run.ID); err != nil {
		t.Fatalf("ExecuteRun should not itself return an error on pipeline failure, got: %v", err)
	}

	final := runRepo.items[run.ID]
	if final.Status != domain.DiscoveryStatusFailed {
		t.Errorf("final.Status = %q, want %q (nothing saved, nothing deduped, only errors)", final.Status, domain.DiscoveryStatusFailed)
	}
	if final.FoundCount != 0 {
		t.Errorf("final.FoundCount = %d, want 0", final.FoundCount)
	}
}

func TestDiscoveryService_ExecuteRun_PartialFailure_StillSuccess(t *testing.T) {
	collector := &fakeCandidateCollector{candidates: []ai.CandidateTender{
		{Title: "Fails"},
		{Title: "Succeeds"},
	}}
	callCount := 0
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("hermes transient error")
			}
			raw := successScoreJSON(70)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, _, runRepo := newTestDiscoveryService(collector, stub)

	run, _, err := svc.StartRun(context.Background(), nil)
	if err != nil {
		t.Fatalf("StartRun error: %v", err)
	}
	if err := svc.ExecuteRun(context.Background(), run.ID); err != nil {
		t.Fatalf("ExecuteRun error: %v", err)
	}

	final := runRepo.items[run.ID]
	if final.Status != domain.DiscoveryStatusSuccess {
		t.Errorf("final.Status = %q, want %q (2 tenders saved despite 1 scoring error)", final.Status, domain.DiscoveryStatusSuccess)
	}
	if final.FoundCount != 2 {
		t.Errorf("final.FoundCount = %d, want 2", final.FoundCount)
	}
}
