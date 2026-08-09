package researchengine

import (
	"fmt"
	"strings"
)

// QualificationStatus classifies strategy research outcomes.
type QualificationStatus string

const (
	QualExcellent QualificationStatus = "excellent"
	QualGood      QualificationStatus = "good"
	QualAverage   QualificationStatus = "average"
	QualPoor      QualificationStatus = "poor"
	QualRejected  QualificationStatus = "rejected"
)

// Qualification is the automatic pass/fail classification for a strategy run.
type Qualification struct {
	Status         QualificationStatus `json:"status"`
	Reasons        []string            `json:"reasons"`
	Recommendation string              `json:"recommendation"`
}

// Qualify classifies strategy performance using fixed research thresholds.
func Qualify(stats Statistics) Qualification {
	pf := stats.ProfitFactor
	dd := stats.MaxDrawdownPercent
	wr := stats.WinRate

	if pf < 0.8 || dd > 30 {
		reasons := []string{}
		if pf < 0.8 {
			reasons = append(reasons, fmt.Sprintf("Profit Factor = %.2f (must be >= 0.8)", pf))
		}
		if dd > 30 {
			reasons = append(reasons, fmt.Sprintf("Drawdown = %.2f%% (must be <= 30%%)", dd))
		}
		return Qualification{
			Status:         QualRejected,
			Reasons:        reasons,
			Recommendation: "Do not optimize. Do not use for live trading. Remove from candidate pool.",
		}
	}

	if pf > 2.0 && wr > 0.60 && dd < 10 {
		return Qualification{
			Status:         QualExcellent,
			Reasons:        []string{fmt.Sprintf("PF %.2f, win rate %.1f%%, drawdown %.2f%%", pf, wr*100, dd)},
			Recommendation: "Strong candidate for optimization and walk-forward validation.",
		}
	}

	if pf > 1.5 && dd < 15 {
		return Qualification{
			Status:         QualGood,
			Reasons:        []string{fmt.Sprintf("PF %.2f, drawdown %.2f%%", pf, dd)},
			Recommendation: "Candidate for parameter optimization.",
		}
	}

	if pf > 1.2 {
		return Qualification{
			Status:         QualAverage,
			Reasons:        []string{fmt.Sprintf("PF %.2f above minimum profitability threshold", pf)},
			Recommendation: "Marginal edge; optimize cautiously or combine with regime filters.",
		}
	}

	return Qualification{
		Status:         QualPoor,
		Reasons:        []string{fmt.Sprintf("PF %.2f in weak range (0.8–1.2)", pf)},
		Recommendation: "Low edge; deprioritize unless regime analysis shows niche fit.",
	}
}

// FormatQualification renders qualification for CLI output.
func FormatQualification(q Qualification) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Status: %s\n", strings.ToUpper(string(q.Status))))
	if len(q.Reasons) > 0 {
		b.WriteString("Reasons:\n")
		for _, r := range q.Reasons {
			b.WriteString("  - " + r + "\n")
		}
	}
	if q.Recommendation != "" {
		b.WriteString("Recommendation: " + q.Recommendation + "\n")
	}
	return b.String()
}
