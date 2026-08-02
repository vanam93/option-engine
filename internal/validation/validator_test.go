package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

func testValidatorConfig() Config {
	return Config{
		Enabled:              true,
		SubscriberBuffer:     16,
		MinConfidence:        0.70,
		MinOptimizationScore: 0.60,
		MinWalkforwardScore:  0.60,
		MinMonteCarloScore:   0.60,
		MinWinRate:           0.50,
		MaxDrawdown:          0.20,
		FreshnessSeconds:     300,
		SuppressDuplicates:   true,
	}
}

func validInput(at time.Time) InputRecommendation {
	return InputRecommendation{
		Symbol:               "NIFTY",
		Timeframe:            "1m",
		Recommendation:       recommendation.LevelBuy,
		Confidence:           0.82,
		OptimizationScore:    0.75,
		WalkforwardScore:     0.72,
		MonteCarloScore:      0.68,
		WinRate:              0.58,
		Drawdown:             0.10,
		GeneratedAt:          at,
		hasOptimizationScore: true,
		hasWalkforwardScore:  true,
		hasMonteCarloScore:   true,
		hasWinRate:           true,
		hasDrawdown:          true,
	}
}

func TestValidRecommendation(t *testing.T) {
	validator := NewValidator(testValidatorConfig())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	outcome := validator.Validate(validInput(at), at)

	require.Equal(t, StatusValid, outcome.Result.ValidationStatus)
	require.Empty(t, outcome.Result.RejectionReasons)
}

func TestRejectedLowConfidence(t *testing.T) {
	validator := NewValidator(testValidatorConfig())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	input := validInput(at)
	input.Confidence = 0.55
	input.Recommendation = recommendation.LevelWatch

	outcome := validator.Validate(input, at)

	require.Equal(t, StatusRejected, outcome.Result.ValidationStatus)
	require.Contains(t, outcome.Result.RejectionReasons, "confidence below minimum threshold")
}

func TestRejectedHighDrawdown(t *testing.T) {
	validator := NewValidator(testValidatorConfig())
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	input := validInput(at)
	input.Drawdown = 0.35

	outcome := validator.Validate(input, at)

	require.Equal(t, StatusRejected, outcome.Result.ValidationStatus)
	require.Contains(t, outcome.Result.RejectionReasons, "drawdown exceeds maximum threshold")
}
