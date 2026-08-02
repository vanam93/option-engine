package experiments

import (
	"sort"
	"sync"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

// Cache stores experiment runs, results, and rankings.
type Cache struct {
	mu sync.Mutex

	experimentID string
	runs         map[string]*ExperimentRun
	completed    map[string]RunResult
	rankings     []RunResult
	pending      map[string]struct{}
}

// NewCache creates experiment state storage.
func NewCache(experimentID string) *Cache {
	return &Cache{
		experimentID: experimentID,
		runs:         make(map[string]*ExperimentRun),
		completed:    make(map[string]RunResult),
		pending:      make(map[string]struct{}),
	}
}

// RegisterRuns stores generated experiment runs.
func (c *Cache) RegisterRuns(runs []ExperimentRun) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, run := range runs {
		copyRun := run
		copyRun.Status = RunStatusQueued
		c.runs[run.RunID] = &copyRun
	}
}

// MarkRunning marks a run as executing.
func (c *Cache) MarkRunning(runID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if run, ok := c.runs[runID]; ok {
		run.Status = RunStatusRunning
		c.pending[runID] = struct{}{}
	}
}

// MarkFailed marks a run as failed and removes it from pending.
func (c *Cache) MarkFailed(runID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if run, ok := c.runs[runID]; ok {
		run.Status = RunStatusFailed
	}
	delete(c.pending, runID)
}

// StoreResult records an optimization outcome and re-ranks completed runs.
func (c *Cache) StoreResult(runID string, score float64, metrics optimization.EvaluationMetrics, at time.Time) (RunResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	run, ok := c.runs[runID]
	if !ok {
		return RunResult{}, false
	}

	run.Status = RunStatusCompleted
	delete(c.pending, runID)

	result := RunResult{
		RunID:             runID,
		ExperimentID:      run.ExperimentID,
		Strategy:          run.Strategy,
		Parameters:        cloneParams(run.Parameters),
		OptimizationScore: score,
		Metrics:           metrics,
		CompletedAt:       at,
	}
	c.completed[runID] = result
	c.rebuildRankings()
	result.Rank = c.completed[runID].Rank
	return result, true
}

func (c *Cache) rebuildRankings() {
	rankings := make([]RunResult, 0, len(c.completed))
	for _, result := range c.completed {
		rankings = append(rankings, result)
	}
	sort.Slice(rankings, func(i, j int) bool {
		if rankings[i].OptimizationScore != rankings[j].OptimizationScore {
			return rankings[i].OptimizationScore > rankings[j].OptimizationScore
		}
		return rankings[i].RunID < rankings[j].RunID
	})
	for i := range rankings {
		rankings[i].Rank = i + 1
		c.completed[rankings[i].RunID] = rankings[i]
	}
	c.rankings = append([]RunResult(nil), rankings...)
}

// IsPending reports whether a run is awaiting optimization results.
func (c *Cache) IsPending(runID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.pending[runID]
	return ok
}

// Snapshot returns an immutable copy of experiment state.
func (c *Cache) Snapshot() StateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	runs := make([]ExperimentRun, 0, len(c.runs))
	for _, run := range c.runs {
		copyRun := *run
		runs = append(runs, copyRun)
	}
	completed := make([]RunResult, 0, len(c.completed))
	for _, result := range c.completed {
		completed = append(completed, result)
	}
	return StateSnapshot{
		ExperimentID: c.experimentID,
		Runs:         runs,
		Completed:    completed,
		Rankings:     append([]RunResult(nil), c.rankings...),
	}
}

func (c *Cache) experimentsCreated() int {
	return 1
}

func (c *Cache) runsCount() int {
	return len(c.runs)
}

func (c *Cache) completedCount() int {
	return len(c.completed)
}

func (c *Cache) failedCount() int {
	count := 0
	for _, run := range c.runs {
		if run.Status == RunStatusFailed {
			count++
		}
	}
	return count
}
