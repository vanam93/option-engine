package aicontext

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	contextsGenerated        uint64
	contextsCached           uint64
	publishFailures          uint64
	totalGenerationLatencyNs uint64
	generationSamples        uint64
}

func (h *healthSnapshot) recordGenerated(duration time.Duration) {
	h.contextsGenerated++
	h.totalGenerationLatencyNs += uint64(duration.Nanoseconds())
	h.generationSamples++
}

func (h *healthSnapshot) recordCached() {
	h.contextsCached++
}

func (h *healthSnapshot) recordPublishFailure() {
	h.publishFailures++
}

func (h *healthSnapshot) averageGenerationLatencyMs() float64 {
	if h.generationSamples == 0 {
		return 0
	}
	return float64(h.totalGenerationLatencyNs) / float64(h.generationSamples) / 1e6
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
		Message:   "ai context engine",
		Details: map[string]string{
			"enabled":                      boolString(cfg.Enabled),
			"executive_prompt":             boolString(cfg.ExecutivePrompt),
			"technical_prompt":             boolString(cfg.TechnicalPrompt),
			"json_prompt":                  boolString(cfg.JSONPrompt),
			"contexts_generated":           u64String(h.contextsGenerated),
			"contexts_cached":              u64String(h.contextsCached),
			"average_generation_latency":   floatString(h.averageGenerationLatencyMs()),
			"publish_failures":             u64String(h.publishFailures),
			"repository_entries":           strconv.Itoa(repositoryEntries),
			"dropped_events":               u64String(dropped),
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
