package montecarlo

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	started        atomic.Uint64
	completed      atomic.Uint64
	reports        atomic.Uint64
	totalRuntimeMs atomic.Uint64
	lastEvent      atomic.Value // time.Time
}

func (h *healthSnapshot) recordStarted() {
	h.started.Add(1)
}

func (h *healthSnapshot) recordCompleted(duration time.Duration) {
	h.completed.Add(1)
	h.totalRuntimeMs.Add(uint64(duration.Milliseconds()))
	h.lastEvent.Store(time.Now().UTC())
}

func (h *healthSnapshot) recordReport() {
	h.reports.Add(1)
}

func (h *healthSnapshot) averageRuntimeMs() uint64 {
	completed := h.completed.Load()
	if completed == 0 {
		return 0
	}
	return h.totalRuntimeMs.Load() / completed
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, cache *Cache) health.Report {
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

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     connected,
		LastEventTime: last,
		Message:       "monte carlo simulation engine",
		Details: map[string]string{
			"enabled":               boolString(cfg.Enabled),
			"simulations_started":   u64String(h.started.Load()),
			"simulations_completed": u64String(h.completed.Load()),
			"reports_generated":     u64String(h.reports.Load()),
			"active_jobs":           u64String(uint64(cache.activeJobs())),
			"average_runtime_ms":    u64String(h.averageRuntimeMs()),
			"dropped":               u64String(dropped),
			"completed_simulations": u64String(uint64(cache.completedCount())),
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
