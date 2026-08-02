package validation

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

// Status identifies the validation outcome.
type Status string

const (
	StatusValid    Status = "VALID"
	StatusRejected Status = "REJECTED"
)

// ValidatedRecommendation is the payload published on validated.recommendation events.
type ValidatedRecommendation struct {
	Symbol           string               `json:"symbol"`
	Timeframe        string               `json:"timeframe"`
	Recommendation   recommendation.Level `json:"recommendation"`
	Confidence       float64              `json:"confidence"`
	ValidationStatus Status               `json:"validation_status"`
	RejectionReasons []string             `json:"rejection_reasons"`
	ValidatedAt      time.Time            `json:"validated_at"`
}

// InputRecommendation mirrors the recommendation.updated payload consumed by the engine.
type InputRecommendation struct {
	Symbol               string
	Timeframe            string
	Recommendation       recommendation.Level
	Confidence           float64
	OptimizationScore    float64
	WalkforwardScore     float64
	MonteCarloScore      float64
	WinRate              float64
	Drawdown             float64
	OptimizationSummary  string
	WalkForwardSummary   string
	MonteCarloSummary    string
	GeneratedAt          time.Time
	hasOptimizationScore bool
	hasWalkforwardScore  bool
	hasMonteCarloScore   bool
	hasWinRate           bool
	hasDrawdown          bool
}
