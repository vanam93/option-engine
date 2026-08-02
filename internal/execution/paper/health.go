package paper

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	received  atomic.Uint64
	filled    atomic.Uint64
	rejected  atomic.Uint64
	lastEvent atomic.Value // time.Time
}

func (h *healthSnapshot) record(report ExecutionReport) {
	h.received.Add(1)
	if report.Status == Filled {
		h.filled.Add(1)
	} else {
		h.rejected.Add(1)
	}
	h.lastEvent.Store(report.Timestamp)
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, activePositions int) health.Report {
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
		Message:       "paper execution engine",
		Details: map[string]string{
			"enabled":           boolString(cfg.Enabled),
			"orders_received":     u64String(h.received.Load()),
			"orders_filled":       u64String(h.filled.Load()),
			"orders_rejected":     u64String(h.rejected.Load()),
			"active_positions":    u64String(uint64(activePositions)),
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
