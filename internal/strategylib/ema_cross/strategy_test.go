package ema_cross_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/ema_cross"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestEMAcrossIgnoreInsufficientHistory(t *testing.T) {
	s := ema_cross.NewDefault()
	candles := testutil.ClosesToCandles([]float64{100, 101, 102})
	decisions := testutil.EvaluateSeries(s, candles, strategylib.PositionFlat)
	require.False(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
	require.False(t, testutil.HasDecision(decisions, strategylib.DecisionSell))
}

func TestEMAcrossBuyOnUptrendCross(t *testing.T) {
	s := ema_cross.New(map[string]any{"fast": 3, "slow": 5})
	closes := []float64{100, 99, 98, 97, 96, 95, 94, 93, 92, 91, 90, 95, 100, 105, 110, 115, 120}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
}

func TestEMAcrossExitOnOppositeCross(t *testing.T) {
	s := ema_cross.New(map[string]any{"fast": 3, "slow": 5})
	up := make([]float64, 20)
	for i := range up {
		up[i] = 100 + float64(i)
	}
	down := make([]float64, 10)
	for i := range down {
		down[i] = 120 - float64(i)*2
	}
	closes := append(up, down...)
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionLong)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionExit))
}

func TestEMAcrossIgnoreInvalidCandle(t *testing.T) {
	s := ema_cross.NewDefault()
	c := testutil.MakeCandle(0, 0)
	c.Close = 0
	sig := s.Evaluate(strategylib.Context{Candle: c, Position: strategylib.PositionFlat})
	require.Equal(t, strategylib.DecisionIgnore, sig.Decision)
}

func TestEMAcrossMetadata(t *testing.T) {
	s := ema_cross.NewDefault()
	meta := s.Metadata()
	require.Equal(t, ema_cross.Name, meta.Name)
	require.NotEmpty(t, meta.ParameterRanges)
	require.Contains(t, meta.SupportedTimeframes, "5m")
}
