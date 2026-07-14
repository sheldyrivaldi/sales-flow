// Package telemetry implements EP-17 ST-17.1: recording in-app metric
// events (chat_opened, review_pursue, scoring_generated, report_generated,
// outcome_recorded) so PRD §2 metrics are queryable without an external
// analytics stack.
package telemetry

import (
	"context"
	"encoding/json"
	"log"

	"salespilot/internal/domain"
)

// Emitter writes telemetry events. Emit is fire-and-forget — like
// ai.LearningHermes, a failure to record a metric must never affect the
// request that triggered it (PRD §8: AI/observability is always
// non-blocking), so it runs in its own goroutine with its own context and
// recovers from any panic.
type Emitter struct {
	repo domain.TelemetryRepository
}

func NewEmitter(repo domain.TelemetryRepository) *Emitter {
	return &Emitter{repo: repo}
}

// Emit records event with props asynchronously. props may be nil.
//
// If props contains an "actor" key holding a string, it is promoted to
// TelemetryEvent's dedicated actor column (and excluded from the stored
// props, so it isn't recorded twice) — callers like ChatHandler/TenderHandler
// pass the acting user's ID this way rather than through a separate
// parameter, so existing call sites didn't need to change when actor
// tracking was added. The caller's map is never mutated: the actor key is
// filtered into a copy only when present.
func (e *Emitter) Emit(ctx context.Context, event string, props map[string]any) {
	var actor string
	if a, ok := props["actor"]; ok {
		if s, ok := a.(string); ok {
			actor = s
			// Copy without "actor" rather than deleting from the caller's map.
			rest := make(map[string]any, len(props)-1)
			for k, v := range props {
				if k != "actor" {
					rest[k] = v
				}
			}
			props = rest
		}
	}

	var raw json.RawMessage
	if len(props) > 0 {
		b, err := json.Marshal(props)
		if err != nil {
			log.Printf("telemetry: gagal marshal props untuk event %q: %v", event, err)
			return
		}
		raw = b
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("telemetry: panic saat emit event %q: %v", event, r)
			}
		}()

		rec := &domain.TelemetryEvent{Event: event, Props: raw}
		if actor != "" {
			rec.Actor = &actor
		}
		if err := e.repo.Create(context.Background(), rec); err != nil {
			log.Printf("telemetry: gagal simpan event %q: %v", event, err)
		}
	}()
}
