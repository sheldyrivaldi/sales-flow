package service

import "testing"

func TestDefaultAggregate(t *testing.T) {
	agg := defaultAggregate("PT Contoh")

	if agg.Profile.CompanyName != "PT Contoh" {
		t.Errorf("CompanyName = %q, want %q", agg.Profile.CompanyName, "PT Contoh")
	}
	if agg.Target == nil {
		t.Fatal("Target is nil")
	}
	if agg.Target.ValueMin == nil || *agg.Target.ValueMin != 1_000_000_000 {
		t.Errorf("ValueMin = %v, want 1000000000", agg.Target.ValueMin)
	}
	if agg.Target.DeadlineMinDays == nil || *agg.Target.DeadlineMinDays != 7 {
		t.Errorf("DeadlineMinDays = %v, want 7", agg.Target.DeadlineMinDays)
	}
	if len(agg.Target.Countries) != 1 || agg.Target.Countries[0] != "Indonesia" {
		t.Errorf("Countries = %v, want [Indonesia]", agg.Target.Countries)
	}
	if len(agg.Target.ProcurementTypes) == 0 {
		t.Error("ProcurementTypes should have a preset, got empty")
	}
	if agg.NoGo == nil {
		t.Fatal("NoGo is nil")
	}
}
