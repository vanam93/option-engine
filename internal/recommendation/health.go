package recommendation

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	generated         uint64
	strongBuy         uint64
	buy               uint64
	watch             uint64
	avoid             uint64
	totalConfidence   float64
	recommendationCnt uint64
}

func (h *healthSnapshot) record(rec RecommendationUpdated) {
	h.generated++
	h.recommendationCnt++
	h.totalConfidence += rec.Confidence
	switch rec.Recommendation {
	case LevelStrongBuy:
		h.strongBuy++
	case LevelBuy:
		h.buy++
	case LevelWatch:
		h.watch++
	case LevelAvoid:
		h.avoid++
	}
}

func (h *healthSnapshot) averageConfidence() float64 {
	if h.recommendationCnt == 0 {
		return 0
	}
	return h.totalConfidence / float64(h.recommendationCnt)
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
		Message:   "recommendation engine",
		Details: map[string]string{
			"enabled":                   boolString(cfg.Enabled),
			"recommendations_generated": u64String(h.generated),
			"strong_buy":                u64String(h.strongBuy),
			"buy":                       u64String(h.buy),
			"watch":                     u64String(h.watch),
			"avoid":                     u64String(h.avoid),
			"average_confidence":        floatString(h.averageConfidence()),
			"dropped":                   u64String(dropped),
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
