// Package hermestui backs the admin-only Hermes TUI feature: a ticket ->
// session-cookie handoff plus an in-memory session registry, sitting in
// front of a reverse proxy to the hermes-tui/ttyd sidecar. See the feature
// plan for the full connection lifecycle and security boundary.
package hermestui

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// TicketTTL is how long a minted ticket remains valid and consumable.
const TicketTTL = 30 * time.Second

type ticketEntry struct {
	userID  string
	expires time.Time
}

// TicketStore issues short-lived, single-use tickets that bridge an
// authenticated (JWT Bearer) request to a plain browser navigation, which
// cannot carry a custom Authorization header. A ticket proves "an
// already-authenticated admin requested this" for exactly one exchange.
type TicketStore struct {
	mu      sync.Mutex
	tickets map[string]ticketEntry
}

func NewTicketStore() *TicketStore {
	return &TicketStore{tickets: make(map[string]ticketEntry)}
}

// Issue mints a new single-use ticket for userID, valid for TicketTTL.
func (s *TicketStore) Issue(userID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()

	id := uuid.NewString()
	s.tickets[id] = ticketEntry{userID: userID, expires: time.Now().Add(TicketTTL)}
	return id
}

// Consume validates and atomically deletes ticketID, returning the userID it
// was issued for. A ticket is consumed at most once — even looking up an
// expired ticket deletes it, so a replayed ticket ID never succeeds twice.
func (s *TicketStore) Consume(ticketID string) (userID string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.tickets[ticketID]
	delete(s.tickets, ticketID)
	if !found {
		return "", false
	}
	if time.Now().After(entry.expires) {
		return "", false
	}
	return entry.userID, true
}

// sweepLocked drops expired tickets opportunistically. Caller must hold s.mu.
func (s *TicketStore) sweepLocked() {
	now := time.Now()
	for id, e := range s.tickets {
		if now.After(e.expires) {
			delete(s.tickets, id)
		}
	}
}
