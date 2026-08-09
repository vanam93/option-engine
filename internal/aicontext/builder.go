package aicontext

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/airesearch"
	"github.com/vanam-gangireddy/option-engine/internal/laboratory"
)

// ContextBuilder assembles structured AI context from study and report data.
type ContextBuilder struct {
	cfg Config
}

// NewContextBuilder creates a context builder with the given configuration.
func NewContextBuilder(cfg Config) *ContextBuilder {
	return &ContextBuilder{cfg: cfg.WithDefaults()}
}

// Build constructs a complete AI context package.
func (b *ContextBuilder) Build(study laboratory.Study, report airesearch.ResearchReport, at time.Time) (AIContext, error) {
	ctx := AIContext{
		ContextID:       generateContextID(study.StudyID, report.ReportID, at),
		ReportID:        report.ReportID,
		StudyID:         study.StudyID,
		ResearchVersion: study.ResearchVersion,
		Metadata:        buildMetadata(study),
		Performance:     buildPerformance(study),
		Optimization:    buildOptimization(study),
		WalkForward:     buildWalkForward(study),
		MonteCarlo:      buildMonteCarlo(study),
		Risk:            buildRiskMetrics(study),
		Trades:          buildTradeStatistics(study),
		ResearchReport:  buildReportContext(report),
		Timeline:        buildTimeline(study, report, at),
		KeyFindings:     buildKeyFindings(report),
		Strengths:       splitListItems(report.Sections.Strengths),
		Weaknesses:      splitListItems(report.Sections.Weaknesses),
		FutureExperiments: splitListItems(report.Sections.SuggestedFutureExperiments),
		GeneratedAt:     at.UTC(),
	}

	if b.cfg.ExecutivePrompt {
		ctx.ExecutivePrompt = BuildExecutivePrompt(ctx)
	}
	if b.cfg.TechnicalPrompt {
		ctx.TechnicalPrompt = BuildTechnicalPrompt(ctx)
	}
	if b.cfg.JSONPrompt {
		ctx.JSONPrompt = BuildJSONPrompt(ctx)
	}

	return ctx, nil
}

func buildMetadata(study laboratory.Study) StudyMetadata {
	timeframes := make([]string, 0, len(study.Timeframes))
	for _, tf := range study.Timeframes {
		timeframes = append(timeframes, string(tf))
	}
	sort.Strings(timeframes)

	symbols := append([]string(nil), study.Symbols...)
	sort.Strings(symbols)

	params := sortedParameters(study.Parameters)

	return StudyMetadata{
		StudyID:         study.StudyID,
		Name:            study.Name,
		Description:     study.Description,
		Strategy:        study.Strategy,
		Parameters:      params,
		Symbols:         symbols,
		Timeframes:      timeframes,
		StartDate:       study.StartTime.Format("2006-01-02"),
		EndDate:         study.EndTime.Format("2006-01-02"),
		Status:          string(study.Status),
		ResearchVersion: study.ResearchVersion,
		BacktestIDs:     append([]string(nil), study.BacktestSessionIDs...),
	}
}

func buildPerformance(study laboratory.Study) PerformanceSummary {
	p := PerformanceSummary{}
	if study.Output == nil {
		return p
	}

	var confSum, retSum, winSum, lossSum float64
	confCount := 0

	for _, s := range study.Output.BacktestSummaries {
		p.SessionCount++
		p.RecommendationsGenerated += s.RecommendationsGenerated
		p.RecommendationsClosed += s.RecommendationsClosed
		p.BuyCount += s.BuyCount
		p.WatchCount += s.WatchCount
		p.AvoidCount += s.AvoidCount

		if s.RecommendationsGenerated > 0 {
			confSum += s.AverageConfidence
			retSum += s.AverageReturn
			winSum += s.WinRate
			lossSum += s.LossRate
			confCount++
		}
	}

	if confCount > 0 {
		p.AverageConfidence = normalizeFloat(confSum / float64(confCount))
		p.AverageReturn = normalizeFloat(retSum / float64(confCount))
		p.WinRate = normalizeFloat(winSum / float64(confCount))
		p.LossRate = normalizeFloat(lossSum / float64(confCount))
	}

	return p
}

func buildOptimization(study laboratory.Study) OptimizationContext {
	out := OptimizationContext{}
	if study.Output == nil {
		return out
	}
	for _, o := range study.Output.OptimizationSummaries {
		out.TotalRuns += o.OptimizationRuns
		out.PerSession = append(out.PerSession, OptimizationSession{
			BacktestID: o.BacktestID,
			Runs:       o.OptimizationRuns,
		})
	}
	sort.Slice(out.PerSession, func(i, j int) bool {
		return out.PerSession[i].BacktestID < out.PerSession[j].BacktestID
	})
	return out
}

