package airesearch

import (
	"fmt"
	"strings"

	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
)

// BuildPrompt constructs a deterministic analysis prompt from study data.
func BuildPrompt(study laboratory.Study) string {
	var b strings.Builder

	b.WriteString("AI Research Analysis Request\n")
	b.WriteString("============================\n\n")

	b.WriteString(fmt.Sprintf("Study ID: %s\n", study.StudyID))
	b.WriteString(fmt.Sprintf("Name: %s\n", study.Name))
	if study.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", study.Description))
	}
	b.WriteString(fmt.Sprintf("Strategy: %s\n", study.Strategy))
	b.WriteString(fmt.Sprintf("Research Version: %s\n", study.ResearchVersion))
	b.WriteString(fmt.Sprintf("Status: %s\n", study.Status))
	b.WriteString(fmt.Sprintf("Date Range: %s to %s\n", study.StartTime.Format("2006-01-02"), study.EndTime.Format("2006-01-02")))
	b.WriteString(fmt.Sprintf("Symbols: %s\n", strings.Join(study.Symbols, ", ")))

	timeframes := make([]string, 0, len(study.Timeframes))
	for _, tf := range study.Timeframes {
		timeframes = append(timeframes, string(tf))
	}
	b.WriteString(fmt.Sprintf("Timeframes: %s\n", strings.Join(timeframes, ", ")))

	if len(study.Parameters) > 0 {
		b.WriteString("\nParameters:\n")
		for k, v := range study.Parameters {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	if study.Output == nil {
		b.WriteString("\nNo study output available.\n")
		return b.String()
	}

	out := study.Output
	b.WriteString("\nBacktest Sessions: ")
	b.WriteString(fmt.Sprintf("%d\n", len(out.BacktestSummaries)))

	for i, summary := range out.BacktestSummaries {
		b.WriteString(fmt.Sprintf("\n--- Session %d (%s) ---\n", i+1, summary.BacktestID))
		b.WriteString(fmt.Sprintf("Recommendations: %d generated, %d closed\n", summary.RecommendationsGenerated, summary.RecommendationsClosed))
		b.WriteString(fmt.Sprintf("Levels: BUY=%d WATCH=%d AVOID=%d\n", summary.BuyCount, summary.WatchCount, summary.AvoidCount))
		b.WriteString(fmt.Sprintf("Confidence: avg=%.4f high=%.4f low=%.4f\n", summary.AverageConfidence, summary.HighestConfidence, summary.LowestConfidence))
		b.WriteString(fmt.Sprintf("Returns: avg=%.4f win_rate=%.4f loss_rate=%.4f\n", summary.AverageReturn, summary.WinRate, summary.LossRate))
		b.WriteString(fmt.Sprintf("Optimization runs: %d\n", summary.OptimizationRuns))
		b.WriteString(fmt.Sprintf("Walk-forward runs: %d\n", summary.WalkForwardRuns))
		b.WriteString(fmt.Sprintf("Monte Carlo runs: %d\n", summary.MonteCarloRuns))
		b.WriteString(fmt.Sprintf("Alerts: %d\n", summary.AlertsGenerated))
		b.WriteString(fmt.Sprintf("Research reports: %d\n", summary.ResearchReportsGenerated))
		if len(summary.SymbolDistribution) > 0 {
			b.WriteString("Symbol distribution: ")
			b.WriteString(formatDistribution(summary.SymbolDistribution))
			b.WriteString("\n")
		}
		if len(summary.TimeframeDistribution) > 0 {
			b.WriteString("Timeframe distribution: ")
			b.WriteString(formatDistribution(summary.TimeframeDistribution))
			b.WriteString("\n")
		}
		if len(summary.QualityDistribution) > 0 {
			b.WriteString("Quality distribution: ")
			b.WriteString(formatDistribution(summary.QualityDistribution))
			b.WriteString("\n")
		}
	}

	b.WriteString("\nAnalyze this study and produce a structured research report covering strengths, weaknesses, risk, consistency, and actionable recommendations.\n")
	return b.String()
}

func formatDistribution(dist map[string]int) string {
	parts := make([]string, 0, len(dist))
	for k, v := range dist {
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	return strings.Join(parts, ", ")
}
