package risk

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	received   atomic.Uint64
	approved   atomic.Uint64
	rejected   atomic.Uint64
	lastEvent  atomic.Value // time.Time
	rejections rejectionCounters
}

type rejectionCounters struct {
	confidence   atomic.Uint64
	maxPositions atomic.Uint64
	duplicate    atomic.Uint64
	dailyLimit   atomic.Uint64
	other        atomic.Uint64
}

func (h *healthSnapshot) record(intent ApprovedTradeIntent) {
	h.received.Add(1)
	if intent.Status == Approved {
		h.approved.Add(1)
	} else {
		h.rejected.Add(1)
		h.rejections.record(intent.Reason)
	}
	h.lastEvent.Store(intent.Timestamp)
}

func (r *rejectionCounters) record(reason string) {
	switch reason {
	case "confidence below threshold":
		r.confidence.Add(1)
	case "max positions exceeded":
		r.maxPositions.Add(1)
	case "duplicate long position", "duplicate short position":
		r.duplicate.Add(1)
	case "daily trade limit exceeded":
		r.dailyLimit.Add(1)
	default:
		r.other.Add(1)
	}
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
		Message:       "decision and risk engine",
		Details: map[string]string{
			"enabled":             boolString(cfg.Enabled),
			"decisions_received":  u64String(h.received.Load()),
			"approved_trades":     u64String(h.approved.Load()),
			"rejected_trades":     u64String(h.rejected.Load()),
			"active_positions":    u64String(uint64(activePositions)),
			"rejection_reasons":   h.rejections.summary(),
			"dropped":             u64String(dropped),
		},
	}
}

func (r *rejectionCounters) summary() string {
	return "confidence=" + u64String(r.confidence.Load()) +
		",max_positions=" + u64String(r.maxPositions.Load()) +
		",duplicate=" + u64String(r.duplicate.Load()) +
		",daily_limit=" + u64String(r.dailyLimit.Load()) +
		",other=" + u64String(r.other.Load())
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
