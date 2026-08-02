package aicontext

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildExecutivePrompt renders a short summary prompt for quick AI analysis.
func BuildExecutivePrompt(ctx AIContext) string {
	var b strings.Builder
	b.WriteString(tmplExecutiveHeader)
	b.WriteString(fmt.Sprintf(
		tmplExecutiveBody,
		ctx.Metadata.Name,
		ctx.StudyID,
		ctx.Metadata.Strategy,
		ctx.ResearchVersion,
		ctx.Metadata.StartDate,
		ctx.Metadata.EndDate,
		strings.Join(ctx.Metadata.Symbols, ", "),
		strings.Join(ctx.Metadata.Timeframes, ", "),
		ctx.ResearchReport.OverallVerdict,
		formatFloat(ctx.Performance.AverageReturn),
		formatFloat(ctx.Performance.WinRate),
		formatFloat(ctx.Performance.AverageConfidence),
		ctx.Trades.WinningTrades,
		ctx.Trades.LosingTrades,
		formatFloat(ctx.Trades.Expectancy),
		formatFloat(ctx.Trades.ProfitFactor),
		formatBulletList(ctx.KeyFindings),
	))
	return b.String()
}

// BuildTechnicalPrompt renders a complete research report with all metrics.
func BuildTechnicalPrompt(ctx AIContext) string {
	var b strings.Builder

	b.WriteString(tmplTechnicalHeader)
	b.WriteString(fmt.Sprintf(
		tmplTechnicalMetadata,
		ctx.Metadata.StudyID,
		ctx.Metadata.Name,
		ctx.Metadata.Strategy,
		ctx.ResearchVersion,
		ctx.Metadata.Status,
		ctx.Metadata.StartDate,
		ctx.Metadata.EndDate,
		strings.Join(ctx.Metadata.Symbols, ", "),
		strings.Join(ctx.Metadata.Timeframes, ", "),
		formatParameters(ctx.Metadata.Parameters),
		strings.Join(ctx.Metadata.BacktestIDs, ", "),
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalPerformance,
		ctx.Performance.RecommendationsGenerated,
		ctx.Performance.RecommendationsClosed,
		ctx.Performance.BuyCount,
		ctx.Performance.WatchCount,
		ctx.Performance.AvoidCount,
		formatFloat(ctx.Performance.AverageConfidence),
		formatFloat(ctx.Risk.ConfidenceLow),
		formatFloat(ctx.Risk.ConfidenceHigh),
		formatFloat(ctx.Performance.AverageReturn),
		formatFloat(ctx.Performance.WinRate),
		formatFloat(ctx.Performance.LossRate),
		ctx.Performance.SessionCount,
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalOptimization,
		ctx.Optimization.TotalRuns,
		formatSessionLines(ctx.Optimization.PerSession, func(s OptimizationSession) string { return s.BacktestID }, func(s OptimizationSession) int { return s.Runs }),
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalWalkForward,
		ctx.WalkForward.TotalRuns,
		formatSessionLines(ctx.WalkForward.PerSession, func(s WalkForwardSession) string { return s.BacktestID }, func(s WalkForwardSession) int { return s.Runs }),
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalMonteCarlo,
		ctx.MonteCarlo.TotalRuns,
		formatSessionLines(ctx.MonteCarlo.PerSession, func(s MonteCarloSession) string { return s.BacktestID }, func(s MonteCarloSession) int { return s.Runs }),
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalRisk,
		ctx.Risk.AlertsGenerated,
		formatFloat(ctx.Risk.LossRate),
		ctx.Risk.AverageHoldingTimeMs,
		formatFloat(ctx.Risk.Drawdown),
		formatFloat(ctx.Risk.ConfidenceLow),
		formatFloat(ctx.Risk.ConfidenceHigh),
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalTrades,
		ctx.Trades.TotalTrades,
		ctx.Trades.WinningTrades,
		ctx.Trades.LosingTrades,
		formatFloat(ctx.Trades.Drawdown),
		formatFloat(ctx.Trades.Expectancy),
		formatFloat(ctx.Trades.ProfitFactor),
	))

	b.WriteString(fmt.Sprintf(
		tmplTechnicalReport,
		ctx.ResearchReport.Analyzer,
		ctx.ResearchReport.ExecutiveSummary,
		ctx.ResearchReport.OverallVerdict,
		ctx.ResearchReport.Consistency,
		ctx.ResearchReport.RiskAnalysis,
		ctx.ResearchReport.Confidence,
		ctx.ResearchReport.RegimeSuitability,
	))

	b.WriteString(fmt.Sprintf(tmplTechnicalFindings, formatBulletList(ctx.KeyFindings)))
	b.WriteString(fmt.Sprintf(tmplTechnicalStrengths, formatBulletList(ctx.Strengths)))
	b.WriteString(fmt.Sprintf(tmplTechnicalWeaknesses, formatBulletList(ctx.Weaknesses)))
	b.WriteString(fmt.Sprintf(tmplTechnicalExperiments, formatBulletList(ctx.FutureExperiments)))
	b.WriteString(fmt.Sprintf(tmplTechnicalTimeline, formatTimeline(ctx.Timeline)))

	return b.String()
}

// BuildJSONPrompt renders machine-readable context with deterministic ordering.
func BuildJSONPrompt(ctx AIContext) string {
	data, err := json.Marshal(toJSONContext(ctx))
	if err != nil {
		return ""
	}
	return fmt.Sprintf(tmplJSONEnvelope, string(data))
}
