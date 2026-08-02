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

// Cache stores deduplication state, alert history, and alert ID sequencing.
type Cache struct {
	mu sync.Mutex

	cooldown time.Duration
	maxHistory int

	seenRecommendations map[string]struct{}
	lastEmitted         map[fingerprintKey]time.Time
	seqBySymbolDate     map[string]uint64
	history             []AlertGenerated
}

// NewCache creates an empty alert deduplication cache.
func NewCache(cooldown time.Duration) *Cache {
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}
	return &Cache{
		cooldown:            cooldown,
		maxHistory:          10000,
		seenRecommendations: make(map[string]struct{}),
		lastEmitted:         make(map[fingerprintKey]time.Time),
		seqBySymbolDate:     make(map[string]uint64),
		history:             make([]AlertGenerated, 0, 256),
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

// Record stores a published alert in queryable history.
func (c *Cache) Record(alert AlertGenerated) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = append(c.history, alert)
	if len(c.history) > c.maxHistory {
		c.history = append([]AlertGenerated(nil), c.history[len(c.history)-c.maxHistory:]...)
	}
}

// List returns alerts matching optional filters.
func (c *Cache) List(symbol, strategy, timeframe, status string, confidenceMin float64) []AlertGenerated {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]AlertGenerated, 0, len(c.history))
	for _, alert := range c.history {
		if symbol != "" && alert.Symbol != symbol {
			continue
		}
		if timeframe != "" && alert.Timeframe != timeframe {
			continue
		}
		if status != "" && string(alert.CurrentStatus) != status {
			continue
		}
		if confidenceMin > 0 && alert.Confidence < confidenceMin {
			continue
		}
		_ = strategy
		out = append(out, alert)
	}
	return out
}
