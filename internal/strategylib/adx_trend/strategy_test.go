package adx_trend_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/adx_trend"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestADXTrendWarmupIgnore(t *testing.T) {
	s := adx_trend.NewDefault()
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles([]float64{100, 101}), strategylib.PositionFlat)
	require.Equal(t, strategylib.DecisionIgnore, decisions[len(decisions)-1].Decision)
}

func TestADXTrendSignals(t *testing.T) {
	s := adx_trend.New(map[string]any{"period": 3, "threshold": 10})
	closes := make([]float64, 40)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.NotEmpty(t, decisions)
}
