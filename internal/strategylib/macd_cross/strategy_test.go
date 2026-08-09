package macd_cross_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/macd_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestMACDCrossSignals(t *testing.T) {
	s := macd_cross.NewDefault()
	closes := make([]float64, 50)
	for i := range closes {
		closes[i] = 100 + float64(i%3)
	}
	for i := 20; i < 50; i++ {
		closes[i] = 100 + float64(i)
	}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.True(t, len(decisions) > 0)
}

func TestMACDCrossIgnoreInvalid(t *testing.T) {
	s := macd_cross.NewDefault()
	c := testutil.MakeCandle(-1, 0)
	sig := s.Evaluate(strategylib.Context{Candle: c, Position: strategylib.PositionFlat})
	require.Equal(t, strategylib.DecisionIgnore, sig.Decision)
}
