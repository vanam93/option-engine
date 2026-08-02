package recommendation

import "time"

// Level identifies the recommendation tier.
type Level string

const (
	LevelStrongBuy Level = "STRONG_BUY"
	LevelBuy       Level = "BUY"
	LevelWatch     Level = "WATCH"
	LevelAvoid     Level = "AVOID"
)

// RecommendationUpdated is the payload published on recommendation.updated events.
type RecommendationUpdated struct {
	Symbol               string    `json:"symbol"`
	Timeframe            string    `json:"timeframe"`
	Recommendation       Level     `json:"recommendation"`
	Confidence           float64   `json:"confidence"`
	Rank                 int       `json:"rank"`
	Reasons              []string  `json:"reasons"`
	SupportingIndicators []string  `json:"supporting_indicators"`
	SupportingStrategies []string  `json:"supporting_strategies"`
	OptimizationSummary  string    `json:"optimization_summary"`
	WalkForwardSummary   string    `json:"walk_forward_summary"`
	MonteCarloSummary    string    `json:"monte_carlo_summary"`
	GeneratedAt          time.Time `json:"generated_at"`
}

// InputOpportunity mirrors the opportunity.updated payload consumed by the engine.
type InputOpportunity struct {
	Symbol         string
	Timeframe      string
	Rank           int
	Confidence     float64
	Classification string
	Score          float64
	Components     map[string]float64
	Timestamp      time.Time
}
