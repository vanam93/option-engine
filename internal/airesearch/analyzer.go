package airesearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
)

// ResearchAnalyzer produces structured research reports from study data.
type ResearchAnalyzer interface {
	Analyze(ctx context.Context, study laboratory.Study, prompt string) (ReportSections, error)
}

// RuleBasedAnalyzer generates deterministic reports without invoking an LLM.
type RuleBasedAnalyzer struct{}

// NewRuleBasedAnalyzer creates a rule-based research analyzer.
func NewRuleBasedAnalyzer() *RuleBasedAnalyzer {
	return &RuleBasedAnalyzer{}
}

// Analyze produces a deterministic structured report from study metrics.
func (a *RuleBasedAnalyzer) Analyze(_ context.Context, study laboratory.Study, _ string) (ReportSections, error) {
	metrics := aggregateMetrics(study)
	return ReportSections{
		ExecutiveSummary:           buildExecutiveSummary(study, metrics),
		StrategyOverview:           buildStrategyOverview(study),
		Strengths:                  buildStrengths(metrics),
		Weaknesses:                 buildWeaknesses(study, metrics),
		BestPerformingSymbols:      buildRankedSymbols(metrics.symbolCounts, true),
		WorstPerformingSymbols:     buildRankedSymbols(metrics.symbolCounts, false),
		BestTimeframes:             buildRankedTimeframes(metrics.timeframeCounts, true),
		WorstTimeframes:            buildRankedTimeframes(metrics.timeframeCounts, false),
		ParameterSensitivity:       buildParameterSensitivity(study, metrics),
		ConsistencyAnalysis:        buildConsistencyAnalysis(metrics),
		WalkForwardSummary:         buildWalkForwardSummary(study),
		MonteCarloSummary:          buildMonteCarloSummary(study),
		RiskAnalysis:               buildRiskAnalysis(metrics),
		ConfidenceAssessment:       buildConfidenceAssessment(metrics),
		MarketRegimeSuitability:    buildMarketRegimeSuitability(metrics),
		SuggestedImprovements:      buildSuggestedImprovements(study, metrics),
		SuggestedFutureExperiments: buildSuggestedFutureExperiments(study, metrics),
		OverallVerdict:             buildOverallVerdict(study, metrics),
	}, nil
}

type studyMetrics struct {
	sessionCount        int
	totalRecommendations int
	totalClosed         int
	buyCount            int
	watchCount          int
	avoidCount          int
	avgConfidence       float64
	highConfidence      float64
	lowConfidence       float64
	avgReturn           float64
	winRate             float64
	lossRate            float64
	optimizationRuns    int
	walkForwardRuns     int
	monteCarloRuns      int
	alertsGenerated     int
	researchReports     int
	symbolCounts        map[string]int
	timeframeCounts     map[string]int
	qualityCounts       map[string]int
	winRates            []float64
	returns             []float64
	confidences         []float64
}

func aggregateMetrics(study laboratory.Study) studyMetrics {
	m := studyMetrics{
		symbolCounts:    make(map[string]int),
		timeframeCounts: make(map[string]int),
		qualityCounts:   make(map[string]int),
	}
	if study.Output == nil {
		return m
	}

	for _, summary := range study.Output.BacktestSummaries {
		m.sessionCount++
		m.totalRecommendations += summary.RecommendationsGenerated
		m.totalClosed += summary.RecommendationsClosed
		m.buyCount += summary.BuyCount
		m.watchCount += summary.WatchCount
		m.avoidCount += summary.AvoidCount
		m.optimizationRuns += summary.OptimizationRuns
		m.walkForwardRuns += summary.WalkForwardRuns
		m.monteCarloRuns += summary.MonteCarloRuns
		m.alertsGenerated += summary.AlertsGenerated
		m.researchReports += summary.ResearchReportsGenerated

		if summary.WinRate > 0 || summary.RecommendationsGenerated > 0 {
			m.winRates = append(m.winRates, summary.WinRate)
		}
		if summary.RecommendationsGenerated > 0 {
			m.returns = append(m.returns, summary.AverageReturn)
			m.confidences = append(m.confidences, summary.AverageConfidence)
		}

		mergeCounts(m.symbolCounts, summary.SymbolDistribution)
		mergeCounts(m.timeframeCounts, summary.TimeframeDistribution)
		mergeCounts(m.qualityCounts, summary.QualityDistribution)
	}

	if len(study.Output.BacktestSummaries) > 0 {
		var confSum, retSum float64
		confCount := 0
		for _, summary := range study.Output.BacktestSummaries {
			if summary.RecommendationsGenerated > 0 {
				confSum += summary.AverageConfidence
				retSum += summary.AverageReturn
				confCount++
			}
			if summary.HighestConfidence > m.highConfidence {
				m.highConfidence = summary.HighestConfidence
			}
			if confCount == 1 || summary.LowestConfidence < m.lowConfidence {
				m.lowConfidence = summary.LowestConfidence
			}
		}
		if confCount > 0 {
			m.avgConfidence = confSum / float64(confCount)
			m.avgReturn = retSum / float64(confCount)
		}

		var winSum, lossSum float64
		winCount := 0
		for _, wr := range m.winRates {
			winSum += wr
			winCount++
		}
		if winCount > 0 {
			m.winRate = winSum / float64(winCount)
		}
		for _, summary := range study.Output.BacktestSummaries {
			if summary.LossRate > 0 {
				lossSum += summary.LossRate
			}
		}
		if winCount > 0 {
			m.lossRate = lossSum / float64(winCount)
		}
	}

	return m
}

