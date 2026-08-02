package strategy

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	generated  atomic.Uint64
	longEntry  atomic.Uint64
	shortEntry atomic.Uint64
	exits      atomic.Uint64
	holds      atomic.Uint64
	lastEvent  atomic.Value // time.Time
}

func (h *healthSnapshot) record(d StrategyDecision) {
	h.generated.Add(1)
	switch d.Decision {
	case LongEntry:
		h.longEntry.Add(1)
	case ShortEntry:
		h.shortEntry.Add(1)
	case LongExit, ShortExit:
		h.exits.Add(1)
	case Hold:
		h.holds.Add(1)
	}
	h.lastEvent.Store(d.Timestamp)
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64) health.Report {
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
		Message:       "strategy decision engine",
		Details: map[string]string{
			"enabled":             boolString(cfg.Enabled),
			"strategies_enabled":  joinStrings(cfg.EnabledStrategies()),
			"decisions_generated": u64String(h.generated.Load()),
			"long_entries":        u64String(h.longEntry.Load()),
			"short_entries":       u64String(h.shortEntry.Load()),
			"exits":               u64String(h.exits.Load()),
			"holds":               u64String(h.holds.Load()),
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

func joinStrings(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	out := vals[0]
	for i := 1; i < len(vals); i++ {
		out += "," + vals[i]
	}
	return out
}
