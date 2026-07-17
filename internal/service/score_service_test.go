package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// --- In-memory fakes (no DB needed — mirrors internal/mcp/mcp_test.go's
// fake-repo pattern, scoped to what ScoreService actually needs). ---

type fakeScoreTenderRepo struct{ items map[string]domain.Tender }

func (r *fakeScoreTenderRepo) Create(_ context.Context, t *domain.Tender) error {
	if t.ID == "" {
		t.ID = fmt.Sprintf("tender-%d", len(r.items)+1)
	}
	r.items[t.ID] = *t
	return nil
}
func (r *fakeScoreTenderRepo) GetByID(_ context.Context, id string) (*domain.Tender, error) {
	t, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &t, nil
}
func (r *fakeScoreTenderRepo) List(_ context.Context, _ domain.TenderFilter, _, _ int) ([]domain.Tender, int64, error) {
	return nil, 0, nil
}
func (r *fakeScoreTenderRepo) Update(_ context.Context, t *domain.Tender) error {
	r.items[t.ID] = *t
	return nil
}
func (r *fakeScoreTenderRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}
func (r *fakeScoreTenderRepo) TopByFitScore(_ context.Context, _ int) ([]domain.Tender, error) {
	return nil, nil
}
func (r *fakeScoreTenderRepo) CountDiscoveryToday(_ context.Context) (int64, error) {
	return 0, nil
}
func (r *fakeScoreTenderRepo) GetByDedupKey(_ context.Context, key string) (*domain.Tender, error) {
	for _, t := range r.items {
		if t.DedupKey != nil && *t.DedupKey == key {
			return &t, nil
		}
	}
	return nil, nil
}

type fakeScoreProspectRepo struct{ items map[string]domain.Prospect }

func (r *fakeScoreProspectRepo) Create(_ context.Context, p *domain.Prospect) error {
	if p.ID == "" {
		p.ID = fmt.Sprintf("prospect-%d", len(r.items)+1)
	}
	r.items[p.ID] = *p
	return nil
}
func (r *fakeScoreProspectRepo) GetByID(_ context.Context, id string) (*domain.Prospect, error) {
	p, ok := r.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}
func (r *fakeScoreProspectRepo) GetBySource(_ context.Context, _ domain.ProspectSource, _ string) (*domain.Prospect, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeScoreProspectRepo) List(_ context.Context, _ domain.ProspectFilter, _, _ int) ([]domain.Prospect, int64, error) {
	return nil, 0, nil
}
func (r *fakeScoreProspectRepo) Update(_ context.Context, p *domain.Prospect) error {
	r.items[p.ID] = *p
	return nil
}
func (r *fakeScoreProspectRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}
func (r *fakeScoreProspectRepo) SummaryByStage(_ context.Context) ([]domain.ProspectStageSummary, error) {
	return nil, nil
}

type fakeScoreProfileRepo struct{ current *domain.ProfileAggregate }

func (r *fakeScoreProfileRepo) GetCurrent(_ context.Context) (*domain.ProfileAggregate, error) {
	if r.current == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.current, nil
}
func (r *fakeScoreProfileRepo) CreateVersion(_ context.Context, agg *domain.ProfileAggregate) (*domain.ProfileAggregate, error) {
	r.current = agg
	return agg, nil
}

type fakeOutcomeRepo struct{ events []domain.OutcomeEvent }

func (r *fakeOutcomeRepo) Create(_ context.Context, e *domain.OutcomeEvent) error {
	r.events = append(r.events, *e)
	return nil
}

type fakeProspectScoreRepo struct{ rows []domain.ProspectScore }

func (r *fakeProspectScoreRepo) Create(_ context.Context, s *domain.ProspectScore) error {
	if s.ID == "" {
		s.ID = fmt.Sprintf("score-%d", len(r.rows)+1)
	}
	r.rows = append(r.rows, *s)
	return nil
}
func (r *fakeProspectScoreRepo) GetLatest(_ context.Context, targetType, targetID string) (*domain.ProspectScore, error) {
	var latest *domain.ProspectScore
	for i := range r.rows {
		row := r.rows[i]
		if row.TargetType != targetType || row.TargetID != targetID {
			continue
		}
		if latest == nil {
			latest = &row
		} else {
			latest = &row // last match wins — rows appended in creation order
		}
	}
	return latest, nil
}

