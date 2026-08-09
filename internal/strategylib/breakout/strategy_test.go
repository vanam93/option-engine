package breakout_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/breakout"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestBreakoutBuy(t *testing.T) {
	s := breakout.New(map[string]any{"period": 3, "atr_period": 3, "atr_multiple": 0.5})
	closes := []float64{10, 10, 10, 10, 10, 18}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
}

func TestBreakoutIgnoreInsufficientHistory(t *testing.T) {
	s := breakout.NewDefault()
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles([]float64{10, 11}), strategylib.PositionFlat)
	require.Equal(t, strategylib.DecisionIgnore, decisions[len(decisions)-1].Decision)
}
