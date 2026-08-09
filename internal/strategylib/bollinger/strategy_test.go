package bollinger_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/bollinger"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestBollingerMeanReversion(t *testing.T) {
	s := bollinger.New(map[string]any{"period": 5, "stddev": 2.0})
	stable := []float64{100, 100, 100, 100, 100, 90, 95}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(stable), strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
}

func TestBollingerExit(t *testing.T) {
	s := bollinger.New(map[string]any{"period": 3, "stddev": 2.0})
	closes := []float64{100, 100, 90, 100}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionLong)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionExit))
}
