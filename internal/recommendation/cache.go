package recommendation

import "sync"

type symbolKey struct {
	symbol    string
	timeframe string
}

// Cache stores the latest recommendation per symbol and timeframe.
type Cache struct {
	mu              sync.RWMutex
	recommendations map[symbolKey]RecommendationUpdated
}

// NewCache creates an empty recommendation cache.
func NewCache() *Cache {
	return &Cache{recommendations: make(map[symbolKey]RecommendationUpdated)}
}

// Put stores the latest recommendation for a symbol/timeframe pair.
func (c *Cache) Put(rec RecommendationUpdated) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := symbolKey{symbol: rec.Symbol, timeframe: rec.Timeframe}
	c.recommendations[key] = rec
}

// Get returns the latest recommendation for a symbol/timeframe pair.
func (c *Cache) Get(symbol, timeframe string) (RecommendationUpdated, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	rec, ok := c.recommendations[symbolKey{symbol: symbol, timeframe: timeframe}]
	return rec, ok
}

// Count returns the number of cached recommendations.
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.recommendations)
}

// All returns a snapshot of all latest recommendations.
func (c *Cache) All() []RecommendationUpdated {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]RecommendationUpdated, 0, len(c.recommendations))
	for _, rec := range c.recommendations {
		out = append(out, rec)
	}
	return out
}
