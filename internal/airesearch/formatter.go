package airesearch

import "strings"

// FormatReport renders a research report as structured plain text.
func FormatReport(report ResearchReport) string {
	var b strings.Builder

	b.WriteString("AI Research Report\n")
	b.WriteString("==================\n\n")
	b.WriteString("Report ID: " + report.ReportID + "\n")
	b.WriteString("Study ID: " + report.StudyID + "\n")
	b.WriteString("Version: " + report.ResearchVersion + "\n")
	b.WriteString("Analyzer: " + report.Analyzer + "\n")
	b.WriteString("Generated: " + report.GeneratedAt.Format("2006-01-02T15:04:05Z") + "\n\n")

	writeSection(&b, "Executive Summary", report.Sections.ExecutiveSummary)
	writeSection(&b, "Strategy Overview", report.Sections.StrategyOverview)
	writeSection(&b, "Strengths", report.Sections.Strengths)
	writeSection(&b, "Weaknesses", report.Sections.Weaknesses)
	writeSection(&b, "Best Performing Symbols", report.Sections.BestPerformingSymbols)
	writeSection(&b, "Worst Performing Symbols", report.Sections.WorstPerformingSymbols)
	writeSection(&b, "Best Timeframes", report.Sections.BestTimeframes)
	writeSection(&b, "Worst Timeframes", report.Sections.WorstTimeframes)
	writeSection(&b, "Parameter Sensitivity", report.Sections.ParameterSensitivity)
	writeSection(&b, "Consistency Analysis", report.Sections.ConsistencyAnalysis)
	writeSection(&b, "Walk Forward Summary", report.Sections.WalkForwardSummary)
	writeSection(&b, "Monte Carlo Summary", report.Sections.MonteCarloSummary)
	writeSection(&b, "Risk Analysis", report.Sections.RiskAnalysis)
	writeSection(&b, "Confidence Assessment", report.Sections.ConfidenceAssessment)
	writeSection(&b, "Market Regime Suitability", report.Sections.MarketRegimeSuitability)
	writeSection(&b, "Suggested Improvements", report.Sections.SuggestedImprovements)
	writeSection(&b, "Suggested Future Experiments", report.Sections.SuggestedFutureExperiments)
	writeSection(&b, "Overall Verdict", report.Sections.OverallVerdict)

	return b.String()
}

func writeSection(b *strings.Builder, title, content string) {
	b.WriteString("--- " + title + " ---\n")
	b.WriteString(content + "\n\n")
}
