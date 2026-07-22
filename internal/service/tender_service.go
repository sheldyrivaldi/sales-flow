package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/telemetry"
)

// tenderTransitions defines the set of valid target statuses for each current status.
var tenderTransitions = map[domain.TenderStatus][]domain.TenderStatus{
	domain.TenderStatusIdentified: {domain.TenderStatusQualifying, domain.TenderStatusLost},
	domain.TenderStatusQualifying: {domain.TenderStatusBidding, domain.TenderStatusLost},
	domain.TenderStatusBidding:    {domain.TenderStatusSubmitted, domain.TenderStatusWon, domain.TenderStatusLost},
	domain.TenderStatusSubmitted:  {domain.TenderStatusWon, domain.TenderStatusLost},
	domain.TenderStatusWon:        {},
	domain.TenderStatusLost:       {},
}

func canTransition(from, to domain.TenderStatus) bool {
	targets, ok := tenderTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// LearningHook is called after an outcome is recorded, or a discovery-origin
// tender is rejected from the inbox, so EP-16 can push a note to Hermes
// workspace memory. Both methods are fire-and-forget (no error return) —
// implementations must never let an AI/network failure propagate back into
// the CRUD flow that triggered them (PRD §8 non-blocking AI).
type LearningHook interface {
	RecordOutcome(ctx context.Context, e domain.OutcomeEvent)
	// RecordDiscoveryReject is called when a discovery-origin tender is
	// rejected from the Discovery Inbox with a reason (EP-12 ST-12.7
	// "Tolak") — reason is guaranteed non-empty by the caller.
	RecordDiscoveryReject(ctx context.Context, tenderID, reason string)
}

type noopLearningHook struct{}

func (noopLearningHook) RecordOutcome(_ context.Context, _ domain.OutcomeEvent)      {}
func (noopLearningHook) RecordDiscoveryReject(_ context.Context, _ string, _ string) {}

// NoopLearningHook returns a LearningHook that does nothing (used wherever a
// concrete workspace-memory hook — ai.LearningHermes — isn't wired, e.g. some
// test doubles).
func NoopLearningHook() LearningHook { return noopLearningHook{} }

// TenderService handles business logic for the tender entity.
type TenderService struct {
	repo     domain.TenderRepository
	outcomes domain.OutcomeRepository
	learn    LearningHook
	emit     *telemetry.Emitter
	// aiWarmup (nil-able, set via SetAIWarmup) dipanggil async saat tender
	// temuan AI DITERIMA dari Radar Tender — pre-generate konten AI
	// (playbook dsb) supaya halaman detail langsung terisi tanpa menunggu
	// user menekan tombol generate satu-satu.
	aiWarmup func(tenderID string)
	// projects (nil-able, set via SetProjectCreator) membuat baris Proyek
	// Berjalan saat tender di-mark WON — jembatan pra-deal (tender) ke
	// delivery (project). Nil-safe: bila tak diwire, WON hanya set status
	// tanpa membuat proyek (mis. beberapa test double).
	projects *ProjectService
}

func NewTenderService(repo domain.TenderRepository, outcomes domain.OutcomeRepository, learn LearningHook) *TenderService {
	return &TenderService{repo: repo, outcomes: outcomes, learn: learn}
}

// SetEmitter wires telemetry (EP-17 ST-17.1) after construction — optional,
// nil-safe, kept out of the constructor so existing call sites/tests are
// unaffected by observability being wired in or not.
func (s *TenderService) SetEmitter(e *telemetry.Emitter) { s.emit = e }

// Create creates a new tender from a create request.
func (s *TenderService) Create(ctx context.Context, req *dto.TenderCreateRequest) (*domain.Tender, error) {
	t := &domain.Tender{
		Title:    req.Title,
		Status:   domain.TenderStatusIdentified,
		Currency: "IDR",
		Origin:   domain.OriginManual,
	}

	// Apply optional fields.
	t.BuyerName = req.BuyerName
	t.BuyerCountry = req.BuyerCountry
	t.BuyerIndustry = req.BuyerIndustry
	t.ValueEstimate = req.ValueEstimate
	t.SourceName = req.SourceName
	t.SourceURL = req.SourceURL
	t.ServiceCategory = req.ServiceCategory
	t.ScopeSummary = req.ScopeSummary
	t.EligibilityRequirements = req.EligibilityRequirements
	t.TechnicalRequirements = req.TechnicalRequirements
	t.DedupKey = req.DedupKey

	if req.Currency != nil {
		t.Currency = *req.Currency
	}
	if req.Status != nil {
		s := domain.TenderStatus(*req.Status)
		if !s.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STATUS", "status tidak valid")
		}
		t.Status = s
	}
	if req.PublishedDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PublishedDate)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format published_date tidak valid, gunakan RFC3339")
		}
		t.PublishedDate = &parsed
	}
	if req.SubmissionDeadline != nil {
		parsed, err := time.Parse(time.RFC3339, *req.SubmissionDeadline)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format submission_deadline tidak valid, gunakan RFC3339")
		}
		t.SubmissionDeadline = &parsed
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Create: %w", err)
	}
	return t, nil
}

