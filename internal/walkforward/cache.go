package walkforward

import (
	"sync"

	"github.com/vanam-gangireddy/option-engine/internal/experiments"
)

// Cache stores walk-forward windows, runs, and aggregated validation metrics.
type Cache struct {
	mu sync.Mutex

	walkForwardID string
	windows       map[int]*WindowRecord
	completed     map[int]WindowResult
	aggregated    AggregatedValidation
}

// NewCache creates walk-forward state storage.
func NewCache(walkForwardID string) *Cache {
	return &Cache{
		walkForwardID: walkForwardID,
		windows:       make(map[int]*WindowRecord),
		completed:     make(map[int]WindowResult),
		aggregated:    AggregatedValidation{ParameterDrift: map[string]float64{}},
	}
}

// RegisterWindows stores generated walk-forward windows.
func (c *Cache) RegisterWindows(windows []Window) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, window := range windows {
		copyWindow := window
		c.windows[window.Index] = &WindowRecord{
			Window: copyWindow,
			Status: WindowStatusQueued,
		}
	}
}

// MarkTraining marks a window as in training phase.
func (c *Cache) MarkTraining(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if record, ok := c.windows[index]; ok {
		record.Status = WindowStatusTraining
	}
}

// MarkValidating marks a window as in validation phase.
func (c *Cache) MarkValidating(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if record, ok := c.windows[index]; ok {
		record.Status = WindowStatusValidating
	}
}

// MarkFailed marks a window as failed.
func (c *Cache) MarkFailed(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if record, ok := c.windows[index]; ok {
		record.Status = WindowStatusFailed
	}
}

// StoreResult records a completed window and recomputes aggregated metrics.
func (c *Cache) StoreResult(result WindowResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if record, ok := c.windows[result.WindowIndex]; ok {
		record.Status = WindowStatusCompleted
	}
	c.completed[result.WindowIndex] = result
	c.rebuildAggregation()
}

func (c *Cache) rebuildAggregation() {
	completed := make([]WindowResult, 0, len(c.completed))
	for _, result := range c.completed {
		completed = append(completed, result)
	}
	c.aggregated = AggregateValidation(completed)
}

// Snapshot returns an immutable copy of walk-forward state.
func (c *Cache) Snapshot() StateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	windows := make([]WindowRecord, 0, len(c.windows))
	for _, record := range c.windows {
		copyRecord := *record
		windows = append(windows, copyRecord)
	}
	completed := make([]WindowResult, 0, len(c.completed))
	for _, result := range c.completed {
		completed = append(completed, result)
	}
	return StateSnapshot{
		WalkForwardID: c.walkForwardID,
		Windows:       windows,
		Completed:     completed,
		Aggregated:    c.aggregated,
	}
}

func (c *Cache) windowsCreated() int {
	return len(c.windows)
}

func (c *Cache) windowsCompleted() int {
	return len(c.completed)
}

func (c *Cache) activeWindows() int {
	count := 0
	for _, record := range c.windows {
		if record.Status == WindowStatusTraining || record.Status == WindowStatusValidating {
			count++
		}
	}
	return count
}

func (c *Cache) validationRuns() int {
	return len(c.completed)
}

// stripMetadata removes walk-forward correlation fields from parameters for reporting.
func stripMetadata(params experiments.ParameterSet) experiments.ParameterSet {
	out := experiments.ParameterSet{}
	for k, v := range params {
		switch k {
		case "run_id", "experiment_id", "walkforward_id", "window_index", "phase",
			"train_start", "train_end", "validation_start", "validation_end":
			continue
		default:
			out[k] = v
		}
	}
	return out
}
