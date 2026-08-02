package aicontext

// Executive prompt templates.
const (
	tmplExecutiveHeader = "AI Research Context — Executive Summary\n"
	tmplExecutiveBody   = "Study: %s (%s)\nStrategy: %s | Version: %s\nPeriod: %s to %s\nSymbols: %s | Timeframes: %s\n\nVerdict: %s\n\nPerformance: return=%s win_rate=%s confidence=%s\nTrades: %d wins / %d losses | expectancy=%s profit_factor=%s\n\nKey Findings:\n%s\n"
)

// Technical prompt templates.
const (
	tmplTechnicalHeader      = "AI Research Context — Technical Report\n"
	tmplTechnicalMetadata    = "--- Study Metadata ---\nStudy ID: %s\nName: %s\nStrategy: %s\nVersion: %s\nStatus: %s\nPeriod: %s to %s\nSymbols: %s\nTimeframes: %s\nParameters: %s\nBacktest Sessions: %s\n\n"
	tmplTechnicalPerformance = "--- Performance Summary ---\nRecommendations: %d generated, %d closed\nLevels: BUY=%d WATCH=%d AVOID=%d\nConfidence: avg=%s range=%s-%s\nReturns: avg=%s win_rate=%s loss_rate=%s\nSessions: %d\n\n"
	tmplTechnicalOptimization = "--- Optimization Summary ---\nTotal runs: %d\n%s\n"
	tmplTechnicalWalkForward  = "--- Walk-Forward Summary ---\nTotal runs: %d\n%s\n"
	tmplTechnicalMonteCarlo   = "--- Monte Carlo Summary ---\nTotal runs: %d\n%s\n"
	tmplTechnicalRisk         = "--- Risk Metrics ---\nAlerts: %d | Loss rate: %s | Holding time: %dms\nDrawdown: %s | Confidence range: %s-%s\n\n"
	tmplTechnicalTrades       = "--- Trade Statistics ---\nTotal: %d | Wins: %d | Losses: %d\nDrawdown: %s | Expectancy: %s | Profit factor: %s\n\n"
	tmplTechnicalReport       = "--- Research Report ---\nAnalyzer: %s\nExecutive: %s\nVerdict: %s\nConsistency: %s\nRisk: %s\nConfidence: %s\nRegime: %s\n\n"
	tmplTechnicalFindings     = "--- Key Findings ---\n%s\n\n"
	tmplTechnicalStrengths    = "--- Strengths ---\n%s\n\n"
	tmplTechnicalWeaknesses   = "--- Weaknesses ---\n%s\n\n"
	tmplTechnicalExperiments  = "--- Future Experiments ---\n%s\n\n"
	tmplTechnicalTimeline     = "--- Timeline ---\n%s\n"
)

// Section line templates for per-session detail blocks.
const (
	tmplSessionLine       = "%s: %d run(s)"
	tmplTimelineLine      = "[%s] %s — %s"
	tmplBulletLine        = "- %s"
	tmplParameterLine     = "%s=%s"
	tmplParameterJoin     = ", "
)

// JSON prompt envelope (no markdown).
const tmplJSONEnvelope = "{\"type\":\"ai_research_context\",\"version\":\"1\",\"data\":%s}"
