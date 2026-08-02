package quality

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// Status mirrors recommendation lifecycle states.
type Status string

const (
	StatusCreated         Status = "CREATED"
	StatusActive          Status = "ACTIVE"
	StatusWatch           Status = "WATCH"
	StatusExitRecommended Status = "EXIT_RECOMMENDED"
	StatusClosed          Status = "CLOSED"
)

// Level identifies the recommendation tier.
type Level string

const (
	LevelStrongBuy Level = "STRONG_BUY"
	LevelBuy       Level = "BUY"
	LevelWatch     Level = "WATCH"
	LevelAvoid     Level = "AVOID"
)

// Outcome classifies the final recommendation result.
type Outcome string

const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeFailed  Outcome = "FAILED"
	OutcomeNeutral Outcome = "NEUTRAL"
	OutcomeExpired Outcome = "EXPIRED"
)

// Classification labels recommendation quality.
type Classification string

const (
	ClassificationExcellent Classification = "EXCELLENT"
	ClassificationGood      Classification = "GOOD"
	ClassificationAverage   Classification = "AVERAGE"
	ClassificationPoor      Classification = "POOR"
	ClassificationFailed    Classification = "FAILED"
)

// IntelligenceInput mirrors recommendation.intelligence.updated payload fields used by the engine.
type IntelligenceInput struct {
	RecommendationID    string
	Symbol                string
	Timeframe             string
	Strategy              string
	RecommendationLevel   Level
	Confidence            float64
	CurrentStatus         Status
	GeneratedAt           time.Time
}

// StateInput mirrors recommendation.state.updated payload fields used by the engine.
type StateInput struct {
	RecommendationID string
	Symbol           string
	Timeframe        string
	Strategy         string
	CurrentStatus    Status
	Confidence       float64
	UpdatedAt        time.Time
}

// PriceStatistics holds computed price metrics for a tracked recommendation.
type PriceStatistics struct {
	EntryPrice       float64       `json:"entry_price"`
	LatestPrice      float64       `json:"latest_price"`
	HighestPrice     float64       `json:"highest_price"`
	LowestPrice      float64       `json:"lowest_price"`
	ExitPrice        float64       `json:"exit_price,omitempty"`
	AbsoluteReturn   float64       `json:"absolute_return"`
	PercentageReturn float64       `json:"percentage_return"`
	HoldingDuration  time.Duration `json:"holding_duration"`
}

// QualityMetrics holds excursion and drawdown metrics.
type QualityMetrics struct {
	MFE              float64 `json:"mfe"`
	MAE              float64 `json:"mae"`
	MaximumReturn    float64 `json:"maximum_return"`
	MaximumDrawdown  float64 `json:"maximum_drawdown"`
	ReturnPct        float64 `json:"return_pct"`
	HoldingDuration  int64   `json:"holding_duration_ms"`
}

// QualityReport is the complete evaluation for one recommendation.
type QualityReport struct {
	RecommendationID    string         `json:"recommendation_id"`
	Symbol              string         `json:"symbol"`
	Timeframe           string         `json:"timeframe"`
	Strategy            string         `json:"strategy"`
	RecommendationLevel Level          `json:"recommendation_level"`
	Confidence          float64        `json:"confidence"`
	EntryTime           time.Time      `json:"entry_time"`
	ExitTime            *time.Time     `json:"exit_time,omitempty"`
	CurrentStatus       Status         `json:"current_status"`
	Outcome             Outcome        `json:"outcome"`
	Classification      Classification `json:"classification"`
	QualityScore        float64        `json:"quality_score"`
	PriceStatistics     PriceStatistics `json:"price_statistics"`
	QualityMetrics      QualityMetrics  `json:"quality_metrics"`
	TrackingActive      bool           `json:"tracking_active"`
	Completed           bool           `json:"completed"`
	EvaluatedAt         time.Time      `json:"evaluated_at"`
}

// RecommendationQualityUpdated is published on recommendation.quality.updated.
type RecommendationQualityUpdated struct {
	RecommendationID string        `json:"recommendation_id"`
	Symbol           string        `json:"symbol"`
	Timeframe        string        `json:"timeframe"`
	Strategy         string        `json:"strategy"`
	Report           QualityReport `json:"report"`
	GeneratedAt      time.Time     `json:"generated_at"`
}

// CandleUpdate carries parsed candle data for price tracking.
type CandleUpdate struct {
	Symbol    string
	Timeframe string
	Candle    market.Candle
	At        time.Time
}
