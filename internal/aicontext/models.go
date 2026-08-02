package aicontext

import (
	"fmt"
	"time"
)

// AIContext is a complete structured context package for future LLM providers.
type AIContext struct {
	ContextID        string              `json:"context_id"`
	ReportID         string              `json:"report_id"`
	StudyID          string              `json:"study_id"`
	ResearchVersion  string              `json:"research_version"`
	Metadata         StudyMetadata       `json:"metadata"`
	Performance      PerformanceSummary  `json:"performance"`
	Optimization     OptimizationContext `json:"optimization"`
	WalkForward      WalkForwardContext  `json:"walk_forward"`
	MonteCarlo       MonteCarloContext   `json:"monte_carlo"`
	Risk             RiskMetrics         `json:"risk"`
	Trades           TradeStatistics     `json:"trades"`
	ResearchReport   ReportContext       `json:"research_report"`
	Timeline         []TimelineEvent     `json:"timeline"`
	KeyFindings      []string            `json:"key_findings"`
	Strengths        []string            `json:"strengths"`
	Weaknesses       []string            `json:"weaknesses"`
	FutureExperiments []string            `json:"future_experiments"`
	ExecutivePrompt  string              `json:"executive_prompt,omitempty"`
	TechnicalPrompt  string              `json:"technical_prompt,omitempty"`
	JSONPrompt       string              `json:"json_prompt,omitempty"`
	GeneratedAt      time.Time           `json:"generated_at"`
}

// StudyMetadata captures study identification and scope.
type StudyMetadata struct {
	StudyID         string            `json:"study_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Strategy        string            `json:"strategy"`
	Parameters      map[string]string `json:"parameters"`
	Symbols         []string          `json:"symbols"`
	Timeframes      []string          `json:"timeframes"`
	StartDate       string            `json:"start_date"`
	EndDate         string            `json:"end_date"`
	Status          string            `json:"status"`
	ResearchVersion string            `json:"research_version"`
	BacktestIDs     []string          `json:"backtest_ids"`
}

// PerformanceSummary aggregates recommendation performance metrics.
type PerformanceSummary struct {
	RecommendationsGenerated int     `json:"recommendations_generated"`
	RecommendationsClosed    int     `json:"recommendations_closed"`
	BuyCount                 int     `json:"buy_count"`
	WatchCount               int     `json:"watch_count"`
	AvoidCount               int     `json:"avoid_count"`
	AverageConfidence        float64 `json:"average_confidence"`
	AverageReturn            float64 `json:"average_return"`
	WinRate                  float64 `json:"win_rate"`
	LossRate                 float64 `json:"loss_rate"`
	SessionCount             int     `json:"session_count"`
}

// OptimizationContext captures optimization activity.
type OptimizationContext struct {
	TotalRuns int                    `json:"total_runs"`
	PerSession []OptimizationSession `json:"per_session"`
}

// OptimizationSession records optimization runs for a backtest session.
type OptimizationSession struct {
	BacktestID string `json:"backtest_id"`
	Runs       int    `json:"runs"`
}

// WalkForwardContext captures walk-forward validation activity.
type WalkForwardContext struct {
	TotalRuns  int                     `json:"total_runs"`
	PerSession []WalkForwardSession    `json:"per_session"`
}

// WalkForwardSession records walk-forward runs for a backtest session.
type WalkForwardSession struct {
	BacktestID string `json:"backtest_id"`
	Runs       int    `json:"runs"`
}

// MonteCarloContext captures Monte Carlo simulation activity.
type MonteCarloContext struct {
	TotalRuns  int                    `json:"total_runs"`
	PerSession []MonteCarloSession    `json:"per_session"`
}

// MonteCarloSession records Monte Carlo runs for a backtest session.
type MonteCarloSession struct {
	BacktestID string `json:"backtest_id"`
	Runs       int    `json:"runs"`
}

// RiskMetrics captures risk-related aggregates.
type RiskMetrics struct {
	AlertsGenerated      int     `json:"alerts_generated"`
	LossRate             float64 `json:"loss_rate"`
	AverageHoldingTimeMs int64   `json:"average_holding_time_ms"`
	ConfidenceLow        float64 `json:"confidence_low"`
	ConfidenceHigh       float64 `json:"confidence_high"`
	Drawdown             float64 `json:"drawdown"`
}

// TradeStatistics captures trade outcome statistics.
type TradeStatistics struct {
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	TotalTrades   int     `json:"total_trades"`
	Drawdown      float64 `json:"drawdown"`
	Expectancy    float64 `json:"expectancy"`
	ProfitFactor  float64 `json:"profit_factor"`
}

// ReportContext embeds research report findings for LLM consumption.
type ReportContext struct {
	ReportID         string `json:"report_id"`
	Analyzer         string `json:"analyzer"`
	ExecutiveSummary string `json:"executive_summary"`
	OverallVerdict   string `json:"overall_verdict"`
	Consistency      string `json:"consistency_analysis"`
	RiskAnalysis     string `json:"risk_analysis"`
	Confidence       string `json:"confidence_assessment"`
	RegimeSuitability string `json:"market_regime_suitability"`
}

// TimelineEvent records a significant point in the study lifecycle.
type TimelineEvent struct {
	Timestamp string `json:"timestamp"`
	Event     string `json:"event"`
	Detail    string `json:"detail"`
}

// JSONContext is the machine-readable context for JSON prompt generation.
type JSONContext struct {
	ContextID        string              `json:"context_id"`
	ReportID         string              `json:"report_id"`
	StudyID          string              `json:"study_id"`
	ResearchVersion  string              `json:"research_version"`
	Metadata         StudyMetadata       `json:"metadata"`
	Performance      PerformanceSummary  `json:"performance"`
	Optimization     OptimizationContext `json:"optimization"`
	WalkForward      WalkForwardContext  `json:"walk_forward"`
	MonteCarlo       MonteCarloContext   `json:"monte_carlo"`
	Risk             RiskMetrics         `json:"risk"`
	Trades           TradeStatistics     `json:"trades"`
	ResearchReport   ReportContext       `json:"research_report"`
	Timeline         []TimelineEvent     `json:"timeline"`
	KeyFindings      []string            `json:"key_findings"`
	Strengths        []string            `json:"strengths"`
	Weaknesses       []string            `json:"weaknesses"`
	FutureExperiments []string            `json:"future_experiments"`
	GeneratedAt      string              `json:"generated_at"`
}

func generateContextID(studyID, reportID string, at time.Time) string {
	suffix := studyID
	if len(suffix) > 12 {
		suffix = suffix[len(suffix)-12:]
	}
	return fmt.Sprintf("CTX-%s-%s", at.UTC().Format("20060102T150405"), suffix)
}
