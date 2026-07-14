package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"salespilot/internal/domain"
)

type fakeRepo struct {
	mu      sync.Mutex
	created []*domain.TelemetryEvent
	failErr error
}

func (f *fakeRepo) Create(_ context.Context, e *domain.TelemetryEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failErr != nil {
		return f.failErr
	}
	f.created = append(f.created, e)
	return nil
}

func (f *fakeRepo) CountByEvent(_ context.Context, event string, since time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, e := range f.created {
		if e.Event == event && !e.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) snapshot() []*domain.TelemetryEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*domain.TelemetryEvent, len(f.created))
	copy(out, f.created)
	return out
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout menunggu kondisi")
}

func TestEmit_Success(t *testing.T) {
	repo := &fakeRepo{}
	e := NewEmitter(repo)

	e.Emit(context.Background(), "chat_opened", map[string]any{"user": "u1"})

	waitFor(t, func() bool { return len(repo.snapshot()) == 1 })

	rec := repo.snapshot()[0]
	if rec.Event != "chat_opened" {
		t.Fatalf("event = %q, want chat_opened", rec.Event)
	}
	var props map[string]any
	if err := json.Unmarshal(rec.Props, &props); err != nil {
		t.Fatalf("unmarshal props: %v", err)
	}
	if props["user"] != "u1" {
		t.Fatalf("props[user] = %v, want u1", props["user"])
	}
}

func TestEmit_PromotesActorFromProps(t *testing.T) {
	repo := &fakeRepo{}
	e := NewEmitter(repo)

	e.Emit(context.Background(), "review_pursue", map[string]any{"actor": "user-42", "tender_id": "t-1"})

	waitFor(t, func() bool { return len(repo.snapshot()) == 1 })

	rec := repo.snapshot()[0]
	if rec.Actor == nil || *rec.Actor != "user-42" {
		t.Fatalf("Actor = %v, want \"user-42\"", rec.Actor)
	}
	var props map[string]any
	if err := json.Unmarshal(rec.Props, &props); err != nil {
		t.Fatalf("unmarshal props: %v", err)
	}
	if _, stillPresent := props["actor"]; stillPresent {
		t.Error("actor should be removed from props once promoted to the Actor column")
	}
	if props["tender_id"] != "t-1" {
		t.Fatalf("props[tender_id] = %v, want t-1", props["tender_id"])
	}
}

func TestEmit_ActorOnlyPropsLeavesPropsNil(t *testing.T) {
	repo := &fakeRepo{}
	e := NewEmitter(repo)

	e.Emit(context.Background(), "chat_opened", map[string]any{"actor": "user-1"})

	waitFor(t, func() bool { return len(repo.snapshot()) == 1 })

	rec := repo.snapshot()[0]
	if rec.Actor == nil || *rec.Actor != "user-1" {
		t.Fatalf("Actor = %v, want \"user-1\"", rec.Actor)
	}
	if rec.Props != nil {
		t.Fatalf("Props = %v, want nil once actor is the only key", rec.Props)
	}
}

func TestEmit_NilProps(t *testing.T) {
	repo := &fakeRepo{}
	e := NewEmitter(repo)

	e.Emit(context.Background(), "outcome_recorded", nil)

	waitFor(t, func() bool { return len(repo.snapshot()) == 1 })
	if repo.snapshot()[0].Props != nil {
		t.Fatalf("props = %v, want nil", repo.snapshot()[0].Props)
	}
}

func TestEmit_RepoErrorDoesNotPanic(t *testing.T) {
	repo := &fakeRepo{failErr: errors.New("db down")}
	e := NewEmitter(repo)

	// Should not panic; Emit has no return value to check, so we just
	// verify the call returns promptly and no created records exist.
	e.Emit(context.Background(), "chat_opened", map[string]any{"user": "u1"})
	time.Sleep(50 * time.Millisecond)

	if len(repo.snapshot()) != 0 {
		t.Fatalf("expected no records created on repo error, got %d", len(repo.snapshot()))
	}
}

func TestCountByEvent(t *testing.T) {
	repo := &fakeRepo{}
	now := time.Now()
	repo.created = []*domain.TelemetryEvent{
		{Event: "chat_opened", CreatedAt: now},
		{Event: "chat_opened", CreatedAt: now.Add(-time.Hour)},
		{Event: "outcome_recorded", CreatedAt: now},
	}

	count, err := repo.CountByEvent(context.Background(), "chat_opened", now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("CountByEvent: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
