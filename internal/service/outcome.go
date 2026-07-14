package service

import (
	"context"
	"fmt"

	"salespilot/internal/domain"
	"salespilot/internal/telemetry"
)

// recordOutcome creates an outcome_event row for the given target and result,
// then notifies the learning hook asynchronously (no-op until EP-16). Shared
// by TenderService and ProspectService so WON/LOST recording behaves
// identically regardless of which entity or entry point triggers it.
//
// emit may be nil (telemetry not wired, e.g. in tests) — Emit itself is
// nil-safe (EP-17 ST-17.1: metric recording must never affect this flow).
func recordOutcome(
	ctx context.Context,
	outcomes domain.OutcomeRepository,
	learn LearningHook,
	emit *telemetry.Emitter,
	targetType domain.OutcomeTargetType,
	targetID string,
	result domain.OutcomeResult,
	notes string,
) (*domain.OutcomeEvent, error) {
	oe := &domain.OutcomeEvent{
		TargetType: targetType,
		TargetID:   targetID,
		Result:     result,
	}
	if notes != "" {
		oe.Notes = &notes
	}

	if err := outcomes.Create(ctx, oe); err != nil {
		return nil, fmt.Errorf("recordOutcome: %w", err)
	}

	go learn.RecordOutcome(context.Background(), *oe)

	if emit != nil {
		emit.Emit(ctx, "outcome_recorded", map[string]any{
			"target_type": string(targetType),
			"result":      string(result),
		})
	}

	return oe, nil
}
