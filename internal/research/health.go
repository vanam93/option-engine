package research

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	reportsGenerated atomic.Uint64
	exportsCompleted atomic.Uint64
	exportFailures   atomic.Uint64
	postgresWrites   atomic.Uint64
	readLatencyMs    atomic.Uint64
	lastEvent        atomic.Value // time.Time
}

func (h *healthSnapshot) recordWrite() {
	h.postgresWrites.Add(1)
}

func (h *healthSnapshot) recordRead(latency time.Duration) {
	h.readLatencyMs.Store(uint64(latency.Milliseconds()))
}

func (h *healthSnapshot) recordReport() {
	h.reportsGenerated.Add(1)
	h.lastEvent.Store(time.Now().UTC())
}

func (h *healthSnapshot) recordExport(success bool) {
	if success {
		h.exportsCompleted.Add(1)
	} else {
		h.exportFailures.Add(1)
	}
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, repo Repository, cache *Cache) health.Report {
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

	entries := int64(0)
	if repo != nil {
		if count, err := repo.CountEntries(context.Background()); err == nil {
			entries = count
		}
	}

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     connected,
		LastEventTime: last,
		Message:       "research repository and reporting engine",
		Details: map[string]string{
			"enabled":                  boolString(cfg.Enabled),
			"repository_entries":       i64String(entries),
			"reports_generated":        u64String(h.reportsGenerated.Load()),
			"exports_completed":        u64String(h.exportsCompleted.Load()),
			"export_failures":          u64String(h.exportFailures.Load()),
			"postgres_writes":          u64String(h.postgresWrites.Load()),
			"postgres_read_latency_ms": u64String(h.readLatencyMs.Load()),
			"active_jobs":              u64String(uint64(cache.ActiveJobs())),
			"dropped":                  u64String(dropped),
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
	return i64String(int64(v))
}

func i64String(v int64) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	if negative {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
