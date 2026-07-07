package domain

import "testing"

func TestProspectStage_Valid(t *testing.T) {
	tests := []struct {
		stage ProspectStage
		want  bool
	}{
		{ProspectStageNew, true},
		{ProspectStageQualified, true},
		{ProspectStageEngaged, true},
		{ProspectStageProposal, true},
		{ProspectStageWon, true},
		{ProspectStageLost, true},
		{"INVALID", false},
		{"", false},
		{"new", false},
	}
	for _, tt := range tests {
		if got := tt.stage.Valid(); got != tt.want {
			t.Errorf("ProspectStage(%q).Valid() = %v, want %v", tt.stage, got, tt.want)
		}
	}
}

func TestProspectSource_Valid(t *testing.T) {
	tests := []struct {
		source ProspectSource
		want   bool
	}{
		{ProspectSourceManual, true},
		{ProspectSourceEvent, true},
		{ProspectSourceTender, true},
		{"INVALID", false},
		{"", false},
		{"MANUAL", false},
	}
	for _, tt := range tests {
		if got := tt.source.Valid(); got != tt.want {
			t.Errorf("ProspectSource(%q).Valid() = %v, want %v", tt.source, got, tt.want)
		}
	}
}
