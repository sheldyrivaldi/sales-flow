package handlers

import (
	"context"
	"encoding/json"
	"log"

	"salespilot/internal/domain"
)

// writeAIAuditEvent best-effort records an audit_log row for one AI-generated
// output (EP-17 ST-17.2 TK-17.2.2: scoring/playbook/report) — sumber
// (actor), waktu (created_at), dan model/reasoning ringkas, alongside the
// existing crawl audit (internal/ai/discovery.go) and discovery_review audit
// (writeDiscoveryReviewAudit in tender_handler.go). A failure here must never
// fail the request — the AI output has already been persisted by the time
// this runs, mirroring the same best-effort pattern used everywhere else
// audit_log is written.
func writeAIAuditEvent(ctx context.Context, audit domain.AuditRepository, actor, action, targetType, targetID string, payload map[string]any) {
	if audit == nil {
		return
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("handlers: AUDIT FAILURE: marshal payload untuk %s (target=%s/%s): %v", action, targetType, targetID, err)
		return
	}
	e := &domain.AuditEvent{
		Actor:      actor,
		Action:     action,
		TargetType: &targetType,
		TargetID:   &targetID,
		Payload:    payloadJSON,
	}
	if err := audit.Create(ctx, e); err != nil {
		log.Printf("handlers: AUDIT FAILURE: gagal menulis audit_log %s (target=%s/%s): %v", action, targetType, targetID, err)
	}
}
