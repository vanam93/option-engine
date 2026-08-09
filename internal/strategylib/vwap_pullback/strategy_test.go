package vwap_pullback_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/vwap_pullback"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

func TestVWAPPullbackRuns(t *testing.T) {
	s := vwap_pullback.New(map[string]any{"trend_period": 3, "tolerance": 0.01})
	open := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	series := make([]market.Candle, 10)
	for i := range series {
		series[i] = testutil.MakeSessionCandle(open.Add(time.Duration(i)*5*time.Minute), 100+float64(i), 1000)
	}
	decisions := testutil.EvaluateSeries(s, series, strategylib.PositionFlat)
	require.NotEmpty(t, decisions)
}
