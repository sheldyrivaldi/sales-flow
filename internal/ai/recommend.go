package ai

import "salespilot/internal/domain"

// defaultThresholdPursue/Review/Watchlist mirror scoring_config's DB defaults
// (0022_profile_enrichment migration) — used whenever cfg is nil, i.e. the
// profile has never touched the Scoring card.
const (
	defaultThresholdPursue    = 80
	defaultThresholdReview    = 65
	defaultThresholdWatchlist = 50
)

// RecommendAction maps a fit score + no-go signals to a deterministic
// domain.RecommendedAction, per rubrik §8 (PRD.md), using cfg's configured
// thresholds when present (RFI §8.2 Q48: "berapa score minimum agar tim
// wajib melakukan review manual") or the defaults above when cfg is nil:
//
//	>= threshold_pursue    -> Pursue
//	>= threshold_review    -> Review
//	>= threshold_watchlist -> Watchlist
//	otherwise              -> Reject
//	no-go triggered        -> Reject (Auto No-Go), or Need Partner if the
//	                          no-go is a partner-closable gap — overrides the
//	                          score-based tier.
//
// The LLM's own recommended_action suggestion (ScoreResult.RecommendedAction)
// is never persisted directly — it is advisory only. This function is the
// single source of truth so recommendations stay consistent regardless of
// model output (AC ST-10.2: "deterministik").
func RecommendAction(fitScore int, noGoTriggered, needPartner bool, cfg *domain.ScoringConfig) domain.RecommendedAction {
	if noGoTriggered {
		if needPartner {
			return domain.ActionNeedPartner
		}
		return domain.ActionReject
	}

	pursue, review, watchlist := defaultThresholdPursue, defaultThresholdReview, defaultThresholdWatchlist
	if cfg != nil {
		pursue, review, watchlist = cfg.ThresholdPursue, cfg.ThresholdReview, cfg.ThresholdWatchlist
	}

	switch {
	case fitScore >= pursue:
		return domain.ActionPursue
	case fitScore >= review:
		return domain.ActionReview
	case fitScore >= watchlist:
		return domain.ActionWatchlist
	default:
		return domain.ActionReject
	}
}
