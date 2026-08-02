package opportunity

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	summary Summary
}

func (h *healthSnapshot) update(summary Summary) {
	h.summary = summary
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, summary Summary) health.Report {
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
		Message:   "opportunity ranking engine",
		Details: map[string]string{
			"enabled":              boolString(cfg.Enabled),
			"opportunities_ranked": u64String(uint64(summary.OpportunitiesRanked)),
			"top_candidates":       u64String(uint64(summary.TopCandidates)),
			"average_confidence":   floatString(summary.AverageConfidence),
			"buy_count":            u64String(uint64(summary.BuyCount)),
			"watch_count":          u64String(uint64(summary.WatchCount)),
			"ignore_count":         u64String(uint64(summary.IgnoreCount)),
			"dropped":              u64String(dropped),
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
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func floatString(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