func newTestScoreService(stub *stubHermesClient) (*ScoreService, *fakeScoreTenderRepo, *fakeScoreProspectRepo, *fakeProspectScoreRepo) {
	tenderRepo := &fakeScoreTenderRepo{items: map[string]domain.Tender{}}
	prospectRepo := &fakeScoreProspectRepo{items: map[string]domain.Prospect{}}
	profileRepo := &fakeScoreProfileRepo{}
	scoreRepo := &fakeProspectScoreRepo{}

	tenderSvc := NewTenderService(tenderRepo, &fakeOutcomeRepo{}, NoopLearningHook())
	prospectSvc := NewProspectService(prospectRepo, &fakeOutcomeRepo{}, NoopLearningHook())
	profileSvc := NewProfileService(profileRepo, "", nil, nil)
	scorer := ai.NewScorer(stub, "sk-test")

	svc := NewScoreService(scorer, scoreRepo, tenderSvc, prospectSvc, profileSvc)
	return svc, tenderRepo, prospectRepo, scoreRepo
}

func TestScoreService_ScoreTarget_Tender_Success(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"fit_score":85,"recommended_action":"PURSUE","confidence":0.9,` +
				`"reasoning":"cocok kapabilitas","evidence":[{"dimension":"Capability fit","verdict":"pass","note":"ok"}],` +
				`"risk_flags":["deadline ketat"],"no_go_triggered":false,"need_partner":false}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _, scoreRepo := newTestScoreService(stub)

	tenderRepo.items["t1"] = domain.Tender{ID: "t1", Title: "Tender A", Status: domain.TenderStatusIdentified, Currency: "IDR"}

	score, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetTender, "t1")
	if err != nil {
		t.Fatalf("ScoreTarget error: %v", err)
	}
	if score.FitScore != 85 {
		t.Errorf("FitScore = %d, want 85", score.FitScore)
	}
	if score.RecommendedAction != string(domain.ActionPursue) {
		t.Errorf("RecommendedAction = %q, want %q", score.RecommendedAction, domain.ActionPursue)
	}
	if len(scoreRepo.rows) != 1 {
		t.Fatalf("prospect_score rows = %d, want 1", len(scoreRepo.rows))
	}

	updated := tenderRepo.items["t1"]
	if updated.FitScore == nil || *updated.FitScore != 85 {
		t.Errorf("tender.FitScore not denormalized: %v", updated.FitScore)
	}
	if updated.RecommendedAction == nil || *updated.RecommendedAction != domain.ActionPursue {
		t.Errorf("tender.RecommendedAction not denormalized: %v", updated.RecommendedAction)
	}
	if updated.ReasoningSummary == nil || *updated.ReasoningSummary != "cocok kapabilitas" {
		t.Errorf("tender.ReasoningSummary not denormalized: %v", updated.ReasoningSummary)
	}
}

func TestScoreService_ScoreTarget_Prospect_NoTenderDenormalization(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"fit_score":40,"recommended_action":"REJECT","confidence":0.6,` +
				`"reasoning":"kurang cocok","evidence":[],"risk_flags":[],"no_go_triggered":false,"need_partner":false}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, _, prospectRepo, scoreRepo := newTestScoreService(stub)
	prospectRepo.items["p1"] = domain.Prospect{ID: "p1", Name: "Prospek A", Stage: domain.ProspectStageNew}

	score, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetProspect, "p1")
	if err != nil {
		t.Fatalf("ScoreTarget error: %v", err)
	}
	if score.TargetType != string(ai.ScoreTargetProspect) {
		t.Errorf("TargetType = %q, want %q", score.TargetType, ai.ScoreTargetProspect)
	}
	if score.RecommendedAction != string(domain.ActionReject) {
		t.Errorf("RecommendedAction = %q, want %q (score 40 < 50)", score.RecommendedAction, domain.ActionReject)
	}
	if len(scoreRepo.rows) != 1 {
		t.Fatalf("prospect_score rows = %d, want 1", len(scoreRepo.rows))
	}
}

