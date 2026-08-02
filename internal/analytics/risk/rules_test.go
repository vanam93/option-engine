package risk_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/risk"
)

func baseCfg() risk.Config {
	return risk.Config{
		Enabled:          true,
		MinConfidence:    0.70,
		MaxPositions:     5,
		MaxTradesPerDay:  20,
		DefaultQuantity:  1,
		DayResetTimezone: "UTC",
	}
}

func decision(action string, confidence float64, at time.Time) risk.InputDecision {
	return risk.InputDecision{
		Symbol:     "NIFTY",
		Timeframe:  "1m",
		Decision:   action,
		Strategy:   "trend_following",
		Confidence: confidence,
		Timestamp:  at,
		Reason:     "bullish alignment",
	}
}

func TestValidLongEntryApproved(t *testing.T) {
	cfg := baseCfg()
	eval := risk.NewEvaluator(cfg)
	cache, err := risk.NewCache("UTC")
	require.NoError(t, err)

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	intent := eval.Process(decision("LONG_ENTRY", 0.75, at), cache)

	require.Equal(t, risk.Approved, intent.Status)
	require.Equal(t, "LONG_ENTRY", intent.Action)
	require.Equal(t, 1, intent.Quantity)
}

func TestLowConfidenceRejected(t *testing.T) {
	cfg := baseCfg()
	eval := risk.NewEvaluator(cfg)
	cache, err := risk.NewCache("UTC")
	require.NoError(t, err)

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	intent := eval.Process(decision("LONG_ENTRY", 0.60, at), cache)

	require.Equal(t, risk.Rejected, intent.Status)
	require.Equal(t, "confidence below threshold", intent.Reason)
}

func TestDuplicateLongPositionRejected(t *testing.T) {
	cfg := baseCfg()
	eval := risk.NewEvaluator(cfg)
	cache, err := risk.NewCache("UTC")
	require.NoError(t, err)

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	first := eval.Process(decision("LONG_ENTRY", 0.75, at), cache)
	require.Equal(t, risk.Approved, first.Status)

	second := eval.Process(decision("LONG_ENTRY", 0.75, at.Add(time.Minute)), cache)
	require.Equal(t, risk.Rejected, second.Status)
	require.Equal(t, "duplicate long position", second.Reason)
}
