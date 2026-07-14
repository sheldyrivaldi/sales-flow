package ai

import "salespilot/internal/domain"

// RecommendAction maps a fit score + no-go signals to a deterministic
// domain.RecommendedAction, per rubrik §8 (PRD.md):
//
//	80-100          -> Pursue
//	65-79           -> Review
//	50-64           -> Watchlist
//	<50             -> Reject
//	no-go triggered -> Reject (Auto No-Go), or Need Partner if the no-go is a
//	                   partner-closable gap — overrides the score-based tier.
//
// The LLM's own recommended_action suggestion (ScoreResult.RecommendedAction)
// is never persisted directly — it is advisory only. This function is the
// single source of truth so recommendations stay consistent regardless of
// model output (AC ST-10.2: "deterministik").
func RecommendAction(fitScore int, noGoTriggered, needPartner bool) domain.RecommendedAction {
	if noGoTriggered {
		if needPartner {
			return domain.ActionNeedPartner
		}
		return domain.ActionReject
	}

	switch {
	case fitScore >= 80:
		return domain.ActionPursue
	case fitScore >= 65:
		return domain.ActionReview
	case fitScore >= 50:
		return domain.ActionWatchlist
	default:
		return domain.ActionReject
	}
}
