package console

import (
	"strconv"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	documentsRendered    uint64
	updatesRendered      uint64
	renderErrors         uint64
	totalRenderLatencyNs uint64
	renderSamples        uint64
}

func (h *healthSnapshot) recordRender(duration time.Duration, isUpdate bool, err error) {
	if err != nil {
		h.renderErrors++
		return
	}
	if isUpdate {
		h.updatesRendered++
	} else {
		h.documentsRendered++
	}
	h.totalRenderLatencyNs += uint64(duration.Nanoseconds())
	h.renderSamples++
}

func (h *healthSnapshot) averageRenderLatencyMs() float64 {
	if h.renderSamples == 0 {
		return 0
	}
	return float64(h.totalRenderLatencyNs) / float64(h.renderSamples) / 1e6
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, tracked int) health.Report {
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
		Message:   "recommendation console",
		Details: map[string]string{
			"enabled":                   boolString(cfg.Enabled),
			"tracked_recommendations":   strconv.Itoa(tracked),
			"documents_rendered":        u64String(h.documentsRendered),
			"updates_rendered":          u64String(h.updatesRendered),
			"render_errors":             u64String(h.renderErrors),
			"average_render_latency_ms": floatString(h.averageRenderLatencyMs()),
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