// CreateDiscovery persists a crawler-found candidate (EP-12) as a new tender
// with Origin=discovery, Status=IDENTIFIED — the only path that produces
// discovery-origin tenders (Create above always sets Origin=manual).
// DedupKey is computed from buyer+title+deadline (ai.ComputeDedupKey) and
// set on the row; the DB partial unique index on dedup_key is the last line
// of defense against a race with a concurrent duplicate insert (see
// DiscoveryService.persistCandidate, which checks-then-creates and treats a
// unique-violation here as "someone else just created the duplicate").
func (s *TenderService) CreateDiscovery(ctx context.Context, c ai.CandidateTender) (*domain.Tender, error) {
	t := &domain.Tender{
		Title:                   c.Title,
		BuyerName:               strPtrOrNil(c.BuyerName),
		BuyerCountry:            strPtrOrNil(c.BuyerCountry),
		BuyerIndustry:           strPtrOrNil(c.BuyerIndustry),
		ValueEstimate:           c.ValueEstimate,
		Currency:                "IDR",
		SubmissionDeadline:      c.SubmissionDeadline,
		SourceName:              strPtrOrNil(c.SourceName),
		SourceURL:               strPtrOrNil(c.SourceURL),
		ServiceCategory:         strPtrOrNil(c.ServiceCategory),
		ScopeSummary:            strPtrOrNil(c.ScopeSummary),
		EligibilityRequirements: strPtrOrNil(c.EligibilityRequirements),
		TechnicalRequirements:   strPtrOrNil(c.TechnicalRequirements),
		Status:                  domain.TenderStatusIdentified,
		Origin:                  domain.OriginDiscovery,
		DedupKey:                strPtrOrNil(ai.ComputeDedupKey(c.BuyerName, c.Title, c.SubmissionDeadline)),
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.CreateDiscovery: %w", err)
	}
	return t, nil
}

// FindByDedupKey looks up an existing tender by dedup key. Returns (nil,
// nil) when none exists, or when key is empty (nothing to look up) — both
// are the normal "not a duplicate" case, not an error.
func (s *TenderService) FindByDedupKey(ctx context.Context, key string) (*domain.Tender, error) {
	if key == "" {
		return nil, nil
	}
	t, err := s.repo.GetByDedupKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("tender.FindByDedupKey: %w", err)
	}
	return t, nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Get returns a tender by ID.
func (s *TenderService) Get(ctx context.Context, id string) (*domain.Tender, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("tender tidak ditemukan")
		}
		return nil, fmt.Errorf("tender.Get: %w", err)
	}
	return t, nil
}

// List returns paginated tenders matching the filter.
func (s *TenderService) List(ctx context.Context, f domain.TenderFilter, page, pageSize int) ([]domain.Tender, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.repo.List(ctx, f, page, pageSize)
}

