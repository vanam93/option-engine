package validation

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	validated            uint64
	rejected             uint64
	duplicateSuppressed  uint64
	expired              uint64
	totalValidationScore float64
	validationCount      uint64
}

func (h *healthSnapshot) record(result ValidatedRecommendation, score float64, duplicateSuppressed, expired bool) {
	if duplicateSuppressed {
		h.duplicateSuppressed++
		return
	}
	if expired {
		h.expired++
	}
	h.validationCount++
	h.totalValidationScore += score
	switch result.ValidationStatus {
	case StatusValid:
		h.validated++
	case StatusRejected:
		h.rejected++
	}
}

func (h *healthSnapshot) averageValidationScore() float64 {
	if h.validationCount == 0 {
		return 0
	}
	return h.totalValidationScore / float64(h.validationCount)
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
		Message:   "recommendation validation engine",
		Details: map[string]string{
			"enabled":                  boolString(cfg.Enabled),
			"validated":                u64String(h.validated),
			"rejected":                 u64String(h.rejected),
			"duplicate_suppressed":     u64String(h.duplicateSuppressed),
			"expired":                  u64String(h.expired),
			"average_validation_score": floatString(h.averageValidationScore()),
			"dropped":                  u64String(dropped),
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
