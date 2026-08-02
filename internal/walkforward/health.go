package walkforward

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	reports   atomic.Uint64
	lastEvent atomic.Value // time.Time
}

func (h *healthSnapshot) recordReport() {
	h.reports.Add(1)
	h.lastEvent.Store(time.Now().UTC())
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

	activeWindows := 0
	if scheduler != nil {
		activeWindows = scheduler.ActiveWindows()
	}

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     connected,
		LastEventTime: last,
		Message:       "walk-forward analysis engine",
		Details: map[string]string{
			"enabled":             boolString(cfg.Enabled),
			"windows_created":     u64String(uint64(cache.windowsCreated())),
			"windows_completed":   u64String(uint64(cache.windowsCompleted())),
			"active_windows":      u64String(uint64(activeWindows)),
			"validation_runs":     u64String(uint64(cache.validationRuns())),
			"reports_generated":   u64String(h.reports.Load()),
			"dropped":             u64String(dropped),
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
