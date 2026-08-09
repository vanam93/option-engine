package ema_pullback_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_pullback"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestEMAPullbackEntry(t *testing.T) {
	s := ema_pullback.New(map[string]any{"fast": 3, "slow": 5, "tolerance": 0.01})
	closes := make([]float64, 15)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	closes[len(closes)-1] = closes[len(closes)-2]
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.NotEmpty(t, decisions)
}

func TestEMAPullbackExit(t *testing.T) {
	s := ema_pullback.New(map[string]any{"fast": 3, "slow": 5})
	closes := []float64{110, 108, 105, 100, 95}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionLong)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionExit))
}
