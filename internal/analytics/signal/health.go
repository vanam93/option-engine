package signal

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	generated atomic.Uint64
	buy       atomic.Uint64
	sell      atomic.Uint64
	neutral   atomic.Uint64
	lastEvent atomic.Value // time.Time
}

func (h *healthSnapshot) record(sig GeneratedSignal) {
	h.generated.Add(1)
	switch sig.Signal {
	case Buy, ExitShort:
		h.buy.Add(1)
	case Sell, ExitLong:
		h.sell.Add(1)
	case Neutral:
		h.neutral.Add(1)
	}
	h.lastEvent.Store(sig.Timestamp)
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
		Message:       "signal evaluation engine",
		Details: map[string]string{
			"enabled":           boolString(cfg.Enabled),
			"signals_generated": u64String(h.generated.Load()),
			"buy_count":         u64String(h.buy.Load()),
			"sell_count":        u64String(h.sell.Load()),
			"neutral_count":     u64String(h.neutral.Load()),
			"active_rules":      joinStrings(cfg.ActiveRules()),
			"dropped":           u64String(dropped),
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
