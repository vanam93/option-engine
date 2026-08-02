package airesearch

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ResearchReport is a deterministic AI-generated research report for a completed study.
type ResearchReport struct {
	ReportID         string         `json:"report_id"`
	StudyID          string         `json:"study_id"`
	ResearchVersion  string         `json:"research_version"`
	Analyzer         string         `json:"analyzer"`
	Prompt           string         `json:"prompt"`
	Sections         ReportSections `json:"sections"`
	FormattedText    string         `json:"formatted_text"`
	GeneratedAt      time.Time      `json:"generated_at"`
}

// ReportSections holds structured report content.
type ReportSections struct {
	ExecutiveSummary          string `json:"executive_summary"`
	StrategyOverview          string `json:"strategy_overview"`
	Strengths                 string `json:"strengths"`
	Weaknesses                string `json:"weaknesses"`
	BestPerformingSymbols     string `json:"best_performing_symbols"`
	WorstPerformingSymbols    string `json:"worst_performing_symbols"`
	BestTimeframes            string `json:"best_timeframes"`
	WorstTimeframes           string `json:"worst_timeframes"`
	ParameterSensitivity      string `json:"parameter_sensitivity"`
	ConsistencyAnalysis       string `json:"consistency_analysis"`
	WalkForwardSummary        string `json:"walk_forward_summary"`
	MonteCarloSummary         string `json:"monte_carlo_summary"`
	RiskAnalysis              string `json:"risk_analysis"`
	ConfidenceAssessment      string `json:"confidence_assessment"`
	MarketRegimeSuitability   string `json:"market_regime_suitability"`
	SuggestedImprovements       string `json:"suggested_improvements"`
	SuggestedFutureExperiments string `json:"suggested_future_experiments"`
	OverallVerdict            string `json:"overall_verdict"`
}

func generateReportID(at time.Time) string {
	return fmt.Sprintf("REP-%s-%s", at.UTC().Format("20060102T150405"), uuid.NewString()[:8])
}
