package feedback

import "sync"

// Cache is a thread-safe in-memory feedback snapshot store.
type Cache struct {
	mu         sync.RWMutex
	seen       map[string]struct{}
	aggregator *Aggregator
	latest     FeedbackSnapshot
	entries    int
}

// NewCache creates a feedback cache.
func NewCache(cfg Config) *Cache {
	return &Cache{
		seen:       make(map[string]struct{}),
		aggregator: NewAggregator(cfg),
	}
}

// Record applies a completed recommendation if it has not been seen before.
func (c *Cache) Record(input QualityInput, learner *Learner) (FeedbackSnapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.seen[input.RecommendationID]; exists {
		return c.latest, false
	}
	c.seen[input.RecommendationID] = struct{}{}

	snapshot := learner.Learn(input)
	c.latest = snapshot
	c.entries = len(c.seen)
	return snapshot, true
}

// Snapshot returns the latest feedback snapshot.
func (c *Cache) Snapshot() (FeedbackSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.entries == 0 {
		return FeedbackSnapshot{}, false
	}
	return c.latest, true
}

// Entries returns the number of unique recommendations tracked.
func (c *Cache) Entries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries
}

// Seen returns whether a recommendation has already been recorded.
func (c *Cache) Seen(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.seen[id]
	return ok
}

// Stats returns dimensional counts for health reporting.
func (c *Cache) Stats() (strategies, symbols, timeframes, recommendations int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.aggregator.Stats()
}

// BucketCount returns the number of confidence buckets.
func (c *Cache) BucketCount() int {
	return len(c.aggregator.buckets)
}

// RollingWindowCount returns the number of rolling windows.
func (c *Cache) RollingWindowCount() int {
	return len(c.aggregator.rollingWindows)
}
