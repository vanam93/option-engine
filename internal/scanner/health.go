package scanner

import (
	"sync/atomic"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	eventsProcessed atomic.Uint64
	matchesFound    atomic.Uint64
	lastEvent       atomic.Value // time.Time
}

func (h *healthSnapshot) recordEvent() {
	h.eventsProcessed.Add(1)
}

func (h *healthSnapshot) recordMatch(at time.Time) {
	h.matchesFound.Add(1)
	if !at.IsZero() {
		h.lastEvent.Store(at)
	}
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, symbolsScanned int) health.Report {
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
		Message:       "market scanner engine",
		Details: map[string]string{
			"enabled":           boolString(cfg.Enabled),
			"symbols_scanned":   u64String(uint64(symbolsScanned)),
			"events_processed":  u64String(h.eventsProcessed.Load()),
			"matches_found":     u64String(h.matchesFound.Load()),
			"scanner_count":     u64String(uint64(cfg.EnabledScannerCount())),
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
