package mean_reversion_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/mean_reversion"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestMeanReversionBuy(t *testing.T) {
	s := mean_reversion.New(map[string]any{
		"rsi_period":       3,
		"oversold":         30,
		"overbought":       70,
		"bollinger_period": 5,
		"stddev":           2.0,
	})
	closes := []float64{100, 100, 100, 100, 100, 90, 88}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
}
