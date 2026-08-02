package delivery

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	eventsProcessed      uint64
	updates              uint64
	totalUpdateLatencyNs uint64
	updateSamples        uint64
}

func (h *healthSnapshot) recordEvent(duration time.Duration, published bool) {
	h.eventsProcessed++
	if published {
		h.updates++
	}
	h.totalUpdateLatencyNs += uint64(duration.Nanoseconds())
	h.updateSamples++
}

func (h *healthSnapshot) averageUpdateLatencyMs() float64 {
	if h.updateSamples == 0 {
		return 0
	}
	return float64(h.totalUpdateLatencyNs) / float64(h.updateSamples) / 1e6
}

func (h *healthSnapshot) report(
	cfg Config,
	connected bool,
	dropped uint64,
	documents, active, closed, timelineEntries int,
	cacheHits, cacheMisses uint64,
) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}
	if dropped > 0 {
		status = health.StatusDegraded
	}

	return health.Report{
		Component: engineName,
		Status:    status,
		Connected: connected,
		Message:   "recommendation delivery engine",
		Details: map[string]string{
			"enabled":                   boolString(cfg.Enabled),
			"documents":                 strconv.Itoa(documents),
			"active_documents":          strconv.Itoa(active),
			"closed_documents":          strconv.Itoa(closed),
			"events_processed":          u64String(h.eventsProcessed),
			"updates":                   u64String(h.updates),
			"cache_hits":                u64String(cacheHits),
			"cache_misses":              u64String(cacheMisses),
			"timeline_entries":          strconv.Itoa(timelineEntries),
			"average_update_latency_ms": floatString(h.averageUpdateLatencyMs()),
			"dropped_events":            u64String(dropped),
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
	return strconv.FormatUint(v, 10)
}

func floatString(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
