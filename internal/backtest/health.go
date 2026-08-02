package backtest

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

// ReplayStatus describes backtest replay lifecycle state.
type ReplayStatus string

const (
	ReplayStatusIdle      ReplayStatus = "idle"
	ReplayStatusRunning   ReplayStatus = "running"
	ReplayStatusCompleted ReplayStatus = "completed"
	ReplayStatusStopped   ReplayStatus = "stopped"
)

type replayMetrics struct {
	status         atomic.Value // ReplayStatus
	position       atomic.Uint64
	total          atomic.Uint64
	processed      atomic.Uint64
	symbolsLoaded  atomic.Uint64
	connected      atomic.Bool
	lastEvent      atomic.Value // time.Time
}

func newReplayMetrics() *replayMetrics {
	m := &replayMetrics{}
	m.status.Store(ReplayStatusIdle)
	return m
}

func (m *replayMetrics) setStatus(s ReplayStatus) {
	m.status.Store(s)
}

func (m *replayMetrics) statusValue() ReplayStatus {
	if v := m.status.Load(); v != nil {
		if s, ok := v.(ReplayStatus); ok {
			return s
		}
	}
	return ReplayStatusIdle
}

func (m *replayMetrics) recordReplay(at time.Time) {
	m.processed.Add(1)
	m.lastEvent.Store(at)
}

func (m *replayMetrics) report(cfg Config) health.Report {
	status := health.StatusHealthy
	replayStatus := m.statusValue()
	switch replayStatus {
	case ReplayStatusStopped:
		status = health.StatusDegraded
	case ReplayStatusIdle:
		if cfg.Enabled {
			status = health.StatusDegraded
		}
	}

	var last *time.Time
	if v := m.lastEvent.Load(); v != nil {
		if t, ok := v.(time.Time); ok && !t.IsZero() {
			last = &t
		}
	}

	total := m.total.Load()
	position := m.position.Load()
	progress := "0%"
	if total > 0 {
		pct := (position * 100) / total
		progress = u64String(pct) + "%"
	}

	return health.Report{
		Component:     engineName,
		Status:        status,
		Connected:     m.connected.Load(),
		LastEventTime: last,
		Message:       "backtest replay engine",
		Details: map[string]string{
			"enabled":          boolString(cfg.Enabled),
			"status":           string(replayStatus),
			"candles_replayed": u64String(m.processed.Load()),
			"symbols_loaded":   u64String(m.symbolsLoaded.Load()),
			"progress":         progress,
			"position":         u64String(position),
			"total_candles":    u64String(total),
			"speed":            f64String(cfg.Speed) + "x",
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
	if v == float64(int(v)) {
		return u64String(uint64(v))
	}
	neg := v < 0
	if neg {
		v = -v
	}
	intPart := uint64(v)
	frac := v - float64(intPart)
	fracPart := uint64(frac*10 + 0.5)
	buf := make([]byte, 0, 16)
	if neg {
		buf = append(buf, '-')
	}
	if intPart == 0 {
		buf = append(buf, '0')
	} else {
		n := intPart
		digits := make([]byte, 0, 12)
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
	buf = append(buf, byte('0'+fracPart%10))
	return string(buf)
}
