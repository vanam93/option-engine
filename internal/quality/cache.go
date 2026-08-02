package quality

import "sync"

// Cache stores active recommendations, completed evaluations, and historical statistics.
type Cache struct {
	mu              sync.RWMutex
	active          map[string]QualityReport
	completed       map[string]QualityReport
	latest          map[string]QualityReport
	historical      aggregateHistoricalStats
}

// NewCache creates an empty quality cache.
func NewCache() *Cache {
	return &Cache{
		active:    make(map[string]QualityReport),
		completed: make(map[string]QualityReport),
		latest:    make(map[string]QualityReport),
	}
}

// UpdateActive stores the latest in-progress quality report.
func (c *Cache) UpdateActive(report QualityReport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active[report.RecommendationID] = report
	c.latest[report.RecommendationID] = report
}

// Complete moves a report from active to completed and updates historical stats.
func (c *Cache) Complete(report QualityReport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.active, report.RecommendationID)
	c.completed[report.RecommendationID] = report
	c.latest[report.RecommendationID] = report
	c.historical.add(report)
}

// GetLatest returns the latest quality report for a recommendation ID.
func (c *Cache) GetLatest(id string) (QualityReport, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	report, ok := c.latest[id]
	return report, ok
}

// GetCompleted returns a completed evaluation by recommendation ID.
func (c *Cache) GetCompleted(id string) (QualityReport, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	report, ok := c.completed[id]
	return report, ok
}

// Stats returns cache counters for health reporting.
func (c *Cache) Stats() (active, completed int, historical aggregateHistoricalStats) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.active), len(c.completed), c.historical
}
