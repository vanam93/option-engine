package backtestrunner

import (
	"sync"
	"time"
)

// Repository stores completed backtest sessions.
type Repository struct {
	mu       sync.RWMutex
	sessions map[string]Session
	order    []string
}

// NewRepository creates an empty session repository.
func NewRepository() *Repository {
	return &Repository{
		sessions: make(map[string]Session),
	}
}

// Save stores a completed session.
func (r *Repository) Save(session Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sessions[session.BacktestID]; !exists {
		r.order = append(r.order, session.BacktestID)
	}
	r.sessions[session.BacktestID] = session
}

// GetSession returns a session by ID.
func (r *Repository) GetSession(id string) (Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[id]
	return session, ok
}

// LatestSession returns the most recently completed session.
func (r *Repository) LatestSession() (Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.order) - 1; i >= 0; i-- {
		if session, ok := r.sessions[r.order[i]]; ok {
			return session, true
		}
	}
	return Session{}, false
}

// ListSessions returns all stored sessions in creation order.
func (r *Repository) ListSessions() []Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Session, 0, len(r.order))
	for _, id := range r.order {
		if session, ok := r.sessions[id]; ok {
			out = append(out, session)
		}
	}
	return out
}

// ListByDate returns sessions whose configured period overlaps the given day.
func (r *Repository) ListByDate(day time.Time) []Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	out := make([]Session, 0)
	for _, id := range r.order {
		session, ok := r.sessions[id]
		if !ok {
			continue
		}
		if session.StartTime.Before(dayEnd) && session.EndTime.After(dayStart) {
			out = append(out, session)
		}
	}
	return out
}

// ListBySymbol returns sessions that include the given symbol.
func (r *Repository) ListBySymbol(symbol string) []Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Session, 0)
	for _, id := range r.order {
		session, ok := r.sessions[id]
		if !ok {
			continue
		}
		for _, item := range session.Request.Symbols {
			if item == symbol {
				out = append(out, session)
				break
			}
		}
	}
	return out
}

// Count returns the number of stored sessions.
func (r *Repository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}
