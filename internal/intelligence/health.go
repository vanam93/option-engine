package intelligence

import (
	"strconv"

	"github.com/vanam-gangireddy/option-engine/internal/core/health"
)

type healthSnapshot struct {
	documentsGenerated    uint64
	timelineSummaries     uint64
	researchSummaries     uint64
	upgradeExplanations   uint64
	downgradeExplanations uint64
	totalConfidence       float64
}

func (h *healthSnapshot) record(doc IntelligenceDocument, upgraded, downgraded bool) {
	h.documentsGenerated++
	h.totalConfidence += doc.Confidence
	if doc.TimelineSummary != "" {
		h.timelineSummaries++
	}
	if doc.ResearchSummary != "" {
		h.researchSummaries++
	}
	if upgraded {
		h.upgradeExplanations++
	}
	if downgraded {
		h.downgradeExplanations++
	}
}

func (h *healthSnapshot) report(cfg Config, connected bool, dropped uint64, documents int, averageConfidence float64) health.Report {
	status := health.StatusHealthy
	if cfg.Enabled && !connected {
		status = health.StatusDegraded
	}
	if dropped > 0 {
		status = health.StatusDegraded
	}

	avg := averageConfidence
	if h.documentsGenerated > 0 && avg == 0 {
		avg = h.totalConfidence / float64(h.documentsGenerated)
	}

	return health.Report{
		Component: engineName,
		Status:    status,
		Connected: connected,
		Message:   "recommendation intelligence engine",
		Details: map[string]string{
			"enabled":                boolString(cfg.Enabled),
			"documents_generated":    u64String(h.documentsGenerated),
			"average_confidence":     floatString(avg),
			"timeline_summaries":     u64String(h.timelineSummaries),
			"research_summaries":     u64String(h.researchSummaries),
			"upgrade_explanations":   u64String(h.upgradeExplanations),
			"downgrade_explanations": u64String(h.downgradeExplanations),
			"cached_documents":       strconv.Itoa(documents),
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

func floatString(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
