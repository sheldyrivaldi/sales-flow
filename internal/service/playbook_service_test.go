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

type fakePlaybookRepo struct{ rows []domain.Playbook }

func (r *fakePlaybookRepo) Create(_ context.Context, p *domain.Playbook) error {
	if p.ID == "" {
		p.ID = fmt.Sprintf("playbook-%d", len(r.rows)+1)
	}
	r.rows = append(r.rows, *p)
	return nil
}

func (r *fakePlaybookRepo) GetByID(_ context.Context, id string) (*domain.Playbook, error) {
	for _, p := range r.rows {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakePlaybookRepo) ListByTarget(_ context.Context, targetType, targetID string) ([]domain.Playbook, error) {
	var out []domain.Playbook
	for _, p := range r.rows {
		if p.TargetType == targetType && p.TargetID == targetID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakePlaybookRepo) GetLatestVersion(_ context.Context, targetType, targetID string) (int, error) {
	max := 0
	for _, p := range r.rows {
		if p.TargetType == targetType && p.TargetID == targetID && p.Version > max {
			max = p.Version
		}
	}
	return max, nil
}

func (r *fakePlaybookRepo) ListByTargetType(_ context.Context, targetType string) ([]domain.Playbook, error) {
	var out []domain.Playbook
	for _, p := range r.rows {
		if p.TargetType == targetType {
			out = append(out, p)
		}
	}
	return out, nil
}

func newTestPlaybookService(stub *stubHermesClient) (*PlaybookService, *fakeScoreTenderRepo, *fakeScoreProspectRepo, *fakePlaybookRepo) {
	tenderRepo := &fakeScoreTenderRepo{items: map[string]domain.Tender{}}
	prospectRepo := &fakeScoreProspectRepo{items: map[string]domain.Prospect{}}
	profileRepo := &fakeScoreProfileRepo{}
	playbookRepo := &fakePlaybookRepo{}

	tenderSvc := NewTenderService(tenderRepo, &fakeOutcomeRepo{}, NoopLearningHook())
	prospectSvc := NewProspectService(prospectRepo, &fakeOutcomeRepo{}, NoopLearningHook())
	profileSvc := NewProfileService(profileRepo, "", nil, nil)
	gen := ai.NewPlaybookGenerator(stub, "sk-test")

	svc := NewPlaybookService(gen, playbookRepo, tenderSvc, prospectSvc, profileSvc)
	return svc, tenderRepo, prospectRepo, playbookRepo
}

const samplePlaybookJSON = `{
	"summary": "Ringkasan peluang",
	"value_prop": "Value prop",
	"stakeholders": ["PIC A"],
	"strategy_checklist": ["Langkah 1"],
	"timeline": ["Minggu 1"],
	"risks": ["Risiko A"],
	"next_actions": ["Aksi 1"]
}`

func TestPlaybookService_Generate_Tender_Success(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(samplePlaybookJSON)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, tenderRepo, _, playbookRepo := newTestPlaybookService(stub)
	tenderRepo.items["t1"] = domain.Tender{ID: "t1", Title: "Tender A", Status: domain.TenderStatusIdentified, Currency: "IDR"}

	pb, err := svc.Generate(context.Background(), ai.ScoreTargetTender, "t1")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if pb.Version != 1 {
		t.Errorf("Version = %d, want 1", pb.Version)
	}
	if pb.TargetType != string(ai.ScoreTargetTender) || pb.TargetID != "t1" {
		t.Errorf("TargetType/TargetID = %s/%s, want tender/t1", pb.TargetType, pb.TargetID)
	}
	if len(playbookRepo.rows) != 1 {
		t.Fatalf("playbook rows = %d, want 1", len(playbookRepo.rows))
	}

	var content ai.PlaybookContent
	if err := json.Unmarshal(pb.Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if content.Summary != "Ringkasan peluang" {
		t.Errorf("content.Summary = %q, want %q", content.Summary, "Ringkasan peluang")
	}
}

func TestPlaybookService_Generate_SecondCall_IncrementsVersion(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(samplePlaybookJSON)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}
	svc, _, prospectRepo, playbookRepo := newTestPlaybookService(stub)
	prospectRepo.items["p1"] = domain.Prospect{ID: "p1", Name: "Prospek A", Stage: domain.ProspectStageNew}

	first, err := svc.Generate(context.Background(), ai.ScoreTargetProspect, "p1")
	if err != nil {
		t.Fatalf("first Generate error: %v", err)
	}
	second, err := svc.Generate(context.Background(), ai.ScoreTargetProspect, "p1")
	if err != nil {
		t.Fatalf("second Generate error: %v", err)
	}

	if first.Version != 1 {
		t.Errorf("first.Version = %d, want 1", first.Version)
	}
	if second.Version != 2 {
		t.Errorf("second.Version = %d, want 2", second.Version)
	}
	if len(playbookRepo.rows) != 2 {
		t.Fatalf("playbook rows = %d, want 2 (both versions kept)", len(playbookRepo.rows))
	}
	// The first row must remain exactly as it was — immutable history.
	if playbookRepo.rows[0].Version != 1 || playbookRepo.rows[0].ID != first.ID {
		t.Errorf("first row mutated: %+v", playbookRepo.rows[0])
	}
}

func TestPlaybookService_Generate_AIFailure_NoWrite(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}
	svc, tenderRepo, _, playbookRepo := newTestPlaybookService(stub)
	tenderRepo.items["t1"] = domain.Tender{ID: "t1", Title: "Tender A", Status: domain.TenderStatusIdentified, Currency: "IDR"}

	_, err := svc.Generate(context.Background(), ai.ScoreTargetTender, "t1")
	if err == nil {
		t.Fatal("expected error when Hermes fails, got nil")
	}
	if len(playbookRepo.rows) != 0 {
		t.Errorf("playbook rows = %d, want 0 (no partial write on AI failure)", len(playbookRepo.rows))
	}
}

func TestPlaybookService_Generate_TargetNotFound(t *testing.T) {
	stub := &stubHermesClient{}
	svc, _, _, playbookRepo := newTestPlaybookService(stub)

	_, err := svc.Generate(context.Background(), ai.ScoreTargetTender, "missing")
	if err == nil {
		t.Fatal("expected error for missing target, got nil")
	}
	if len(playbookRepo.rows) != 0 {
		t.Errorf("playbook rows = %d, want 0", len(playbookRepo.rows))
	}
}

func TestPlaybookService_ListByTarget_ReturnsAllVersions(t *testing.T) {
	svc, _, _, playbookRepo := newTestPlaybookService(&stubHermesClient{})
	playbookRepo.rows = []domain.Playbook{
		{ID: "pb1", TargetType: "tender", TargetID: "t1", Version: 1},
		{ID: "pb2", TargetType: "tender", TargetID: "t1", Version: 2},
		{ID: "pb3", TargetType: "tender", TargetID: "other", Version: 1},
	}

	items, err := svc.ListByTarget(context.Background(), ai.ScoreTargetTender, "t1")
	if err != nil {
		t.Fatalf("ListByTarget error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
}

func TestPlaybookService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _ := newTestPlaybookService(&stubHermesClient{})

	_, err := svc.GetByID(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}
