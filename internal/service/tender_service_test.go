package service

import (
	"context"
	"testing"
	"time"

	"salespilot/internal/domain"
)

func TestTenderService_Review_DiscoveryOrigin_SetsReviewedAt(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {ID: "t1", Title: "Tender A", Origin: domain.OriginDiscovery, Status: domain.TenderStatusIdentified},
	}}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, NoopLearningHook())

	updated, err := svc.Review(context.Background(), "t1", "")
	if err != nil {
		t.Fatalf("Review error: %v", err)
	}
	if updated.ReviewedAt == nil {
		t.Error("ReviewedAt is nil, want set")
	}

	stored := repo.items["t1"]
	if stored.ReviewedAt == nil {
		t.Error("stored tender's ReviewedAt is nil, want persisted")
	}
	// Status/origin must be untouched by Review — only reviewed_at changes.
	if stored.Status != domain.TenderStatusIdentified {
		t.Errorf("Status = %q, want unchanged %q", stored.Status, domain.TenderStatusIdentified)
	}
}

func TestTenderService_Review_ManualOrigin_Rejected(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {ID: "t1", Title: "Tender A", Origin: domain.OriginManual, Status: domain.TenderStatusIdentified},
	}}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, NoopLearningHook())

	_, err := svc.Review(context.Background(), "t1", "")
	if err == nil {
		t.Fatal("Review should reject a manual-origin tender")
	}

	stored := repo.items["t1"]
	if stored.ReviewedAt != nil {
		t.Error("ReviewedAt should remain nil when Review is rejected")
	}
}

func TestTenderService_Review_NotFound(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{}}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, NoopLearningHook())

	_, err := svc.Review(context.Background(), "missing", "")
	if err == nil {
		t.Fatal("Review should error for a nonexistent tender id")
	}
}

// spyLearningHook records calls instead of doing anything real — used to
// verify TenderService.Review notifies the learning hook exactly when
// expected (EP-16 TK-16.2.1), without depending on a real Hermes client.
type spyLearningHook struct {
	outcomes        []domain.OutcomeEvent
	rejectTenderIDs []string
	rejectReasons   []string
}

func (h *spyLearningHook) RecordOutcome(_ context.Context, e domain.OutcomeEvent) {
	h.outcomes = append(h.outcomes, e)
}

func (h *spyLearningHook) RecordDiscoveryReject(_ context.Context, tenderID, reason string) {
	h.rejectTenderIDs = append(h.rejectTenderIDs, tenderID)
	h.rejectReasons = append(h.rejectReasons, reason)
}

func TestTenderService_Review_WithReason_NotifiesLearningHook(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {ID: "t1", Title: "Tender A", Origin: domain.OriginDiscovery, Status: domain.TenderStatusIdentified},
	}}
	hook := &spyLearningHook{}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, hook)

	if _, err := svc.Review(context.Background(), "t1", "Nilai terlalu kecil"); err != nil {
		t.Fatalf("Review error: %v", err)
	}

	// RecordDiscoveryReject is fired in a goroutine — give it a moment to run.
	deadline := time.Now().Add(time.Second)
	for len(hook.rejectTenderIDs) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	if len(hook.rejectTenderIDs) != 1 || hook.rejectTenderIDs[0] != "t1" {
		t.Errorf("rejectTenderIDs = %v, want [t1]", hook.rejectTenderIDs)
	}
	if len(hook.rejectReasons) != 1 || hook.rejectReasons[0] != "Nilai terlalu kecil" {
		t.Errorf("rejectReasons = %v, want [Nilai terlalu kecil]", hook.rejectReasons)
	}
}

func TestTenderService_RecordOutcome_EmitsTelemetry(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {ID: "t1", Title: "Tender A", Status: domain.TenderStatusBidding},
	}}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, NoopLearningHook())

	emitter, telemetryRepo := newTestEmitter()
	svc.SetEmitter(emitter)

	if _, err := svc.RecordOutcome(context.Background(), "t1", domain.OutcomeWon, "menang"); err != nil {
		t.Fatalf("RecordOutcome error: %v", err)
	}

	waitForEvents(t, telemetryRepo, "outcome_recorded", 1)
}