// Update applies a partial update to a tender.
func (s *TenderService) Update(ctx context.Context, id string, req *dto.TenderUpdateRequest) (*domain.Tender, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.BuyerName != nil {
		t.BuyerName = req.BuyerName
	}
	if req.BuyerCountry != nil {
		t.BuyerCountry = req.BuyerCountry
	}
	if req.BuyerIndustry != nil {
		t.BuyerIndustry = req.BuyerIndustry
	}
	if req.ValueEstimate != nil {
		t.ValueEstimate = req.ValueEstimate
	}
	if req.Currency != nil {
		t.Currency = *req.Currency
	}
	if req.SourceName != nil {
		t.SourceName = req.SourceName
	}
	if req.SourceURL != nil {
		t.SourceURL = req.SourceURL
	}
	if req.ServiceCategory != nil {
		t.ServiceCategory = req.ServiceCategory
	}
	if req.ScopeSummary != nil {
		t.ScopeSummary = req.ScopeSummary
	}
	if req.EligibilityRequirements != nil {
		t.EligibilityRequirements = req.EligibilityRequirements
	}
	if req.TechnicalRequirements != nil {
		t.TechnicalRequirements = req.TechnicalRequirements
	}
	if req.Status != nil {
		newStatus := domain.TenderStatus(*req.Status)
		if !newStatus.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STATUS", "status tidak valid")
		}
		t.Status = newStatus
	}
	if req.PublishedDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PublishedDate)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format published_date tidak valid, gunakan RFC3339")
		}
		t.PublishedDate = &parsed
	}
	if req.SubmissionDeadline != nil {
		parsed, err := time.Parse(time.RFC3339, *req.SubmissionDeadline)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format submission_deadline tidak valid, gunakan RFC3339")
		}
		t.SubmissionDeadline = &parsed
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Update: %w", err)
	}
	return t, nil
}

// Delete removes a tender by ID.
func (s *TenderService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("tender.Delete: %w", err)
	}
	return nil
}

// UpdateStatus validates and applies a status transition.
func (s *TenderService) UpdateStatus(ctx context.Context, id string, to domain.TenderStatus) (*domain.Tender, error) {
	if !to.Valid() {
		return nil, httperr.NewBadRequest("INVALID_STATUS", "status tidak valid")
	}

	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if !canTransition(t.Status, to) {
		return nil, httperr.NewBadRequest(
			"INVALID_TRANSITION",
			fmt.Sprintf("transisi dari %s ke %s tidak diizinkan", t.Status, to),
		)
	}

	t.Status = to
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.UpdateStatus: %w", err)
	}
	return t, nil
}

// Promote moves a discovery-origin tender from inbox to the active pipeline
// by transitioning its status from IDENTIFIED to QUALIFYING.
func (s *TenderService) Promote(ctx context.Context, id string) (*domain.Tender, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if t.Origin != domain.OriginDiscovery {
		return nil, httperr.NewBadRequest("NOT_DISCOVERY", "hanya tender temuan AI yang bisa dipromosikan")
	}
	if t.Status != domain.TenderStatusIdentified {
		return nil, httperr.NewBadRequest("ALREADY_PROMOTED", "tender sudah keluar dari inbox discovery")
	}

	t.Status = domain.TenderStatusQualifying
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Promote: %w", err)
	}
	return t, nil
}

// Review marks a discovery-origin tender as reviewed without promoting it to
// the pipeline (EP-12 ST-12.7 Discovery Inbox "Watchlist"/"Tolak" actions —
// Pursue instead uses Promote above). Setting reviewed_at is what removes it
// from the inbox filter (origin=discovery AND status=IDENTIFIED AND
// reviewed_at IS NULL); status/AI fields are left untouched. reason is
// "" for Watchlist and non-empty for Tolak — a non-empty reason also
// notifies the learning hook (EP-16) so future discovery/scoring runs can
// learn from it; audit_log writing remains the caller's concern (see
// TenderHandler.Review).
func (s *TenderService) Review(ctx context.Context, id string, reason string) (*domain.Tender, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Origin != domain.OriginDiscovery {
		return nil, httperr.NewBadRequest("NOT_DISCOVERY", "hanya tender temuan AI yang bisa ditinjau dari inbox")
	}

	now := time.Now()
	t.ReviewedAt = &now
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Review: %w", err)
	}

	if reason != "" {
		go s.learn.RecordDiscoveryReject(context.Background(), t.ID, reason)
	} else if s.aiWarmup != nil {
		// reason kosong = DITERIMA (bukan Tolak) — pre-generate konten AI di
		// background. Best-effort: kegagalan generate tidak memengaruhi
		// review itu sendiri.
		go s.aiWarmup(t.ID)
	}

	return t, nil
}

