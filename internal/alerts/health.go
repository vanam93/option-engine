package alerts

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	alertsGenerated      uint64
	duplicatesSuppressed uint64
	confidenceAlerts     uint64
	statusAlerts         uint64
	createdAlerts        uint64
	closedAlerts         uint64
	cooldownSuppressed   uint64
}

func (h *healthSnapshot) record(alertType AlertType, emitted, duplicate, cooldown bool) {
	if cooldown {
		h.cooldownSuppressed++
		return
	}
	if duplicate {
		h.duplicatesSuppressed++
		return
	}
	if !emitted {
		return
	}

	h.alertsGenerated++
	switch alertType {
	case AlertConfidenceIncreased, AlertConfidenceDecreased:
		h.confidenceAlerts++
	case AlertStatusChanged, AlertEntryZoneReached, AlertExitRecommended:
		h.statusAlerts++
	case AlertRecommendationCreated:
		h.createdAlerts++
	case AlertRecommendationClosed:
		h.closedAlerts++
	}
}

func (h *healthSnapshot) recordNoMeaningfulAlert() {
	h.duplicatesSuppressed++
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64) health.Report {
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
		Message:   "alert engine",
		Details: map[string]string{
			"enabled":               boolString(cfg.Enabled),
			"alerts_generated":      u64String(h.alertsGenerated),
			"duplicates_suppressed": u64String(h.duplicatesSuppressed),
			"confidence_alerts":     u64String(h.confidenceAlerts),
			"status_alerts":         u64String(h.statusAlerts),
			"created_alerts":        u64String(h.createdAlerts),
			"closed_alerts":         u64String(h.closedAlerts),
			"cooldown_suppressed":   u64String(h.cooldownSuppressed),
			"dropped":               u64String(dropped),
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