func buildWalkForward(study laboratory.Study) WalkForwardContext {
	out := WalkForwardContext{}
	if study.Output == nil {
		return out
	}
	for _, wf := range study.Output.WalkForwardSummaries {
		out.TotalRuns += wf.WalkForwardRuns
		out.PerSession = append(out.PerSession, WalkForwardSession{
			BacktestID: wf.BacktestID,
			Runs:       wf.WalkForwardRuns,
		})
	}
	sort.Slice(out.PerSession, func(i, j int) bool {
		return out.PerSession[i].BacktestID < out.PerSession[j].BacktestID
	})
	return out
}

func buildMonteCarlo(study laboratory.Study) MonteCarloContext {
	out := MonteCarloContext{}
	if study.Output == nil {
		return out
	}
	for _, mc := range study.Output.MonteCarloSummaries {
		out.TotalRuns += mc.MonteCarloRuns
		out.PerSession = append(out.PerSession, MonteCarloSession{
			BacktestID: mc.BacktestID,
			Runs:       mc.MonteCarloRuns,
		})
	}
	sort.Slice(out.PerSession, func(i, j int) bool {
		return out.PerSession[i].BacktestID < out.PerSession[j].BacktestID
	})
	return out
}

func buildRiskMetrics(study laboratory.Study) RiskMetrics {
	r := RiskMetrics{}
	if study.Output == nil {
		return r
	}

	var holdingSum time.Duration
	holdingCount := 0
	firstConf := true

	for _, s := range study.Output.BacktestSummaries {
		r.AlertsGenerated += s.AlertsGenerated
		r.LossRate = normalizeFloat(s.LossRate)

		if s.AverageHoldingTime > 0 {
			holdingSum += s.AverageHoldingTime
			holdingCount++
		}
		if firstConf || s.LowestConfidence < r.ConfidenceLow {
			r.ConfidenceLow = normalizeFloat(s.LowestConfidence)
		}
		if firstConf || s.HighestConfidence > r.ConfidenceHigh {
			r.ConfidenceHigh = normalizeFloat(s.HighestConfidence)
		}
		firstConf = false
	}

	if holdingCount > 0 {
		r.AverageHoldingTimeMs = (holdingSum / time.Duration(holdingCount)).Milliseconds()
	}

	trades := buildTradeStatistics(study)
	r.Drawdown = trades.Drawdown

	return r
}

func buildTradeStatistics(study laboratory.Study) TradeStatistics {
	t := TradeStatistics{}
	if study.Output == nil {
		return t
	}

	var minRet, maxRet float64
	retCount := 0
	var winRateSum float64
	var winRateCount int

	for _, s := range study.Output.BacktestSummaries {
		t.TotalTrades += s.RecommendationsGenerated

		for k, v := range s.QualityDistribution {
			switch k {
			case "SUCCESS", "GOOD":
				t.WinningTrades += v
			case "FAILED", "POOR", "BAD":
				t.LosingTrades += v
			}
		}

		if s.RecommendationsGenerated > 0 {
			if retCount == 0 {
				minRet, maxRet = s.AverageReturn, s.AverageReturn
			} else {
				if s.AverageReturn < minRet {
					minRet = s.AverageReturn
				}
				if s.AverageReturn > maxRet {
					maxRet = s.AverageReturn
				}
			}
			retCount++
			winRateSum += s.WinRate
			winRateCount++
		}
	}

	if t.WinningTrades == 0 && t.LosingTrades == 0 {
		closed := 0
		avgWR := 0.0
		for _, s := range study.Output.BacktestSummaries {
			closed += s.RecommendationsClosed
		}
		if winRateCount > 0 {
			avgWR = winRateSum / float64(winRateCount)
		}
		if closed > 0 && avgWR > 0 {
			t.WinningTrades = int(float64(closed) * avgWR)
			t.LosingTrades = closed - t.WinningTrades
		}
	}

	if retCount > 0 {
		t.Drawdown = normalizeFloat(maxRet - minRet)
		if minRet < 0 {
			negDrawdown := normalizeFloat(-minRet)
			if negDrawdown > t.Drawdown {
				t.Drawdown = negDrawdown
			}
		}
	}

	var retSum float64
	for _, s := range study.Output.BacktestSummaries {
		if s.RecommendationsGenerated > 0 {
			retSum += s.AverageReturn
		}
	}
	if winRateCount > 0 {
		t.Expectancy = normalizeFloat(retSum / float64(winRateCount))
	}

	if t.LosingTrades > 0 {
		t.ProfitFactor = normalizeFloat(float64(t.WinningTrades) / float64(t.LosingTrades))
	} else if t.WinningTrades > 0 {
		t.ProfitFactor = normalizeFloat(float64(t.WinningTrades))
	}

	return t
}