func TestTenderService_Review_WithoutReason_DoesNotNotifyLearningHook(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {ID: "t1", Title: "Tender A", Origin: domain.OriginDiscovery, Status: domain.TenderStatusIdentified},
	}}
	hook := &spyLearningHook{}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, hook)

	if _, err := svc.Review(context.Background(), "t1", ""); err != nil {
		t.Fatalf("Review error: %v", err)
	}

	// Give any (unwanted) async call a chance to land before asserting absence.
	time.Sleep(20 * time.Millisecond)

	if len(hook.rejectTenderIDs) != 0 {
		t.Errorf("rejectTenderIDs = %v, want none (Watchlist has no reason)", hook.rejectTenderIDs)
	}
}

// captureProjectRepo is a minimal domain.ProjectRepository that records every
// created project — used to assert the WON->Proyek bridge maps fields
// correctly without a real DB.
type captureProjectRepo struct {
	created []domain.Project
}

func (r *captureProjectRepo) Create(_ context.Context, p *domain.Project) error {
	if p.ID == "" {
		p.ID = "proj-generated"
	}
	r.created = append(r.created, *p)
	return nil
}
func (r *captureProjectRepo) Update(_ context.Context, _ *domain.Project) error { return nil }
func (r *captureProjectRepo) Delete(_ context.Context, _ string) error          { return nil }
func (r *captureProjectRepo) GetByID(_ context.Context, _ string) (*domain.Project, error) {
	return nil, nil
}
func (r *captureProjectRepo) List(_ context.Context, _ domain.ProjectFilter, _, _ int) ([]domain.Project, int64, error) {
	return nil, 0, nil
}

func TestTenderService_RecordOutcome_Won_CreatesLinkedProject(t *testing.T) {
	value := 5_000_000_000.0
	buyer := "Kementerian PU"
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {
			ID: "t1", Title: "Pembangunan Jembatan", BuyerName: &buyer,
			ValueEstimate: &value, Currency: "IDR", Status: domain.TenderStatusSubmitted,
		},
	}}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, NoopLearningHook())
	projRepo := &captureProjectRepo{}
	svc.SetProjectCreator(NewProjectService(projRepo))

	if _, err := svc.RecordOutcome(context.Background(), "t1", domain.OutcomeWon, "menang"); err != nil {
		t.Fatalf("RecordOutcome error: %v", err)
	}

	if len(projRepo.created) != 1 {
		t.Fatalf("created projects = %d, want 1", len(projRepo.created))
	}
	p := projRepo.created[0]
	if p.Name != "Pembangunan Jembatan" {
		t.Errorf("Name = %q, want tender title", p.Name)
	}
	if p.ClientName == nil || *p.ClientName != buyer {
		t.Errorf("ClientName = %v, want %q", p.ClientName, buyer)
	}
	if p.ContractValue == nil || *p.ContractValue != value {
		t.Errorf("ContractValue = %v, want %v", p.ContractValue, value)
	}
	if p.SourceType == nil || *p.SourceType != "tender" {
		t.Errorf("SourceType = %v, want \"tender\"", p.SourceType)
	}
	if p.SourceID == nil || *p.SourceID != "t1" {
		t.Errorf("SourceID = %v, want \"t1\"", p.SourceID)
	}
	if p.Status != domain.ProjectOnTrack {
		t.Errorf("Status = %q, want %q", p.Status, domain.ProjectOnTrack)
	}
}

func TestTenderService_RecordOutcome_Lost_CreatesNoProject(t *testing.T) {
	repo := &fakeScoreTenderRepo{items: map[string]domain.Tender{
		"t1": {ID: "t1", Title: "Tender A", Status: domain.TenderStatusSubmitted},
	}}
	svc := NewTenderService(repo, &fakeOutcomeRepo{}, NoopLearningHook())
	projRepo := &captureProjectRepo{}
	svc.SetProjectCreator(NewProjectService(projRepo))

	if _, err := svc.RecordOutcome(context.Background(), "t1", domain.OutcomeLost, "kalah"); err != nil {
		t.Fatalf("RecordOutcome error: %v", err)
	}

	if len(projRepo.created) != 0 {
		t.Errorf("created projects = %d, want 0 (LOST must not create a project)", len(projRepo.created))
	}
}
