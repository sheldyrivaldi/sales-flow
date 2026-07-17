package hermestui

import (
	"testing"
	"time"
)

func TestRegistry_OpenIsOpenClose(t *testing.T) {
	r := NewRegistry()
	ctx := r.Open("sess-1", "user-1")

	if !r.IsOpen("sess-1") {
		t.Fatal("expected session to be open after Open")
	}
	if got, ok := r.Context("sess-1"); !ok || got != ctx {
		t.Fatal("expected Context to return the same context Open returned")
	}

	if !r.Close("sess-1") {
		t.Fatal("expected Close to report success for an open session")
	}
	if r.IsOpen("sess-1") {
		t.Fatal("expected session to be closed after Close")
	}
	if _, ok := r.Context("sess-1"); ok {
		t.Fatal("expected Context to fail after Close")
	}

	select {
	case <-ctx.Done():
		// expected: Close must cancel the session's context.
	default:
		t.Fatal("expected Close to cancel the session's context")
	}
}

func TestRegistry_CloseIsIdempotent(t *testing.T) {
	r := NewRegistry()
	r.Open("sess-1", "user-1")

	if !r.Close("sess-1") {
		t.Fatal("first Close should succeed")
	}
	if r.Close("sess-1") {
		t.Fatal("second Close on an already-closed session should return false, not panic")
	}
}

func TestRegistry_CloseUnknownSession(t *testing.T) {
	r := NewRegistry()
	if r.Close("never-opened") {
		t.Fatal("expected Close of an unknown session to return false")
	}
}

func TestRegistry_ContextUnknownSession(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Context("never-opened"); ok {
		t.Fatal("expected Context of an unknown session to return false")
	}
}

func TestRegistry_ConnCounting(t *testing.T) {
	r := NewRegistry()
	r.Open("sess-1", "user-1")

	r.IncConn("sess-1")
	r.IncConn("sess-1")
	r.DecConn("sess-1")

	r.mu.Lock()
	got := r.sessions["sess-1"].activeConns
	r.mu.Unlock()
	if got != 1 {
		t.Fatalf("activeConns = %d, want 1", got)
	}

	// DecConn below zero must not go negative.
	r.DecConn("sess-1")
	r.DecConn("sess-1")
	r.mu.Lock()
	got = r.sessions["sess-1"].activeConns
	r.mu.Unlock()
	if got != 0 {
		t.Fatalf("activeConns after over-decrementing = %d, want 0", got)
	}
}

func TestRegistry_Touch(t *testing.T) {
	r := NewRegistry()
	r.Open("sess-1", "user-1")

	r.mu.Lock()
	r.sessions["sess-1"].lastActivity = time.Now().Add(-time.Hour)
	before := r.sessions["sess-1"].lastActivity
	r.mu.Unlock()

	r.Touch("sess-1")

	r.mu.Lock()
	after := r.sessions["sess-1"].lastActivity
	r.mu.Unlock()
	if !after.After(before) {
		t.Fatal("expected Touch to advance lastActivity")
	}
}

func TestRegistry_SweepExpired(t *testing.T) {
	r := NewRegistry()

	idleCtx := r.Open("idle-session", "user-a")
	hardCtx := r.Open("hard-session", "user-b")
	freshCtx := r.Open("fresh-session", "user-c")

	now := time.Now()
	r.mu.Lock()
	r.sessions["idle-session"].lastActivity = now.Add(-time.Hour) // stale activity, within hard cap
	r.sessions["idle-session"].startedAt = now.Add(-time.Hour)
	r.sessions["hard-session"].startedAt = now.Add(-5 * time.Hour) // past hard cap, active recently
	r.sessions["hard-session"].lastActivity = now
	// fresh-session left with the defaults Open() set: both recent.
	r.mu.Unlock()

	ended := r.SweepExpired(30*time.Minute, 4*time.Hour)

	endedIDs := map[string]bool{}
	for _, e := range ended {
		endedIDs[e.ID] = true
	}
	if !endedIDs["idle-session"] {
		t.Error("expected idle-session to be reaped (idle timeout)")
	}
	if !endedIDs["hard-session"] {
		t.Error("expected hard-session to be reaped (hard cap)")
	}
	if endedIDs["fresh-session"] {
		t.Error("fresh-session should not be reaped")
	}
	if r.IsOpen("idle-session") || r.IsOpen("hard-session") {
		t.Error("reaped sessions should no longer be open in the registry")
	}
	if !r.IsOpen("fresh-session") {
		t.Error("fresh-session should still be open")
	}

	select {
	case <-idleCtx.Done():
	default:
		t.Error("expected idle-session's context to be cancelled by the sweep")
	}
	select {
	case <-hardCtx.Done():
	default:
		t.Error("expected hard-session's context to be cancelled by the sweep")
	}
	select {
	case <-freshCtx.Done():
		t.Error("fresh-session's context should NOT be cancelled")
	default:
	}
}
