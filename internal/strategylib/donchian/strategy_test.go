package donchian_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/donchian"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestDonchianBreakoutBuy(t *testing.T) {
	s := donchian.New(map[string]any{"period": 3})
	closes := []float64{10, 11, 12, 16}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
}

func TestDonchianExit(t *testing.T) {
	s := donchian.New(map[string]any{"period": 3})
	closes := []float64{10, 11, 12, 8}
	decisions := testutil.EvaluateSeries(s, testutil.ClosesToCandles(closes), strategylib.PositionLong)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionExit))
}
