package montecarlo

import "sync"

// Cache stores active and completed Monte Carlo simulations.
type Cache struct {
	mu sync.Mutex

	active    map[string]*SimulationRecord
	completed map[string]SimulationResult
}

// NewCache creates Monte Carlo state storage.
func NewCache() *Cache {
	return &Cache{
		active:    make(map[string]*SimulationRecord),
		completed: make(map[string]SimulationResult),
	}
}

// MarkActive registers a simulation as in progress.
func (c *Cache) MarkActive(simulationID, walkForwardID, experimentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active[simulationID] = &SimulationRecord{
		SimulationID:  simulationID,
		WalkForwardID: walkForwardID,
		ExperimentID:  experimentID,
		Status:        SimulationStatusActive,
	}
}

// StoreResult records a completed simulation.
func (c *Cache) StoreResult(result SimulationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.active, result.SimulationID)
	c.completed[result.SimulationID] = result
}

// MarkFailed removes a failed simulation from active tracking.
func (c *Cache) MarkFailed(simulationID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if record, ok := c.active[simulationID]; ok {
		record.Status = SimulationStatusFailed
		delete(c.active, simulationID)
	}
}

// Snapshot returns an immutable copy of Monte Carlo state.
func (c *Cache) Snapshot() StateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	active := make([]SimulationRecord, 0, len(c.active))
	for _, record := range c.active {
		copyRecord := *record
		active = append(active, copyRecord)
	}
	completed := make([]SimulationResult, 0, len(c.completed))
	for _, result := range c.completed {
		completed = append(completed, result)
	}
	return StateSnapshot{
		Active:    active,
		Completed: completed,
	}
}

func (c *Cache) activeJobs() int {
	return len(c.active)
}

func (c *Cache) completedCount() int {
	return len(c.completed)
}
