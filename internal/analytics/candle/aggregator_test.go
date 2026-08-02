package candle_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/candle"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

func TestBucketStartMinuteTimeframes(t *testing.T) {
	loc := time.UTC
	ts := time.Date(2026, 8, 2, 10, 7, 30, 0, loc)

	open, err := candle.BucketStart(ts, market.TF5m, loc)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 2, 10, 5, 0, 0, loc), open)

	closeAt, err := candle.BucketClose(open, market.TF5m)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 2, 10, 10, 0, 0, loc), closeAt)
}

func TestAggregatorClosesBarOnBucketRollover(t *testing.T) {
	loc := time.UTC
	agg := candle.NewAggregator(loc, candle.AggregatorOptions{})

	tick1 := market.Tick{
		Symbol:     "NIFTY",
		LTP:        100,
		ProviderTS: time.Date(2026, 8, 2, 10, 0, 10, 0, loc),
	}
	closed, stats, err := agg.Update(tick1, []market.Timeframe{market.TF1m})
	require.NoError(t, err)
	require.Empty(t, closed)
	require.Zero(t, stats.Rejected)

	tick2 := market.Tick{
		Symbol:     "NIFTY",
		LTP:        105,
		ProviderTS: time.Date(2026, 8, 2, 10, 1, 5, 0, loc),
	}
	closed, stats, err = agg.Update(tick2, []market.Timeframe{market.TF1m})
	require.NoError(t, err)
	require.Len(t, closed, 1)
	require.Equal(t, 100.0, closed[0].Open)
	require.Equal(t, 100.0, closed[0].Close)
	require.Equal(t, market.TF1m, closed[0].Timeframe)
	require.Zero(t, stats.Rejected)
}

func TestAggregatorVWAPUsesCumulativeVolumeDelta(t *testing.T) {
	loc := time.UTC
	agg := candle.NewAggregator(loc, candle.AggregatorOptions{VolumeMode: candle.VolumeCumulative})
	tfs := []market.Timeframe{market.TF1m}
	ts := time.Date(2026, 8, 2, 10, 0, 0, 0, loc)

	_, _, err := agg.Update(market.Tick{Symbol: "NIFTY", LTP: 100, Volume: 10, ProviderTS: ts}, tfs)
	require.NoError(t, err)
	_, _, err = agg.Update(market.Tick{Symbol: "NIFTY", LTP: 110, Volume: 30, ProviderTS: ts.Add(5 * time.Second)}, tfs)
	require.NoError(t, err)

	closed, _, err := agg.Update(market.Tick{Symbol: "NIFTY", LTP: 120, Volume: 50, ProviderTS: ts.Add(time.Minute)}, tfs)
	require.NoError(t, err)
	require.Len(t, closed, 1)
	require.InDelta(t, 106.66666666666667, closed[0].VWAP, 0.001)
	require.Equal(t, int64(30), closed[0].Volume)
}

func TestAggregatorCumulativeVolumeUnchangedSkipsVolume(t *testing.T) {
	loc := time.UTC
	agg := candle.NewAggregator(loc, candle.AggregatorOptions{VolumeMode: candle.VolumeCumulative})
	ts := time.Date(2026, 8, 2, 10, 0, 0, 0, loc)
	tfs := []market.Timeframe{market.TF1m}

	_, _, err := agg.Update(market.Tick{Symbol: "NIFTY", LTP: 100, Volume: 1000, ProviderTS: ts}, tfs)
	require.NoError(t, err)
	_, stats, err := agg.Update(market.Tick{Symbol: "NIFTY", LTP: 101, Volume: 1000, ProviderTS: ts.Add(time.Second)}, tfs)
	require.NoError(t, err)
	require.Zero(t, stats.Rejected)

	snap := agg.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, int64(1000), snap[0].Volume)
	require.Equal(t, int64(2), snap[0].TradeCount)
}

func TestAggregatorRejectsOlderBucket(t *testing.T) {
	loc := time.UTC
	agg := candle.NewAggregator(loc, candle.AggregatorOptions{OrderPolicy: candle.OrderRejectOlder})
	tfs := []market.Timeframe{market.TF1m}

	_, _, err := agg.Update(market.Tick{
		Symbol: "NIFTY", LTP: 100, ProviderTS: time.Date(2026, 8, 2, 10, 5, 0, 0, loc),
	}, tfs)
	require.NoError(t, err)

	_, stats, err := agg.Update(market.Tick{
		Symbol: "NIFTY", LTP: 90, ProviderTS: time.Date(2026, 8, 2, 10, 0, 0, 0, loc),
	}, tfs)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Rejected)

	snap := agg.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, 100.0, snap[0].Close)
}

func TestAggregatorFlushClearsBuilders(t *testing.T) {
	loc := time.UTC
	agg := candle.NewAggregator(loc, candle.AggregatorOptions{})
	ts := time.Date(2026, 8, 2, 10, 0, 0, 0, loc)

	_, _, err := agg.Update(market.Tick{Symbol: "NIFTY", LTP: 100, ProviderTS: ts}, []market.Timeframe{market.TF1m})
	require.NoError(t, err)
	require.Equal(t, 1, agg.ActiveBuilders())

	flushed := agg.Flush()
	require.Len(t, flushed, 1)
	require.Zero(t, agg.ActiveBuilders())
}

func TestAggregatorIdleEviction(t *testing.T) {
	loc := time.UTC
	agg := candle.NewAggregator(loc, candle.AggregatorOptions{IdleEvict: time.Minute})
	tfs := []market.Timeframe{market.TF1m, market.TF5m}

	_, _, err := agg.Update(market.Tick{
		Symbol: "NIFTY", LTP: 100, ProviderTS: time.Date(2026, 8, 2, 10, 0, 0, 0, loc),
	}, tfs)
	require.NoError(t, err)
	require.Equal(t, 2, agg.ActiveBuilders())

	closed, stats, err := agg.Update(market.Tick{
		Symbol: "NIFTY", LTP: 110, ProviderTS: time.Date(2026, 8, 2, 10, 2, 0, 0, loc),
	}, []market.Timeframe{market.TF1m})
	require.NoError(t, err)
	require.Equal(t, 2, stats.Evicted)
	require.Len(t, closed, 2)
	require.Equal(t, 1, agg.ActiveBuilders())
}
