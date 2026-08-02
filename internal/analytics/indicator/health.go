package indicator

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

// healthSnapshot collects runtime counters for observability.
type healthSnapshot struct {
	processed atomic.Uint64
	published atomic.Uint64
	lastEvent atomic.Value // time.Time
}

func (h *healthSnapshot) recordProcessed(at time.Time) {
	h.processed.Add(1)
	h.lastEvent.Store(at)
}

func (h *healthSnapshot) recordPublished(n int) {
	if n > 0 {
		h.published.Add(uint64(n))
	}
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, activeSeries int, stats CacheStats) health.Report {
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
		Message:       "indicator computation engine",
		Details: map[string]string{
			"enabled":             boolString(cfg.Enabled),
			"processed":           u64String(h.processed.Load()),
			"published":           u64String(h.published.Load()),
			"dropped":             u64String(dropped),
			"active_series":       u64String(uint64(activeSeries)),
			"ema_periods":         intSliceString(cfg.EMAPeriods()),
			"sma_periods":         intSliceString(cfg.SMAPeriods()),
			"rsi_periods":         intSliceString(cfg.RSIPeriods()),
			"atr_periods":         intSliceString(cfg.ATRPeriods()),
			"macd_instances":      u64String(uint64(stats.MACDInstances)),
			"bollinger_instances": u64String(uint64(stats.BollingerInstances)),
			"warmed_instances":    u64String(uint64(stats.WarmedInstances)),
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

func intSliceString(vals []int) string {
	if len(vals) == 0 {
		return ""
	}
	out := intString(vals[0])
	for i := 1; i < len(vals); i++ {
		out += "," + intString(vals[i])
	}
	return out
}

func intString(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
