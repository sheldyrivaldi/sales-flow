package hermestui

import (
	"context"
	"sync"
	"time"
)

// sessionEntry is live-lifecycle state for one Hermes TUI session, held only
// in memory. It is deliberately separate from the persisted DB audit row
// (domain.HermesTuiSession) — this registry answers "is this session open
// right now", the DB answers "what sessions happened, when, for how long".
// A session may span several transient WS connections (ttyd auto-reconnects
// on any network blip), so activeConns/lastActivity are tracked independent
// of any single connection's lifetime.
type sessionEntry struct {
	userID       string
	startedAt    time.Time
	lastActivity time.Time
	activeConns  int
	ctx          context.Context
	cancel       context.CancelFunc
}

// EndedSession is what SweepExpired reports so the caller can persist the
// end (repo.End) for each session it reaped.
type EndedSession struct {
	ID     string
	UserID string
}

// Registry tracks live Hermes TUI sessions. A session is created once, at
// ticket-exchange time, and closed exactly once — by an explicit end, an
// idle-timeout sweep, or a hard-cap sweep — never by a WS connection merely
// dropping (see plan §Full connection lifecycle, step 10).
type Registry struct {
	mu       sync.Mutex
	sessions map[string]*sessionEntry
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]*sessionEntry)}
}

// Open registers a new session under id and returns a context that is
// cancelled when the session is closed (explicit end, idle sweep, or
// hard-cap sweep). The session is created once (at ticket-exchange time,
// "enter") but its context is retrieved later — by Context — from a
// different request entirely (the WS upgrade), since those are two separate
// HTTP round-trips.
func (r *Registry) Open(id, userID string) context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	r.sessions[id] = &sessionEntry{
		userID:       userID,
		startedAt:    now,
		lastActivity: now,
		ctx:          ctx,
		cancel:       cancel,
	}
	return ctx
}

// Context returns the session's context (see Open) for a live WS pump to
// select on, so an explicit end or a sweep can force it to exit.
func (r *Registry) Context(id string) (context.Context, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.sessions[id]
	if !ok {
		return nil, false
	}
	return e.ctx, true
}

// Touch records activity on id — called by the WS pump on every relayed
// frame in either direction. Idle-timeout sweeping is measured from this.
func (r *Registry) Touch(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.sessions[id]; ok {
		e.lastActivity = time.Now()
	}
}

// IncConn/DecConn track how many live WS connections are currently attached
// to a session (ttyd may reconnect multiple times across one session).
func (r *Registry) IncConn(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.sessions[id]; ok {
		e.activeConns++
	}
}

func (r *Registry) DecConn(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.sessions[id]; ok && e.activeConns > 0 {
		e.activeConns--
	}
}

// IsOpen reports whether id is still a live, unclosed session — this is what
// the cookie-gate middleware checks on every proxied request.
func (r *Registry) IsOpen(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.sessions[id]
	return ok
}

// Close ends session id: cancels its context (forcing any live WS pump to
// exit) and removes it from the registry. Safe to call more than once — a
// second call is a no-op and returns false.
func (r *Registry) Close(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.sessions[id]
	if !ok {
		return false
	}
	e.cancel()
	delete(r.sessions, id)
	return true
}

// SweepExpired reaps sessions whose idle time exceeds idle or whose total
// age exceeds hardCap, cancelling each one's context and removing it from
// the registry (same effect as Close), and returns them for the caller to
// persist via repo.End.
func (r *Registry) SweepExpired(idle, hardCap time.Duration) []EndedSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var ended []EndedSession
	for id, e := range r.sessions {
		if now.Sub(e.lastActivity) > idle || now.Sub(e.startedAt) > hardCap {
			e.cancel()
			ended = append(ended, EndedSession{ID: id, UserID: e.userID})
			delete(r.sessions, id)
		}
	}
	return ended
}
