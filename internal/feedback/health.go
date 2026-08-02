package feedback

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	eventsProcessed           uint64
	feedbackGenerated         uint64
	malformedEvents           uint64
	publishFailures           uint64
	totalProcessingLatencyNs  uint64
	processingSamples         uint64
}

func (h *healthSnapshot) recordProcessed(duration time.Duration) {
	h.eventsProcessed++
	h.totalProcessingLatencyNs += uint64(duration.Nanoseconds())
	h.processingSamples++
}

func (h *healthSnapshot) recordFeedbackGenerated() {
	h.feedbackGenerated++
}

func (h *healthSnapshot) recordMalformed() {
	h.malformedEvents++
}

func (h *healthSnapshot) recordPublishFailure() {
	h.publishFailures++
}

func (h *healthSnapshot) averageProcessingLatencyMs() float64 {
	if h.processingSamples == 0 {
		return 0
	}
	return float64(h.totalProcessingLatencyNs) / float64(h.processingSamples) / 1e6
}

func (h *healthSnapshot) report(
	cfg Config,
	connected bool,
	dropped uint64,
	strategies, symbols, timeframes, recommendations int,
	confidenceBuckets, rollingWindows, cacheEntries int,
) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}
	if dropped > 0 || h.publishFailures > 0 {
		status = health.StatusDegraded
	}

	return health.Report{
		Component: engineName,
		Status:    status,
		Connected: connected,
		Message:   "recommendation feedback engine",
		Details: map[string]string{
			"enabled":                         boolString(cfg.Enabled),
			"events_processed":                u64String(h.eventsProcessed),
			"feedback_generated":              u64String(h.feedbackGenerated),
			"tracked_strategies":              strconv.Itoa(strategies),
			"tracked_symbols":                 strconv.Itoa(symbols),
			"tracked_timeframes":              strconv.Itoa(timeframes),
			"tracked_recommendations":         strconv.Itoa(recommendations),
			"confidence_buckets":              strconv.Itoa(confidenceBuckets),
			"rolling_windows":                 strconv.Itoa(rollingWindows),
			"cache_entries":                   strconv.Itoa(cacheEntries),
			"average_processing_latency_ms":   floatString(h.averageProcessingLatencyMs()),
			"malformed_events":                u64String(h.malformedEvents),
			"publish_failures":                u64String(h.publishFailures),
			"dropped_events":                  u64String(dropped),
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
