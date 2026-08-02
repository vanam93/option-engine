package research

import "sync"

// Cache tracks lightweight runtime state for active report generation.
type Cache struct {
	mu     sync.Mutex
	active map[string]ActiveReport
}

// NewCache creates runtime research state storage.
func NewCache() *Cache {
	return &Cache{
		active: make(map[string]ActiveReport),
	}
}

// MarkActive registers an in-flight report for an experiment.
func (c *Cache) MarkActive(report ActiveReport) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active[report.ExperimentID] = report
}

// ClearActive removes an in-flight report marker.
func (c *Cache) ClearActive(experimentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.active, experimentID)
}

// ActiveJobs returns the number of in-flight report generations.
func (c *Cache) ActiveJobs() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.active)
}

// IsActive reports whether a report is being generated for an experiment.
func (c *Cache) IsActive(experimentID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.active[experimentID]
	return ok
}
