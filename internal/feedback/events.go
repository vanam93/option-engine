package feedback

import "time"

// Outcome classifies the final recommendation result.
type Outcome string

const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeFailed  Outcome = "FAILED"
	OutcomeNeutral Outcome = "NEUTRAL"
	OutcomeExpired Outcome = "EXPIRED"
)

// Level identifies the recommendation tier.
type Level string

const (
	LevelStrongBuy Level = "STRONG_BUY"
	LevelBuy       Level = "BUY"
	LevelWatch     Level = "WATCH"
	LevelAvoid     Level = "AVOID"
)

// QualityInput mirrors recommendation.quality.updated payload fields used by the engine.
type QualityInput struct {
	RecommendationID          string
	Symbol                    string
	Timeframe                 string
	Strategy                  string
	Scanner                   string
	RecommendationLevel       Level
	Confidence                float64
	OpportunityClassification string
	RiskApproval              string
	MarketRegime              string
	RecommendationSource      string
	Outcome                   Outcome
	QualityScore              float64
	ReturnPct                 float64
	MFE                       float64
	MAE                       float64
	MaxDrawdown               float64
	HoldingDurationMs         int64
	Completed                 bool
	EvaluatedAt               time.Time
}

// OverallStatistics summarizes platform-wide recommendation learning metrics.
type OverallStatistics struct {
	TotalRecommendations     int     `json:"total_recommendations"`
	Successful               int     `json:"successful"`
	Failed                   int     `json:"failed"`
	Expired                  int     `json:"expired"`
	Neutral                  int     `json:"neutral"`
	SuccessRate              float64 `json:"success_rate"`
	WinRate                  float64 `json:"win_rate"`
	AverageReturn            float64 `json:"average_return"`
	AverageQuality           float64 `json:"average_quality"`
	AverageConfidence        float64 `json:"average_confidence"`
	AverageHoldingDurationMs int64   `json:"average_holding_duration_ms"`
	AverageMFE               float64 `json:"average_mfe"`
	AverageMAE               float64 `json:"average_mae"`
	AverageDrawdown          float64 `json:"average_drawdown"`
	FalsePositives           int     `json:"false_positives"`
	FalseNegatives           int     `json:"false_negatives"`
	ConfidenceAccuracy       float64 `json:"confidence_accuracy"`
}

// StrategyStatistics summarizes learning metrics for one strategy.
type StrategyStatistics struct {
	Strategy                 string  `json:"strategy"`
	Recommendations          int     `json:"recommendations"`
	Wins                     int     `json:"wins"`
	Losses                   int     `json:"losses"`
	Expired                  int     `json:"expired"`
	AverageReturn            float64 `json:"average_return"`
	AverageQuality           float64 `json:"average_quality"`
	AverageConfidence        float64 `json:"average_confidence"`
	AverageHoldingDurationMs int64   `json:"average_holding_duration_ms"`
	WinRate                  float64 `json:"win_rate"`
	SuccessRate              float64 `json:"success_rate"`
}

// SymbolStatistics summarizes learning metrics for one symbol.
type SymbolStatistics struct {
	Symbol                   string  `json:"symbol"`
	Recommendations          int     `json:"recommendations"`
	Wins                     int     `json:"wins"`
	Losses                   int     `json:"losses"`
	AverageReturn            float64 `json:"average_return"`
	AverageQuality           float64 `json:"average_quality"`
	AverageConfidence        float64 `json:"average_confidence"`
	AverageHoldingDurationMs int64   `json:"average_holding_duration_ms"`
}

// TimeframeStatistics summarizes learning metrics for one timeframe.
type TimeframeStatistics struct {
	Timeframe                string  `json:"timeframe"`
	Recommendations          int     `json:"recommendations"`
	AverageReturn            float64 `json:"average_return"`
	AverageQuality           float64 `json:"average_quality"`
	AverageHoldingDurationMs int64   `json:"average_holding_duration_ms"`
	WinRate                  float64 `json:"win_rate"`
}

// ConfidenceBucketStatistics summarizes calibration for one confidence bucket.
type ConfidenceBucketStatistics struct {
	Label                    string  `json:"label"`
	LowerBound               float64 `json:"lower_bound"`
	UpperBound               float64 `json:"upper_bound"`
	Recommendations          int     `json:"recommendations"`
	Successes                int     `json:"successes"`
	Failures                 int     `json:"failures"`
	SuccessRate              float64 `json:"success_rate"`
	AverageReturn            float64 `json:"average_return"`
	AverageQuality           float64 `json:"average_quality"`
	AverageHoldingDurationMs int64   `json:"average_holding_duration_ms"`
}

// RollingWindowStatistics summarizes metrics for a rolling recommendation window.
type RollingWindowStatistics struct {
	WindowSize               int     `json:"window_size"`
	SuccessRate              float64 `json:"success_rate"`
	AverageReturn            float64 `json:"average_return"`
	AverageQuality           float64 `json:"average_quality"`
	AverageConfidence        float64 `json:"average_confidence"`
	AverageHoldingDurationMs int64   `json:"average_holding_duration_ms"`
}

// FeedbackSnapshot is the complete learning state published to downstream consumers.
type FeedbackSnapshot struct {
	Overall               OverallStatistics            `json:"overall"`
	Strategies            []StrategyStatistics         `json:"strategies"`
	Symbols               []SymbolStatistics           `json:"symbols"`
	Timeframes            []TimeframeStatistics        `json:"timeframes"`
	ConfidenceCalibration []ConfidenceBucketStatistics `json:"confidence_calibration"`
	Rolling               []RollingWindowStatistics    `json:"rolling"`
	Timestamp             time.Time                    `json:"timestamp"`
	Version               string                       `json:"version"`
}

// RecommendationFeedbackUpdated is published on recommendation.feedback.updated.
type RecommendationFeedbackUpdated struct {
	Overall               OverallStatistics            `json:"overall"`
	Strategies            []StrategyStatistics         `json:"strategies"`
	Symbols               []SymbolStatistics           `json:"symbols"`
	Timeframes            []TimeframeStatistics        `json:"timeframes"`
	ConfidenceCalibration []ConfidenceBucketStatistics `json:"confidence_calibration"`
	Rolling               []RollingWindowStatistics    `json:"rolling"`
	Timestamp             time.Time                    `json:"timestamp"`
	Version               string                       `json:"version"`
}
