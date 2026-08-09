package supertrend_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/supertrend"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestSuperTrendSignalsAfterWarmup(t *testing.T) {
	s := supertrend.New(map[string]any{"atr_period": 3, "multiplier": 2.0})
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.False(t, testutil.HasDecision(decisions[:5], strategylib.DecisionBuy))
}

func TestSuperTrendIgnoreInvalid(t *testing.T) {
	s := supertrend.NewDefault()
	sig := s.Evaluate(strategylib.Context{
		Candle:   testutil.MakeCandle(0, 0),
		Position: strategylib.PositionFlat,
	})
	require.Equal(t, strategylib.DecisionIgnore, sig.Decision)
}
