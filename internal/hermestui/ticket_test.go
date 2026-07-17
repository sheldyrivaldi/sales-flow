package hermestui

import (
	"testing"
	"time"
)

func TestTicketStore_IssueThenConsume(t *testing.T) {
	s := NewTicketStore()
	id := s.Issue("user-1")
	if id == "" {
		t.Fatal("Issue returned empty ticket ID")
	}

	userID, ok := s.Consume(id)
	if !ok {
		t.Fatal("expected Consume to succeed for a fresh ticket")
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want %q", userID, "user-1")
	}
}

func TestTicketStore_SingleUse(t *testing.T) {
	s := NewTicketStore()
	id := s.Issue("user-1")

	if _, ok := s.Consume(id); !ok {
		t.Fatal("first Consume should succeed")
	}
	if _, ok := s.Consume(id); ok {
		t.Fatal("second Consume of the same ticket should fail (single-use)")
	}
}

func TestTicketStore_UnknownTicket(t *testing.T) {
	s := NewTicketStore()
	if _, ok := s.Consume("does-not-exist"); ok {
		t.Fatal("expected Consume of an unknown ticket ID to fail")
	}
}

func TestTicketStore_ExpiredTicket(t *testing.T) {
	s := NewTicketStore()
	id := s.Issue("user-1")

	// White-box: force expiry without a real sleep — TicketTTL is 30s and
	// this test must stay fast.
	s.mu.Lock()
	s.tickets[id] = ticketEntry{userID: "user-1", expires: time.Now().Add(-time.Second)}
	s.mu.Unlock()

	if _, ok := s.Consume(id); ok {
		t.Fatal("expected Consume of an expired ticket to fail")
	}
}

func TestTicketStore_IssueSweepsExpiredEntries(t *testing.T) {
	s := NewTicketStore()
	stale := s.Issue("stale-user")
	s.mu.Lock()
	s.tickets[stale] = ticketEntry{userID: "stale-user", expires: time.Now().Add(-time.Minute)}
	s.mu.Unlock()

	// Issue triggers sweepLocked internally.
	s.Issue("fresh-user")

	s.mu.Lock()
	_, staleStillPresent := s.tickets[stale]
	s.mu.Unlock()
	if staleStillPresent {
		t.Error("expected stale ticket to be swept on next Issue")
	}
}
