package clock

import (
	"sync"
	"time"
)

// ReplayClock advances programmatically for backtesting and replay sessions.
type ReplayClock struct {
	mu  sync.RWMutex
	now time.Time
}

// NewReplay creates a replay clock starting at the given instant.
func NewReplay(at time.Time) *ReplayClock {
	return &ReplayClock{now: at.UTC()}
}

func (c *ReplayClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *ReplayClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

func (c *ReplayClock) Until(t time.Time) time.Duration {
	return t.Sub(c.Now())
}

// Advance moves the replay clock forward by d.
func (c *ReplayClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Set moves the replay clock to an absolute instant.
func (c *ReplayClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t.UTC()
}