// SetAIWarmup wires the accepted-from-Radar AI pre-generation hook —
// optional, nil-safe, out of the constructor (same pattern as SetEmitter).
func (s *TenderService) SetAIWarmup(fn func(tenderID string)) { s.aiWarmup = fn }

// SetProjectCreator wires the WON->Proyek Berjalan bridge (a tender that is
// won auto-creates a project). Optional, nil-safe, out of the constructor —
// same pattern as SetEmitter/SetAIWarmup, and needed because ProjectService is
// constructed after TenderService in the wiring graph (router.go).
func (s *TenderService) SetProjectCreator(p *ProjectService) { s.projects = p }

// SetScore denormalizes a fresh AI scoring result (EP-10) onto the tender
// row so list/detail views can read fit_score/recommended_action/risk_flags/
// reasoning_summary without joining prospect_score. Called by ScoreService
// after a prospect_score row has already been persisted — mirrors the
// existing non-atomic sequencing of RecordOutcome (outcome_event written
// before the tender row is updated).
func (s *TenderService) SetScore(ctx context.Context, id string, fitScore int, action domain.RecommendedAction, riskFlags json.RawMessage, reasoning string) (*domain.Tender, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	t.FitScore = &fitScore
	t.RecommendedAction = &action
	t.RiskFlags = riskFlags
	t.ReasoningSummary = &reasoning

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.SetScore: %w", err)
	}
	return t, nil
}

// RecordOutcome records a WON/LOST outcome, creates an outcome_event row,
// sets tender status to the terminal state, and notifies the learning hook.
func (s *TenderService) RecordOutcome(ctx context.Context, id string, result domain.OutcomeResult, notes string) (*domain.Tender, error) {
	if !result.Valid() {
		return nil, httperr.NewBadRequest("INVALID_RESULT", "result harus WON atau LOST")
	}

	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	targetStatus := domain.TenderStatusLost
	if result == domain.OutcomeWon {
		targetStatus = domain.TenderStatusWon
	}
	if !canTransition(t.Status, targetStatus) {
		return nil, httperr.NewBadRequest(
			"INVALID_TRANSITION",
			fmt.Sprintf("tidak bisa merekam outcome %s dari status %s", result, t.Status),
		)
	}

	if _, err := recordOutcome(ctx, s.outcomes, s.learn, s.emit, domain.OutcomeTargetTender, t.ID, result, notes); err != nil {
		return nil, fmt.Errorf("tender.RecordOutcome: %w", err)
	}

	t.Status = targetStatus
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.RecordOutcome update status: %w", err)
	}

	// Tender WON → jembatan ke Proyek Berjalan: buat baris project bertaut
	// (source_type=tender). Best-effort — kegagalan membuat proyek TIDAK
	// menggagalkan pencatatan outcome yang sudah tersimpan (prinsip
	// non-blocking, sama seperti learning hook). WON bersifat terminal
	// (canTransition tak punya target keluar), jadi RecordOutcome hanya sukses
	// sekali per tender → tak ada risiko proyek ganda.
	if result == domain.OutcomeWon && s.projects != nil {
		if err := s.createProjectFromTender(ctx, t); err != nil {
			log.Printf("tender.RecordOutcome: gagal membuat proyek dari tender %s (WON tetap tercatat): %v", t.ID, err)
		}
	}

	return t, nil
}

// createProjectFromTender memetakan tender yang menang menjadi Proyek Berjalan
// baru (menu Daftar Proyek). Field intelijen bid (fit_score, risk_flags, dsb.)
// sengaja tidak dibawa — proyek punya lifecycle sendiri (milestone, progress,
// kesehatan). Tautan asal disimpan lewat source_type/source_id.
func (s *TenderService) createProjectFromTender(ctx context.Context, t *domain.Tender) error {
	sourceType := "tender"
	p := &domain.Project{
		Name:          t.Title,
		ClientName:    t.BuyerName,
		ContractValue: t.ValueEstimate,
		Currency:      t.Currency,
		Status:        domain.ProjectOnTrack,
		SourceType:    &sourceType,
		SourceID:      &t.ID,
	}
	if _, err := s.projects.Create(ctx, p); err != nil {
		return fmt.Errorf("createProjectFromTender: %w", err)
	}
	return nil
}
