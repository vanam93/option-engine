package alerts

import (
	"fmt"
	"sync"
	"time"
)

type fingerprintKey struct {
	recommendationID string
	alertType        AlertType
	currentStatus    Status
	reason           string
}

// Cache stores deduplication state and alert ID sequencing.
type Cache struct {
	mu sync.Mutex

	cooldown time.Duration

	seenRecommendations map[string]struct{}
	lastEmitted         map[fingerprintKey]time.Time
	seqBySymbolDate     map[string]uint64
}

// NewCache creates an empty alert deduplication cache.
func NewCache(cooldown time.Duration) *Cache {
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}
	return &Cache{
		cooldown:            cooldown,
		seenRecommendations: make(map[string]struct{}),
		lastEmitted:         make(map[fingerprintKey]time.Time),
		seqBySymbolDate:     make(map[string]uint64),
	}
}

// MarkSeen records a recommendation ID and reports whether this is the first observation.
func (c *Cache) MarkSeen(recommendationID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.seenRecommendations[recommendationID]; ok {
		return false
	}
	c.seenRecommendations[recommendationID] = struct{}{}
	return true
}

// ShouldEmit reports whether an alert fingerprint is outside the cooldown window.
func (c *Cache) ShouldEmit(update StateUpdate, alertType AlertType, reason string, at time.Time) bool {
	key := fingerprintKey{
		recommendationID: update.RecommendationID,
		alertType:        alertType,
		currentStatus:    update.CurrentStatus,
		reason:           reason,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if last, ok := c.lastEmitted[key]; ok {
		if at.Sub(last) < c.cooldown {
			return false
		}
	}
	c.lastEmitted[key] = at
	return true
}

// NextAlertID generates a unique alert identifier.
func (c *Cache) NextAlertID(symbol string, at time.Time) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	dateKey := fmt.Sprintf("%s:%s", at.UTC().Format("20060102"), symbol)
	c.seqBySymbolDate[dateKey]++
	seq := c.seqBySymbolDate[dateKey]
	return fmt.Sprintf("ALT-%s-%s-%06d", at.UTC().Format("20060102"), symbol, seq)
}
