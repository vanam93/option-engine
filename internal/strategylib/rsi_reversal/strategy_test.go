package rsi_reversal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/rsi_reversal"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestRSIReversalBuyAndExit(t *testing.T) {
	s := rsi_reversal.New(map[string]any{"period": 3, "oversold": 30, "overbought": 70})
	down := []float64{100, 95, 90, 85, 80, 75, 70}
	up := []float64{72, 75, 78, 82, 85}
	closes := append(down, up...)
	flat := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(flat, strategylib.DecisionBuy))

	long := testutil.EvaluateSeries(s, testutil.ClosesToCandles([]float64{85, 90, 95}), strategylib.PositionLong)
	require.True(t, testutil.HasDecision(long, strategylib.DecisionExit))
}

func TestRSIReversalIgnoreWarmup(t *testing.T) {
	s := rsi_reversal.NewDefault()
	dec := testutil.EvaluateSeries(s, testutil.ClosesToCandles([]float64{100}), strategylib.PositionFlat)
	require.Equal(t, strategylib.DecisionIgnore, dec[0].Decision)
}
