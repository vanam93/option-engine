package airesearch

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	reportsGenerated     uint64
	reportsCached        uint64
	publishFailures      uint64
	totalAnalysisLatencyNs uint64
	analysisSamples      uint64
}

func (h *healthSnapshot) recordGenerated(duration time.Duration) {
	h.reportsGenerated++
	h.totalAnalysisLatencyNs += uint64(duration.Nanoseconds())
	h.analysisSamples++
}

func (h *healthSnapshot) recordCached() {
	h.reportsCached++
}

func (h *healthSnapshot) recordPublishFailure() {
	h.publishFailures++
}

func (h *healthSnapshot) averageAnalysisLatencyMs() float64 {
	if h.analysisSamples == 0 {
		return 0
	}
	return float64(h.totalAnalysisLatencyNs) / float64(h.analysisSamples) / 1e6
}

func (h *healthSnapshot) report(cfg Config, connected bool, repositoryEntries int, dropped uint64) health.Report {
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
		Message:   "ai research engine",
		Details: map[string]string{
			"enabled":                 boolString(cfg.Enabled),
			"analyzer":                cfg.Analyzer,
			"reports_generated":       u64String(h.reportsGenerated),
			"reports_cached":          u64String(h.reportsCached),
			"analysis_latency":          floatString(h.averageAnalysisLatencyMs()),
			"publish_failures":        u64String(h.publishFailures),
			"repository_entries":      strconv.Itoa(repositoryEntries),
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