func mergeCounts(dst map[string]int, src map[string]int) {
	for k, v := range src {
		dst[k] += v
	}
}

func buildExecutiveSummary(study laboratory.Study, m studyMetrics) string {
	return fmt.Sprintf(
		"Research report for study %s (%s) evaluating strategy %s across %d symbol(s) over %d backtest session(s). "+
			"Status: %s. Average return: %.4f, win rate: %.4f, average confidence: %.4f. "+
			"Generated %d recommendations with %d optimization, %d walk-forward, and %d Monte Carlo runs.",
		study.Name,
		study.StudyID,
		study.Strategy,
		len(study.Symbols),
		m.sessionCount,
		study.Status,
		m.avgReturn,
		m.winRate,
		m.avgConfidence,
		m.totalRecommendations,
		m.optimizationRuns,
		m.walkForwardRuns,
		m.monteCarloRuns,
	)
}

func buildStrategyOverview(study laboratory.Study) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Strategy: %s (version %s)\n", study.Strategy, study.ResearchVersion))
	b.WriteString(fmt.Sprintf("Period: %s to %s\n", study.StartTime.Format("2006-01-02"), study.EndTime.Format("2006-01-02")))
	b.WriteString(fmt.Sprintf("Symbols: %s\n", strings.Join(study.Symbols, ", ")))
	timeframes := make([]string, 0, len(study.Timeframes))
	for _, tf := range study.Timeframes {
		timeframes = append(timeframes, string(tf))
	}
	b.WriteString(fmt.Sprintf("Timeframes: %s\n", strings.Join(timeframes, ", ")))
	if len(study.Parameters) > 0 {
		b.WriteString("Parameters:\n")
		for k, v := range study.Parameters {
			b.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
	}
	if study.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", study.Description))
	}
	return strings.TrimRight(b.String(), "\n")
}

func buildStrengths(m studyMetrics) string {
	var strengths []string
	if m.winRate >= 0.5 {
		strengths = append(strengths, fmt.Sprintf("Win rate of %.2f%% meets or exceeds the 50%% benchmark", m.winRate*100))
	}
	if m.avgReturn > 0 {
		strengths = append(strengths, fmt.Sprintf("Positive average return of %.4f across sessions", m.avgReturn))
	}
	if m.avgConfidence >= 0.6 {
		strengths = append(strengths, fmt.Sprintf("Strong average confidence of %.4f", m.avgConfidence))
	}
	if m.walkForwardRuns > 0 {
		strengths = append(strengths, fmt.Sprintf("Walk-forward validation completed with %d run(s)", m.walkForwardRuns))
	}
	if m.monteCarloRuns > 0 {
		strengths = append(strengths, fmt.Sprintf("Monte Carlo stress testing completed with %d run(s)", m.monteCarloRuns))
	}
	if m.optimizationRuns > 0 {
		strengths = append(strengths, fmt.Sprintf("Parameter optimization executed with %d run(s)", m.optimizationRuns))
	}
	if len(strengths) == 0 {
		return "No significant strengths identified from available metrics."
	}
	return strings.Join(strengths, "; ")
}

