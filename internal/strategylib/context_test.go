package strategylib_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

func TestValidCandle(t *testing.T) {
	require.False(t, strategylib.ValidCandle(market.Candle{}))
	require.True(t, strategylib.ValidCandle(market.Candle{
		Open:  100,
		High:  101,
		Low:   99,
		Close: 100.5,
	}))
	require.False(t, strategylib.ValidCandle(market.Candle{
		Open:  100,
		High:  99,
		Low:   98,
		Close: 100,
	}))
}

func TestContextAllCandles(t *testing.T) {
	h := []market.Candle{{Close: 1}, {Close: 2}}
	ctx := strategylib.Context{History: h, Candle: market.Candle{Close: 3}}
	all := ctx.AllCandles()
	require.Len(t, all, 3)
	require.Equal(t, 3.0, all[2].Close)
}
