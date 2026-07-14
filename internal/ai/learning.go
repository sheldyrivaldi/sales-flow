// Package ai — learning.go implements EP-16 Continuous Learning:
// service.LearningHook, backed by writing short notes to Hermes workspace
// memory. Memory is written by simply calling Chat with the workspace
// SessionKey — the bridge's "chat" mode accumulates memory (skip_memory=
// false, see services/hermes-bridge/app/agent_factory.py); there is no
// separate "write to memory" API, Chat itself is the write path. The
// reverse — clearing memory — needs a dedicated method (hermes.Client.
// ResetMemory, TK-16.3.1) because there is no "forget everything" chat
// message that reliably does that.
package ai

import (
	"context"
	"fmt"
	"log"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// LearningHermes implements service.LearningHook. Both methods are
// fire-and-forget: they never return an error (the interface has none to
// return) and recover from any panic, because a memory-write failure must
// never be allowed to affect the CRUD flow that triggered it (PRD §8:
// AI is always non-blocking).
type LearningHermes struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewLearningHook(hc hermes.Client, sk hermes.SessionKey) *LearningHermes {
	return &LearningHermes{hc: hc, sk: sk}
}

func (h *LearningHermes) sendNote(ctx context.Context, note string, logCtx string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ai: learning hook panic (%s): %v", logCtx, r)
		}
	}()

	if _, err := h.hc.Chat(ctx, hermes.ChatRequest{
		Messages:   []hermes.Message{{Role: "user", Content: note}},
		Stream:     false,
		SessionKey: h.sk,
	}); err != nil {
		log.Printf("ai: learning hook: gagal kirim catatan ke memory Hermes (%s): %v", logCtx, err)
	}
}

// RecordOutcome sends a short Bahasa Indonesia note to workspace memory
// about a WON/LOST outcome. Called by service.recordOutcome via `go
// learn.RecordOutcome(context.Background(), ...)` — already detached from
// the original request context, so it keeps running even after the HTTP
// response is sent.
func (h *LearningHermes) RecordOutcome(ctx context.Context, e domain.OutcomeEvent) {
	note := fmt.Sprintf("Peluang %s (id=%s) hasil: %s.", e.TargetType, e.TargetID, e.Result)
	if e.Notes != nil && *e.Notes != "" {
		note += fmt.Sprintf(" Catatan: %s", *e.Notes)
	}
	h.sendNote(ctx, note, fmt.Sprintf("outcome %s/%s", e.TargetType, e.TargetID))
}

// RecordDiscoveryReject sends a short Bahasa Indonesia note to workspace
// memory about a discovery-origin tender rejected from the Discovery Inbox,
// so future discovery/scoring runs can learn what this workspace tends to
// reject and why (EP-12 ST-12.7 "Tolak").
func (h *LearningHermes) RecordDiscoveryReject(ctx context.Context, tenderID, reason string) {
	note := fmt.Sprintf("Tender %s ditolak dari Discovery Inbox. Alasan: %s", tenderID, reason)
	h.sendNote(ctx, note, fmt.Sprintf("discovery reject %s", tenderID))
}
