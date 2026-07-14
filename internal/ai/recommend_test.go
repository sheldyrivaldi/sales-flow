package ai

import (
	"testing"

	"salespilot/internal/domain"
)

func TestRecommendAction_ScoreThresholds(t *testing.T) {
	cases := []struct {
		score int
		want  domain.RecommendedAction
	}{
		{0, domain.ActionReject},
		{49, domain.ActionReject},
		{50, domain.ActionWatchlist},
		{64, domain.ActionWatchlist},
		{65, domain.ActionReview},
		{79, domain.ActionReview},
		{80, domain.ActionPursue},
		{100, domain.ActionPursue},
	}

	for _, c := range cases {
		got := RecommendAction(c.score, false, false)
		if got != c.want {
			t.Errorf("RecommendAction(%d, false, false) = %q, want %q", c.score, got, c.want)
		}
	}
}

func TestRecommendAction_NoGoOverridesScore(t *testing.T) {
	// Even a perfect score must be rejected when a no-go rule is triggered
	// and there is no partner-closable gap.
	got := RecommendAction(100, true, false)
	if got != domain.ActionReject {
		t.Errorf("RecommendAction(100, true, false) = %q, want %q (Auto No-Go)", got, domain.ActionReject)
	}
}

func TestRecommendAction_NoGoWithNeedPartner(t *testing.T) {
	got := RecommendAction(90, true, true)
	if got != domain.ActionNeedPartner {
		t.Errorf("RecommendAction(90, true, true) = %q, want %q", got, domain.ActionNeedPartner)
	}

	// needPartner alone (without no-go) must not trigger Need Partner —
	// only a no-go-triggered gap does.
	got = RecommendAction(90, false, true)
	if got != domain.ActionPursue {
		t.Errorf("RecommendAction(90, false, true) = %q, want %q (needPartner alone is a no-op)", got, domain.ActionPursue)
	}
}
