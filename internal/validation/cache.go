package validation

import "sync"

type symbolKey struct {
	symbol    string
	timeframe string
}

// Cache stores the latest validation state per symbol and timeframe.
type Cache struct {
	mu      sync.RWMutex
	entries map[symbolKey]ValidatedRecommendation
}

// NewCache creates an empty validation cache.
func NewCache() *Cache {
	return &Cache{entries: make(map[symbolKey]ValidatedRecommendation)}
}

// Put stores the latest validation for a symbol/timeframe pair.
func (c *Cache) Put(result ValidatedRecommendation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := symbolKey{symbol: result.Symbol, timeframe: result.Timeframe}
	c.entries[key] = result
}

// Get returns the latest validation for a symbol/timeframe pair.
func (c *Cache) Get(symbol, timeframe string) (ValidatedRecommendation, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, ok := c.entries[symbolKey{symbol: symbol, timeframe: timeframe}]
	return result, ok
}

// Count returns the number of cached validation entries.
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
