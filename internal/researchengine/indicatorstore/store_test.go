package indicatorstore_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/researchengine/indicatorstore"
)

func TestStoreEMACacheShared(t *testing.T) {
	candles := make([]market.Candle, 50)
	for i := range candles {
		candles[i] = market.Candle{
			Symbol:    "TEST",
			Timeframe: market.TF5m,
			Open:      100 + float64(i),
			High:      101 + float64(i),
			Low:       99 + float64(i),
			Close:     100 + float64(i),
			Volume:    1000,
		}
	}

	store := indicatorstore.New(candles)
	r1, ok1 := store.EMA(21, 30)
	r2, ok2 := store.EMA(21, 30)
	require.True(t, ok1)
	require.True(t, ok2)
	require.Equal(t, r1.Value, r2.Value)
	require.True(t, r1.WarmedUp)
}

func TestStoreLazyComputeDifferentPeriods(t *testing.T) {
	candles := make([]market.Candle, 30)
	for i := range candles {
		candles[i] = market.Candle{
			Close: 100 + float64(i),
			High:  101 + float64(i),
			Low:   99 + float64(i),
		}
	}
	store := indicatorstore.New(candles)

	ema9, ok := store.EMA(9, 20)
	require.True(t, ok)
	ema21, ok := store.EMA(21, 20)
	require.True(t, ok)
	require.NotEqual(t, ema9.Value, ema21.Value)
}

func TestStoreLen(t *testing.T) {
	candles := make([]market.Candle, 10)
	store := indicatorstore.New(candles)
	require.Equal(t, 10, store.Len())
}
