package performance

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	processed  atomic.Uint64
	snapshots  atomic.Uint64
	lastEvent  atomic.Value // time.Time
}

func (h *healthSnapshot) record(at time.Time) {
	h.processed.Add(1)
	h.snapshots.Add(1)
	h.lastEvent.Store(at)
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

	cache.mu.Lock()
	tradesProcessed := cache.tradesProcessed()
	snapshotsGenerated := cache.snapshotsGenerated()
	totalPnL := cache.totalPnL()
	currentDrawdown := cache.currentDrawdown()
	cache.mu.Unlock()

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     connected,
		LastEventTime: last,
		Message:       "performance analytics engine",
		Details: map[string]string{
			"enabled":              boolString(cfg.Enabled),
			"trades_processed":     u64String(uint64(tradesProcessed)),
			"snapshots_generated":  u64String(uint64(snapshotsGenerated)),
			"total_pnl":            f64String(totalPnL),
			"current_drawdown":     f64String(currentDrawdown),
			"events_processed":     u64String(h.processed.Load()),
			"dropped":              u64String(dropped),
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

func f64String(v float64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	intPart := uint64(v)
	frac := v - float64(intPart)
	fracPart := uint64(frac*10000 + 0.5)
	if fracPart >= 10000 {
		intPart++
		fracPart = 0
	}
	buf := make([]byte, 0, 32)
	if neg {
		buf = append(buf, '-')
	}
	if intPart == 0 {
		buf = append(buf, '0')
	} else {
		n := intPart
		digits := make([]byte, 0, 20)
		for n > 0 {
			digits = append(digits, byte('0'+n%10))
			n /= 10
		}
		for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
			digits[i], digits[j] = digits[j], digits[i]
		}
		buf = append(buf, digits...)
	}
	buf = append(buf, '.')
	for i := 3; i >= 0; i-- {
		d := fracPart
		for j := 0; j < i; j++ {
			d /= 10
		}
		buf = append(buf, byte('0'+d%10))
	}
	return string(buf)
}
