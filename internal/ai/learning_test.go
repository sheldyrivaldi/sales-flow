package ai

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// syncedChatSpy records Chat calls behind a mutex — RecordOutcome/
// RecordDiscoveryReject are called synchronously in these tests (unlike
// TenderService.Review's `go learn.RecordDiscoveryReject(...)`), so no
// synchronization would strictly be required here, but the mutex keeps this
// robust against that assumption ever changing.
type syncedChatSpy struct {
	mu       sync.Mutex
	requests []hermes.ChatRequest
	err      error
}

func (s *syncedChatSpy) call(_ context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	if s.err != nil {
		return hermes.ChatResponse{}, s.err
	}
	return hermes.ChatResponse{Content: "ok"}, nil
}

func (s *syncedChatSpy) calls() []hermes.ChatRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]hermes.ChatRequest(nil), s.requests...)
}

func TestLearningHermes_RecordOutcome_SendsNoteWithSessionKey(t *testing.T) {
	spy := &syncedChatSpy{}
	stub := &stubHermesClient{chat: spy.call}
	hook := NewLearningHook(stub, "sk-workspace")

	notes := "Klien butuh waktu lebih lama"
	hook.RecordOutcome(context.Background(), domain.OutcomeEvent{
		TargetType: domain.OutcomeTargetTender,
		TargetID:   "t1",
		Result:     domain.OutcomeWon,
		Notes:      &notes,
	})

	calls := spy.calls()
	if len(calls) != 1 {
		t.Fatalf("Chat calls = %d, want 1", len(calls))
	}
	req := calls[0]
	if req.SessionKey != "sk-workspace" {
		t.Errorf("SessionKey = %q, want sk-workspace", req.SessionKey)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("Messages = %d, want 1", len(req.Messages))
	}
	content := req.Messages[0].Content
	for _, want := range []string{"t1", "WON", "Klien butuh waktu lebih lama"} {
		if !strings.Contains(content, want) {
			t.Errorf("note %q missing %q", content, want)
		}
	}
}

func TestLearningHermes_RecordDiscoveryReject_SendsNoteWithReason(t *testing.T) {
	spy := &syncedChatSpy{}
	stub := &stubHermesClient{chat: spy.call}
	hook := NewLearningHook(stub, "sk-workspace")

	hook.RecordDiscoveryReject(context.Background(), "t2", "Nilai terlalu kecil")

	calls := spy.calls()
	if len(calls) != 1 {
		t.Fatalf("Chat calls = %d, want 1", len(calls))
	}
	content := calls[0].Messages[0].Content
	for _, want := range []string{"t2", "Nilai terlalu kecil"} {
		if !strings.Contains(content, want) {
			t.Errorf("note %q missing %q", content, want)
		}
	}
}

func TestLearningHermes_RecordOutcome_HermesErrorDoesNotPanic(t *testing.T) {
	spy := &syncedChatSpy{err: errors.New("hermes down")}
	stub := &stubHermesClient{chat: spy.call}
	hook := NewLearningHook(stub, "sk-workspace")

	// Must not panic even though the underlying Chat call fails — the
	// method has no error return, so failure can only be logged.
	hook.RecordOutcome(context.Background(), domain.OutcomeEvent{
		TargetType: domain.OutcomeTargetProspect,
		TargetID:   "p1",
		Result:     domain.OutcomeLost,
	})

	if len(spy.calls()) != 1 {
		t.Fatalf("Chat calls = %d, want 1 (attempted despite eventual error)", len(spy.calls()))
	}
}

func TestLearningHermes_RecordDiscoveryReject_HermesErrorDoesNotPanic(t *testing.T) {
	spy := &syncedChatSpy{err: errors.New("hermes down")}
	stub := &stubHermesClient{chat: spy.call}
	hook := NewLearningHook(stub, "sk-workspace")

	hook.RecordDiscoveryReject(context.Background(), "t3", "alasan apapun")

	if len(spy.calls()) != 1 {
		t.Fatalf("Chat calls = %d, want 1", len(spy.calls()))
	}
}
