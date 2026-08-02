package recommendationstate

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	updatesProcessed uint64
	duplicatesMerged uint64
}

func (h *healthSnapshot) record(duplicateMerged bool) {
	h.updatesProcessed++
	if duplicateMerged {
		h.duplicatesMerged++
	}
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, cache *Cache) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}
	if dropped > 0 {
		status = health.StatusDegraded
	}

	active, closed, timelineEntries, averageConfidence := cache.Stats()

	return health.Report{
		Component: engineName,
		Status:    status,
		Connected: connected,
		Message:   "recommendation state manager",
		Details: map[string]string{
			"enabled":                boolString(cfg.Enabled),
			"active_recommendations": intString(active),
			"closed_recommendations": intString(closed),
			"timeline_entries":       intString(timelineEntries),
			"updates_processed":      u64String(h.updatesProcessed),
			"duplicates_merged":      u64String(h.duplicatesMerged),
			"average_confidence":     floatString(averageConfidence),
			"dropped":                u64String(dropped),
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

func intString(v int) string {
	return strconv.Itoa(v)
}

func floatString(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
