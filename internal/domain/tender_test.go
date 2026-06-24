package domain

import "testing"

func TestTenderStatus_Valid(t *testing.T) {
	tests := []struct {
		status TenderStatus
		want   bool
	}{
		{TenderStatusIdentified, true},
		{TenderStatusQualifying, true},
		{TenderStatusBidding, true},
		{TenderStatusSubmitted, true},
		{TenderStatusWon, true},
		{TenderStatusLost, true},
		{"INVALID", false},
		{"", false},
		{"identified", false},
		{"won", false},
	}
	for _, tt := range tests {
		if got := tt.status.Valid(); got != tt.want {
			t.Errorf("TenderStatus(%q).Valid() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestRecommendedAction_Valid(t *testing.T) {
	tests := []struct {
		action RecommendedAction
		want   bool
	}{
		{ActionPursue, true},
		{ActionReview, true},
		{ActionWatchlist, true},
		{ActionReject, true},
		{ActionNeedPartner, true},
		{"INVALID", false},
		{"", false},
		{"pursue", false},
		{"PARTNER", false},
	}
	for _, tt := range tests {
		if got := tt.action.Valid(); got != tt.want {
			t.Errorf("RecommendedAction(%q).Valid() = %v, want %v", tt.action, got, tt.want)
		}
	}
}

func TestTenderOrigin_Valid(t *testing.T) {
	tests := []struct {
		origin TenderOrigin
		want   bool
	}{
		{OriginManual, true},
		{OriginDiscovery, true},
		{"MANUAL", false},
		{"", false},
		{"ai", false},
	}
	for _, tt := range tests {
		if got := tt.origin.Valid(); got != tt.want {
			t.Errorf("TenderOrigin(%q).Valid() = %v, want %v", tt.origin, got, tt.want)
		}
	}
}
