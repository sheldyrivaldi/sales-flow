package service

import (
	"context"
	"fmt"

	"salespilot/internal/domain"
)

// recordOutcome creates an outcome_event row for the given target and result,
// then notifies the learning hook asynchronously (no-op until EP-16). Shared
// by TenderService and ProspectService so WON/LOST recording behaves
// identically regardless of which entity or entry point triggers it.
func recordOutcome(
	ctx context.Context,
	outcomes domain.OutcomeRepository,
	learn LearningHook,
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

	return oe, nil
}
