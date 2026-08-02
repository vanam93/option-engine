package laboratory

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	studiesCreated             uint64
	studiesCompleted           uint64
	studiesFailed              uint64
	comparisons                uint64
	totalExecutionDurationNs   uint64
	executionSamples           uint64
}

func (h *healthSnapshot) recordCreated() {
	h.studiesCreated++
}

func (h *healthSnapshot) recordCompleted(duration time.Duration) {
	h.studiesCompleted++
	h.totalExecutionDurationNs += uint64(duration.Nanoseconds())
	h.executionSamples++
}

func (h *healthSnapshot) recordFailed(duration time.Duration) {
	h.studiesFailed++
	h.totalExecutionDurationNs += uint64(duration.Nanoseconds())
	h.executionSamples++
}

func (h *healthSnapshot) recordComparison() {
	h.comparisons++
}

func (h *healthSnapshot) averageExecutionDurationMs() float64 {
	if h.executionSamples == 0 {
		return 0
	}
	return float64(h.totalExecutionDurationNs) / float64(h.executionSamples) / 1e6
}

func (h *healthSnapshot) report(cfg Config, connected bool, activeStudies, repositoryEntries, comparisonEntries int) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}

	return health.Report{
		Component: engineName,
		Status:    status,
		Connected: connected,
		Message:   "strategy laboratory",
		Details: map[string]string{
			"enabled":                        boolString(cfg.Enabled),
			"auto_version":                   boolString(cfg.AutoVersion),
			"concurrent_studies":             strconv.Itoa(cfg.ConcurrentStudies),
			"active_studies":                 strconv.Itoa(activeStudies),
			"studies_created":                u64String(h.studiesCreated),
			"studies_completed":              u64String(h.studiesCompleted),
			"studies_failed":                 u64String(h.studiesFailed),
			"comparisons":                    u64String(h.comparisons),
			"repository_entries":             strconv.Itoa(repositoryEntries),
			"average_execution_duration_ms":  floatString(h.averageExecutionDurationMs()),
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