func buildReportContext(report airesearch.ResearchReport) ReportContext {
	return ReportContext{
		ReportID:          report.ReportID,
		Analyzer:          report.Analyzer,
		ExecutiveSummary:  report.Sections.ExecutiveSummary,
		OverallVerdict:    report.Sections.OverallVerdict,
		Consistency:       report.Sections.ConsistencyAnalysis,
		RiskAnalysis:      report.Sections.RiskAnalysis,
		Confidence:        report.Sections.ConfidenceAssessment,
		RegimeSuitability: report.Sections.MarketRegimeSuitability,
	}
}

func buildTimeline(study laboratory.Study, report airesearch.ResearchReport, at time.Time) []TimelineEvent {
	events := []TimelineEvent{
		{
			Timestamp: study.CreatedAt.UTC().Format(time.RFC3339),
			Event:     "study_created",
			Detail:    study.Name,
		},
	}

	if study.CompletedAt != nil {
		events = append(events, TimelineEvent{
			Timestamp: study.CompletedAt.UTC().Format(time.RFC3339),
			Event:     "study_completed",
			Detail:    string(study.Status),
		})
	}

	for _, id := range study.BacktestSessionIDs {
		events = append(events, TimelineEvent{
			Timestamp: study.CreatedAt.UTC().Format(time.RFC3339),
			Event:     "backtest_session",
			Detail:    id,
		})
	}

	events = append(events, TimelineEvent{
		Timestamp: report.GeneratedAt.UTC().Format(time.RFC3339),
		Event:     "research_report_generated",
		Detail:    report.ReportID,
	})

	events = append(events, TimelineEvent{
		Timestamp: at.UTC().Format(time.RFC3339),
		Event:     "context_generated",
		Detail:    "ai_context",
	})

	return events
}

func buildKeyFindings(report airesearch.ResearchReport) []string {
	findings := []string{
		report.Sections.OverallVerdict,
	}
	if report.Sections.ExecutiveSummary != "" {
		findings = append(findings, report.Sections.ExecutiveSummary)
	}
	if report.Sections.ConsistencyAnalysis != "" {
		findings = append(findings, report.Sections.ConsistencyAnalysis)
	}
	return findings
}

func splitListItems(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.Split(text, "; ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func sortedParameters(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = params[k]
	}
	return out
}

func normalizeFloat(v float64) float64 {
	return float64(int(v*10000)) / 10000
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.4f", normalizeFloat(v))
}

func formatSessionLines[T any](sessions []T, idFn func(T) string, runsFn func(T) int) string {
	if len(sessions) == 0 {
		return "none"
	}
	lines := make([]string, 0, len(sessions))
	for _, s := range sessions {
		lines = append(lines, fmt.Sprintf(tmplSessionLine, idFn(s), runsFn(s)))
	}
	return strings.Join(lines, "\n")
}

func formatParameters(params map[string]string) string {
	if len(params) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(tmplParameterLine, k, params[k]))
	}
	return strings.Join(parts, tmplParameterJoin)
}

func formatBulletList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf(tmplBulletLine, item))
	}
	return strings.Join(lines, "\n")
}

func formatTimeline(events []TimelineEvent) string {
	if len(events) == 0 {
		return "none"
	}
	lines := make([]string, 0, len(events))
	for _, e := range events {
		lines = append(lines, fmt.Sprintf(tmplTimelineLine, e.Timestamp, e.Event, e.Detail))
	}
	return strings.Join(lines, "\n")
}

func toJSONContext(ctx AIContext) JSONContext {
	return JSONContext{
		ContextID:         ctx.ContextID,
		ReportID:          ctx.ReportID,
		StudyID:           ctx.StudyID,
		ResearchVersion:   ctx.ResearchVersion,
		Metadata:          ctx.Metadata,
		Performance:       ctx.Performance,
		Optimization:      ctx.Optimization,
		WalkForward:       ctx.WalkForward,
		MonteCarlo:        ctx.MonteCarlo,
		Risk:              ctx.Risk,
		Trades:            ctx.Trades,
		ResearchReport:    ctx.ResearchReport,
		Timeline:          ctx.Timeline,
		KeyFindings:       ctx.KeyFindings,
		Strengths:         ctx.Strengths,
		Weaknesses:        ctx.Weaknesses,
		FutureExperiments: ctx.FutureExperiments,
		GeneratedAt:       ctx.GeneratedAt.UTC().Format(time.RFC3339),
	}
}
