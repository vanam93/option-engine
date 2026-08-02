package experiments

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	created   atomic.Uint64
	started   atomic.Uint64
	completed atomic.Uint64
	failed    atomic.Uint64
	lastEvent atomic.Value // time.Time
}

func (h *healthSnapshot) recordCreated(count int) {
	if count > 0 {
		h.created.Add(uint64(count))
	}
}

func (h *healthSnapshot) recordStarted() {
	h.started.Add(1)
}

func (h *healthSnapshot) recordCompleted() {
	h.completed.Add(1)
	h.lastEvent.Store(time.Now().UTC())
}

func (h *healthSnapshot) recordFailed() {
	h.failed.Add(1)
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, cache *Cache, scheduler *Scheduler) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}
	if dropped > 0 {
		status = health.StatusDegraded
	}

	var last *time.Time
	if v := h.lastEvent.Load(); v != nil {
		if t, ok := v.(time.Time); ok && !t.IsZero() {
			last = &t
		}
	}

	activeWorkers := 0
	queueDepth := 0
	if scheduler != nil {
		activeWorkers = scheduler.ActiveWorkers()
		queueDepth = scheduler.QueueDepth()
	}

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     connected,
		LastEventTime: last,
		Message:       "experiment and parameter sweep engine",
		Details: map[string]string{
			"enabled":             boolString(cfg.Enabled),
			"experiments_created": u64String(h.created.Load()),
			"runs_started":        u64String(h.started.Load()),
			"runs_completed":      u64String(h.completed.Load()),
			"runs_failed":         u64String(h.failed.Load()),
			"active_workers":      u64String(uint64(activeWorkers)),
			"queue_depth":         u64String(uint64(queueDepth)),
			"dropped":             u64String(dropped),
			"total_runs":          u64String(uint64(cache.runsCount())),
			"completed_results":   u64String(uint64(cache.completedCount())),
		},
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func u64String(v uint64) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
