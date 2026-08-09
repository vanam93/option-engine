package opening_range_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/opening_range"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestOpeningRangeBreakoutBuy(t *testing.T) {
	s := opening_range.New(map[string]any{"window_minutes": 15})
	open := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	series := []market.Candle{
		testutil.MakeSessionCandle(open, 102, 1000),
		testutil.MakeSessionCandle(open.Add(5*time.Minute), 103, 1000),
		testutil.MakeSessionCandle(open.Add(20*time.Minute), 110, 1000),
	}
	decisions := testutil.EvaluateSeries(s, series, strategylib.PositionFlat)
	require.True(t, testutil.HasDecision(decisions, strategylib.DecisionBuy))
}

func TestOpeningRangeIgnoreDuringWindow(t *testing.T) {
	s := opening_range.New(map[string]any{"window_minutes": 15})
	open := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	series := []market.Candle{
		testutil.MakeSessionCandle(open, 110, 1000),
	}
	decisions := testutil.EvaluateSeries(s, series, strategylib.PositionFlat)
	require.Equal(t, strategylib.DecisionIgnore, decisions[0].Decision)
}
