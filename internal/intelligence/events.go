package intelligence

import "time"

// Status mirrors recommendation lifecycle states from state updates.
type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusActive          Status = "ACTIVE"
	StatusWatch           Status = "WATCH"
	StatusExitRecommended Status = "EXIT_RECOMMENDED"
	StatusClosed          Status = "CLOSED"
)

// Level identifies the recommendation tier derived from confidence.
type Level string

const (
	LevelStrongBuy Level = "STRONG_BUY"
	LevelBuy       Level = "BUY"
	LevelWatch     Level = "WATCH"
	LevelAvoid     Level = "AVOID"
)

// TimelineEntry mirrors timeline entries from recommendation.state.updated.
type TimelineEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Event         string    `json:"event"`
	Reason        string    `json:"reason"`
	PreviousValue string    `json:"previous_value"`
	NewValue      string    `json:"new_value"`
}

// StateUpdate mirrors the recommendation.state.updated payload consumed by the engine.
type StateUpdate struct {
	RecommendationID     string
	Symbol               string
	Timeframe            string
	Strategy             string
	Recommendation       Level
	CurrentStatus        Status
	Confidence           float64
	LatestTimelineEntry  TimelineEntry
	Summary              string
	Reasons              []string
	SupportingIndicators []string
	SupportingStrategies []string
	OptimizationSummary  string
	WalkForwardSummary   string
	MonteCarloSummary    string
	Components           map[string]float64
}

// ResearchEvidence holds structured research supporting a recommendation.
type ResearchEvidence struct {
	Signal       string `json:"signal,omitempty"`
	Strategy     string `json:"strategy,omitempty"`
	Risk         string `json:"risk,omitempty"`
	Performance  string `json:"performance,omitempty"`
	Optimization string `json:"optimization,omitempty"`
	WalkForward  string `json:"walk_forward,omitempty"`
	MonteCarlo   string `json:"monte_carlo,omitempty"`
	Drawdown     string `json:"drawdown,omitempty"`
	Freshness    string `json:"freshness,omitempty"`
}

// ConfidenceBreakdown displays per-factor confidence contributions.
type ConfidenceBreakdown struct {
	Signal         *float64 `json:"signal_contribution,omitempty"`
	Strategy       *float64 `json:"strategy_contribution,omitempty"`
	Performance    *float64 `json:"performance_contribution,omitempty"`
	Optimization   *float64 `json:"optimization_contribution,omitempty"`
	WalkForward    *float64 `json:"walk_forward_contribution,omitempty"`
	MonteCarlo     *float64 `json:"monte_carlo_contribution,omitempty"`
	Validation     *float64 `json:"validation_contribution,omitempty"`
	Overall        float64  `json:"overall_confidence"`
}

// IntelligenceDocument is the complete explanation for one recommendation.
type IntelligenceDocument struct {
	RecommendationID        string              `json:"recommendation_id"`
	Symbol                  string              `json:"symbol"`
	Timeframe               string              `json:"timeframe"`
	Strategy                string              `json:"strategy"`
	RecommendationLevel     Level               `json:"recommendation_level"`
	Confidence              float64             `json:"confidence"`
	CurrentStatus           Status              `json:"current_status"`
	CurrentRecommendationState string           `json:"current_recommendation_state"`
	ResearchSummary         string              `json:"research_summary,omitempty"`
	DecisionSummary         string              `json:"decision_summary,omitempty"`
	Explanation             string              `json:"explanation"`
	SupportingFactors       []string            `json:"supporting_factors,omitempty"`
	RiskFactors             []string            `json:"risk_factors,omitempty"`
	TimelineSummary         string              `json:"timeline_summary,omitempty"`
	RecommendationHistory   []TimelineEntry     `json:"recommendation_history,omitempty"`
	ReasonForUpgrade        string              `json:"reason_for_upgrade,omitempty"`
	ReasonForDowngrade      string              `json:"reason_for_downgrade,omitempty"`
	ConfidenceBreakdown     ConfidenceBreakdown `json:"confidence_breakdown"`
	ResearchEvidence        ResearchEvidence    `json:"research_evidence"`
	GeneratedAt             time.Time           `json:"generated_at"`
}

// RecommendationIntelligenceUpdated is published on recommendation.intelligence.updated.
type RecommendationIntelligenceUpdated struct {
	RecommendationID string               `json:"recommendation_id"`
	Symbol           string               `json:"symbol"`
	Timeframe        string               `json:"timeframe"`
	Strategy         string               `json:"strategy"`
	Document         IntelligenceDocument `json:"document"`
	GeneratedAt      time.Time            `json:"generated_at"`
}