func TestScoreService_ScoreTarget_AIFailure_NoWriteAndTargetUntouched(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}
	svc, tenderRepo, _, scoreRepo := newTestScoreService(stub)
	original := domain.Tender{ID: "t1", Title: "Tender A", Status: domain.TenderStatusIdentified, Currency: "IDR"}
	tenderRepo.items["t1"] = original

	_, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetTender, "t1")
	if err == nil {
		t.Fatal("ScoreTarget should return an error when the AI call fails")
	}
	if len(scoreRepo.rows) != 0 {
		t.Errorf("prospect_score rows = %d, want 0 (AI failure must not persist a partial score)", len(scoreRepo.rows))
	}

	stillOriginal := tenderRepo.items["t1"]
	if stillOriginal.FitScore != nil || stillOriginal.RecommendedAction != nil {
		t.Errorf("tender was modified despite AI failure: fit_score=%v recommended_action=%v",
			stillOriginal.FitScore, stillOriginal.RecommendedAction)
	}
}

func TestScoreService_ScoreTarget_EmitsTelemetry(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"fit_score":85,"recommended_action":"PURSUE","confidence":0.9,` +
				`"reasoning":"cocok","evidence":[],"risk_flags":[],"no_go_triggered":false,"need_partner":false}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _, _ := newTestScoreService(stub)
	tenderRepo.items["t1"] = domain.Tender{ID: "t1", Title: "Tender A", Status: domain.TenderStatusIdentified, Currency: "IDR"}

	emitter, repo := newTestEmitter()
	svc.SetEmitter(emitter)

	if _, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetTender, "t1"); err != nil {
		t.Fatalf("ScoreTarget error: %v", err)
	}

	waitForEvents(t, repo, "scoring_generated", 1)
}

func TestScoreService_ScoreTarget_NotFound(t *testing.T) {
	stub := &stubHermesClient{}
	svc, _, _, _ := newTestScoreService(stub)

	_, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetTender, "missing")
	if err == nil {
		t.Fatal("ScoreTarget should error for a nonexistent tender id")
	}
}

func TestScoreService_GetLatestScore_NoneScored(t *testing.T) {
	stub := &stubHermesClient{}
	svc, _, _, _ := newTestScoreService(stub)

	score, err := svc.GetLatestScore(context.Background(), ai.ScoreTargetTender, "never-scored")
	if err != nil {
		t.Fatalf("GetLatestScore error: %v", err)
	}
	if score != nil {
		t.Errorf("GetLatestScore = %+v, want nil for a target that was never scored", score)
	}
}

func TestScoreService_GetLatestScore_ReturnsLatest(t *testing.T) {
	callCount := 0
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			callCount++
			score := 50 + callCount*10 // 60, then 70 on the second call
			raw := []byte(fmt.Sprintf(`{"fit_score":%d,"recommended_action":"REVIEW","confidence":0.5,`+
				`"reasoning":"r%d","evidence":[],"risk_flags":[],"no_go_triggered":false,"need_partner":false}`, score, callCount))
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _, scoreRepo := newTestScoreService(stub)
	tenderRepo.items["t1"] = domain.Tender{ID: "t1", Title: "Tender A", Status: domain.TenderStatusIdentified, Currency: "IDR"}

	if _, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetTender, "t1"); err != nil {
		t.Fatalf("first ScoreTarget error: %v", err)
	}
	if _, err := svc.ScoreTarget(context.Background(), ai.ScoreTargetTender, "t1"); err != nil {
		t.Fatalf("second ScoreTarget error: %v", err)
	}

	if len(scoreRepo.rows) != 2 {
		t.Fatalf("prospect_score rows = %d, want 2 (append-history)", len(scoreRepo.rows))
	}

	latest, err := svc.GetLatestScore(context.Background(), ai.ScoreTargetTender, "t1")
	if err != nil {
		t.Fatalf("GetLatestScore error: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatestScore = nil, want the second (latest) row")
	}
	if latest.FitScore != 70 {
		t.Errorf("GetLatestScore.FitScore = %d, want 70 (from the second, more recent, run)", latest.FitScore)
	}
}
