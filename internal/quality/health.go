package quality

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	recommendationsTracked   uint64
	recommendationsCompleted uint64
}

func (h *healthSnapshot) recordTracked() {
	h.recommendationsTracked++
}

func (h *healthSnapshot) recordCompleted() {
	h.recommendationsCompleted++
}

func (h *healthSnapshot) report(
	cfg Config,
	connected bool,
	dropped uint64,
	activeTrackers int,
	completedReports int,
	stats aggregateHistoricalStats,
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
		Message:   "recommendation quality engine",
		Details: map[string]string{
			"enabled":                   boolString(cfg.Enabled),
			"recommendations_tracked":   u64String(h.recommendationsTracked),
			"recommendations_completed": u64String(h.recommendationsCompleted),
			"successful":                strconv.Itoa(stats.Successful),
			"failed":                    strconv.Itoa(stats.Failed),
			"expired":                   strconv.Itoa(stats.Expired),
			"average_return":            floatString(stats.averageReturn()),
			"average_quality_score":     floatString(stats.averageQualityScore()),
			"average_tracking_minutes":  floatString(stats.averageTrackingMinutes()),
			"active_trackers":           strconv.Itoa(activeTrackers),
			"completed_reports":         strconv.Itoa(completedReports),
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
