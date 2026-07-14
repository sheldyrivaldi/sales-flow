package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
	"salespilot/internal/telemetry"
)

// ScoreService orchestrates one AI scoring run end-to-end (EP-10): resolve
// the target + Company Profile, call the AI scorer, compute the
// deterministic recommended_action (ai.RecommendAction — never the LLM's own
// suggestion), persist an append-history prospect_score row, and — for
// tenders only — denormalize the result onto the tender row so list/detail
// views can read it without a join.
type ScoreService struct {
	scorer    *ai.Scorer
	scores    domain.ProspectScoreRepository
	tenders   *TenderService
	prospects *ProspectService
	profile   *ProfileService
	emit      *telemetry.Emitter
}

func NewScoreService(scorer *ai.Scorer, scores domain.ProspectScoreRepository, tenders *TenderService, prospects *ProspectService, profile *ProfileService) *ScoreService {
	return &ScoreService{scorer: scorer, scores: scores, tenders: tenders, prospects: prospects, profile: profile}
}

// SetEmitter wires telemetry (EP-17 ST-17.1) after construction — optional,
// nil-safe.
func (s *ScoreService) SetEmitter(e *telemetry.Emitter) { s.emit = e }

// ScoreTarget runs a fresh AI analysis for a tender or prospect and persists
// it as a new prospect_score row. On any AI failure (Hermes down, invalid
// output after GenerateJSON's own retry) it returns a friendly error and
// leaves both the target row and prospect_score table untouched — AI must
// stay non-blocking to CRUD (PRD §8: "gagal AI → pesan ramah, data utuh").
func (s *ScoreService) ScoreTarget(ctx context.Context, targetType ai.ScoreTargetType, id string) (*domain.ProspectScore, error) {
	profile, err := s.profile.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("score.ScoreTarget: profile: %w", err)
	}

	var in ai.ScoreInput
	switch targetType {
	case ai.ScoreTargetTender:
		t, err := s.tenders.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		in = ai.ScoreInputFromTender(*t, profile)
	case ai.ScoreTargetProspect:
		p, err := s.prospects.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		in = ai.ScoreInputFromProspect(*p, profile)
	default:
		return nil, httperr.NewBadRequest("INVALID_TARGET_TYPE", "target_type harus tender atau prospect")
	}

	start := time.Now()
	res, err := s.scorer.Score(ctx, in)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Analisa AI sedang tidak tersedia, coba lagi nanti")
	}
	duration := time.Since(start)

	action := ai.RecommendAction(res.FitScore, res.NoGoTriggered, res.NeedPartner)

	evidenceJSON, err := json.Marshal(res.Evidence)
	if err != nil {
		return nil, fmt.Errorf("score.ScoreTarget: marshal evidence: %w", err)
	}
	riskFlagsJSON, err := json.Marshal(res.RiskFlags)
	if err != nil {
		return nil, fmt.Errorf("score.ScoreTarget: marshal risk_flags: %w", err)
	}

	confidence := res.Confidence
	reasoning := res.Reasoning
	model := ai.ModelLabel

	score := &domain.ProspectScore{
		TargetType:        string(targetType),
		TargetID:          id,
		FitScore:          res.FitScore,
		RecommendedAction: string(action),
		Confidence:        &confidence,
		Reasoning:         &reasoning,
		Evidence:          evidenceJSON,
		RiskFlags:         riskFlagsJSON,
		Model:             &model,
	}
	if err := s.scores.Create(ctx, score); err != nil {
		return nil, fmt.Errorf("score.ScoreTarget: %w", err)
	}

	// Denormalize onto the tender row after the prospect_score row is
	// already durable — same non-atomic ordering already accepted for
	// outcome_event in TenderService.RecordOutcome. This second write is
	// best-effort: the scoring history row (already persisted above) is the
	// source of truth regardless of whether it lands, so a failure here must
	// not fail the whole ScoreTarget call — doing so would report the run as
	// failed (losing the audit/telemetry that follow) and risk a duplicate
	// prospect_score row on client retry, despite the score itself having
	// already succeeded.
	if targetType == ai.ScoreTargetTender {
		if _, err := s.tenders.SetScore(ctx, id, res.FitScore, action, riskFlagsJSON, res.Reasoning); err != nil {
			log.Printf("score.ScoreTarget: denormalize tender %s gagal (skor tetap tersimpan di riwayat): %v", id, err)
		}
	}

	if s.emit != nil {
		s.emit.Emit(ctx, "scoring_generated", map[string]any{
			"target_type": string(targetType),
			"duration_ms": duration.Milliseconds(),
		})
	}

	return score, nil
}

// GetLatestScore returns the most recent prospect_score row for a target, or
// (nil, nil) if it has never been scored — a normal state, not an error. It
// does not re-validate that the target itself still exists: a missing or
// garbage id simply yields no score, the same observable result as a
// valid-but-never-scored id.
func (s *ScoreService) GetLatestScore(ctx context.Context, targetType ai.ScoreTargetType, id string) (*domain.ProspectScore, error) {
	score, err := s.scores.GetLatest(ctx, string(targetType), id)
	if err != nil {
		return nil, fmt.Errorf("score.GetLatestScore: %w", err)
	}
	return score, nil
}