func buildWeaknesses(study laboratory.Study, m studyMetrics) string {
	var weaknesses []string
	if study.Status == laboratory.StudyStatusFailed {
		weaknesses = append(weaknesses, "Study execution failed")
		if study.Error != "" {
			weaknesses = append(weaknesses, study.Error)
		}
	}
	if m.winRate < 0.5 && m.sessionCount > 0 {
		weaknesses = append(weaknesses, fmt.Sprintf("Win rate of %.2f%% below 50%% benchmark", m.winRate*100))
	}
	if m.avgReturn < 0 {
		weaknesses = append(weaknesses, fmt.Sprintf("Negative average return of %.4f", m.avgReturn))
	}
	if m.avgConfidence < 0.4 {
		weaknesses = append(weaknesses, fmt.Sprintf("Low average confidence of %.4f", m.avgConfidence))
	}
	if m.avoidCount > m.buyCount {
		weaknesses = append(weaknesses, fmt.Sprintf("AVOID signals (%d) exceed BUY signals (%d)", m.avoidCount, m.buyCount))
	}
	if m.walkForwardRuns == 0 {
		weaknesses = append(weaknesses, "No walk-forward validation runs recorded")
	}
	if m.monteCarloRuns == 0 {
		weaknesses = append(weaknesses, "No Monte Carlo stress tests recorded")
	}
	if len(weaknesses) == 0 {
		return "No significant weaknesses identified from available metrics."
	}
	return strings.Join(weaknesses, "; ")
}

func buildRankedSymbols(counts map[string]int, best bool) string {
	if len(counts) == 0 {
		return "No symbol performance data available."
	}
	ranked := rankByCount(counts, best)
	parts := make([]string, 0, len(ranked))
	for i, entry := range ranked {
		parts = append(parts, fmt.Sprintf("%d. %s (%d recommendations)", i+1, entry.key, entry.count))
	}
	return strings.Join(parts, "; ")
}

func buildRankedTimeframes(counts map[string]int, best bool) string {
	if len(counts) == 0 {
		return "No timeframe performance data available."
	}
	ranked := rankByCount(counts, best)
	parts := make([]string, 0, len(ranked))
	for i, entry := range ranked {
		parts = append(parts, fmt.Sprintf("%d. %s (%d recommendations)", i+1, entry.key, entry.count))
	}
	return strings.Join(parts, "; ")
}

type rankedEntry struct {
	key   string
	count int
}

func rankByCount(counts map[string]int, highestFirst bool) []rankedEntry {
	entries := make([]rankedEntry, 0, len(counts))
	for k, v := range counts {
		entries = append(entries, rankedEntry{key: k, count: v})
	}
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			swap := false
			if highestFirst && entries[j].count > entries[i].count {
				swap = true
			}
			if !highestFirst && entries[j].count < entries[i].count {
				swap = true
			}
			if swap {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	return entries
}

func buildParameterSensitivity(study laboratory.Study, m studyMetrics) string {
	if len(study.Parameters) == 0 {
		return "No strategy parameters configured for sensitivity analysis."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Strategy parameters (%d): ", len(study.Parameters)))
	parts := make([]string, 0, len(study.Parameters))
	for k, v := range study.Parameters {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteString(fmt.Sprintf(". Optimization runs: %d. ", m.optimizationRuns))
	if m.optimizationRuns > 0 {
		b.WriteString("Parameter space was explored through the optimization engine.")
	} else {
		b.WriteString("No optimization runs recorded; parameter sensitivity remains untested.")
	}
	return b.String()
}

func buildConsistencyAnalysis(m studyMetrics) string {
	if len(m.winRates) < 2 {
		return fmt.Sprintf("Limited consistency data: %d session(s) with win rate %.4f. Additional sessions needed for cross-session consistency assessment.", m.sessionCount, m.winRate)
	}
	var minWR, maxWR float64
	minWR = m.winRates[0]
	maxWR = m.winRates[0]
	for _, wr := range m.winRates {
		if wr < minWR {
			minWR = wr
		}
		if wr > maxWR {
			maxWR = wr
		}
	}
	spread := maxWR - minWR
	level := "high"
	if spread > 0.3 {
		level = "low"
	} else if spread > 0.15 {
		level = "moderate"
	}
	return fmt.Sprintf(
		"Win rate range across %d sessions: %.4f to %.4f (spread %.4f). Consistency level: %s.",
		len(m.winRates), minWR, maxWR, spread, level,
	)
}

func buildWalkForwardSummary(study laboratory.Study) string {
	if study.Output == nil || len(study.Output.WalkForwardSummaries) == 0 {
		return "No walk-forward analysis runs recorded for this study."
	}
	var total int
	parts := make([]string, 0, len(study.Output.WalkForwardSummaries))
	for _, wf := range study.Output.WalkForwardSummaries {
		total += wf.WalkForwardRuns
		parts = append(parts, fmt.Sprintf("%s: %d run(s)", wf.BacktestID, wf.WalkForwardRuns))
	}
	return fmt.Sprintf("Total walk-forward runs: %d. Per session: %s.", total, strings.Join(parts, "; "))
}

func buildMonteCarloSummary(study laboratory.Study) string {
	if study.Output == nil || len(study.Output.MonteCarloSummaries) == 0 {
		return "No Monte Carlo simulations recorded for this study."
	}
	var total int
	parts := make([]string, 0, len(study.Output.MonteCarloSummaries))
	for _, mc := range study.Output.MonteCarloSummaries {
		total += mc.MonteCarloRuns
		parts = append(parts, fmt.Sprintf("%s: %d run(s)", mc.BacktestID, mc.MonteCarloRuns))
	}
	return fmt.Sprintf("Total Monte Carlo runs: %d. Per session: %s.", total, strings.Join(parts, "; "))
}

func buildRiskAnalysis(m studyMetrics) string {
	return fmt.Sprintf(
		"Risk profile: %d recommendations (%d closed), loss rate %.4f, AVOID signals %d, alerts %d. "+
			"Return range across sessions indicates average return of %.4f with win rate %.4f.",
		m.totalRecommendations,
		m.totalClosed,
		m.lossRate,
		m.avoidCount,
		m.alertsGenerated,
		m.avgReturn,
		m.winRate,
	)
}

func buildConfidenceAssessment(m studyMetrics) string {
	if m.avgConfidence == 0 && len(m.confidences) == 0 {
		return "No confidence data available for assessment."
	}
	level := "moderate"
	if m.avgConfidence >= 0.7 {
		level = "high"
	} else if m.avgConfidence < 0.4 {
		level = "low"
	}
	return fmt.Sprintf(
		"Average confidence: %.4f (%s). Range: %.4f to %.4f across %d session(s).",
		m.avgConfidence, level, m.lowConfidence, m.highConfidence, len(m.confidences),
	)
}

func buildMarketRegimeSuitability(m studyMetrics) string {
	if len(m.qualityCounts) == 0 {
		return "Insufficient quality evaluation data to assess market regime suitability."
	}
	parts := make([]string, 0, len(m.qualityCounts))
	for k, v := range m.qualityCounts {
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	return fmt.Sprintf("Quality distribution: %s. Strategy performance varies across evaluated recommendation outcomes.", strings.Join(parts, ", "))
}

func buildSuggestedImprovements(study laboratory.Study, m studyMetrics) string {
	var suggestions []string
	if m.walkForwardRuns == 0 {
		suggestions = append(suggestions, "Run walk-forward validation to assess temporal stability")
	}
	if m.monteCarloRuns == 0 {
		suggestions = append(suggestions, "Execute Monte Carlo simulations to evaluate tail risk")
	}
	if m.optimizationRuns == 0 && len(study.Parameters) > 0 {
		suggestions = append(suggestions, "Optimize strategy parameters to improve risk-adjusted returns")
	}
	if m.winRate < 0.5 {
		suggestions = append(suggestions, "Review entry criteria to improve win rate above 50%")
	}
	if m.avgConfidence < 0.5 {
		suggestions = append(suggestions, "Enhance signal filtering to raise confidence scores")
	}
	if len(suggestions) == 0 {
		return "Current metrics are within acceptable ranges; consider incremental parameter tuning."
	}
	return strings.Join(suggestions, "; ")
}

func buildSuggestedFutureExperiments(study laboratory.Study, m studyMetrics) string {
	var experiments []string
	if len(study.Symbols) == 1 {
		experiments = append(experiments, fmt.Sprintf("Expand symbol universe beyond %s", study.Symbols[0]))
	}
	if len(study.Timeframes) == 1 {
		experiments = append(experiments, fmt.Sprintf("Test additional timeframes beyond %s", string(study.Timeframes[0])))
	}
	experiments = append(experiments, "Compare against alternative parameter sets via study versioning")
	if m.sessionCount == 1 {
		experiments = append(experiments, "Extend date range to cover multiple market regimes")
	}
	experiments = append(experiments, "Run cross-strategy comparison via Strategy Laboratory")
	return strings.Join(experiments, "; ")
}

func buildOverallVerdict(study laboratory.Study, m studyMetrics) string {
	if study.Status == laboratory.StudyStatusFailed {
		return "FAIL — study execution did not complete successfully"
	}
	score := 0
	if m.winRate >= 0.5 {
		score++
	}
	if m.avgReturn > 0 {
		score++
	}
	if m.avgConfidence >= 0.5 {
		score++
	}
	if m.walkForwardRuns > 0 {
		score++
	}
	if m.monteCarloRuns > 0 {
		score++
	}
	if m.avoidCount <= m.buyCount {
		score++
	}

	switch {
	case score >= 5:
		return "PASS — strategy demonstrates strong metrics across return, confidence, and validation dimensions"
	case score >= 3:
		return "CONDITIONAL PASS — strategy shows promise but requires additional validation or refinement"
	default:
		return "FAIL — strategy metrics fall below minimum research thresholds"
	}
}
