package indicators_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
)

func TestRSIWarmUpAndIncremental(t *testing.T) {
	rsi := indicators.NewRSI(2)
	closes := []float64{10, 12, 11, 13}

	r1 := rsi.Update(closes[0])
	require.False(t, r1.WarmedUp)

	r2 := rsi.Update(closes[1])
	require.False(t, r2.WarmedUp)

	r3 := rsi.Update(closes[2])
	require.True(t, r3.WarmedUp)
	require.InDelta(t, 66.66666666666666, r3.Value, 0.0001)

	r4 := rsi.Update(closes[3])
	require.True(t, r4.WarmedUp)
	require.InDelta(t, 85.71428571428571, r4.Value, 0.0001)
}

func TestATRWarmUpAndIncremental(t *testing.T) {
	atr := indicators.NewATR(2)

	type bar struct{ h, l, c float64 }
	bars := []bar{
		{12, 8, 10},
		{14, 9, 13},
		{15, 11, 12},
		{16, 10, 14},
	}

	r1 := atr.Update(bars[0].h, bars[0].l, bars[0].c)
	require.False(t, r1.WarmedUp)

	r2 := atr.Update(bars[1].h, bars[1].l, bars[1].c)
	require.True(t, r2.WarmedUp)
	first := (4.0 + 5.0) / 2

	r3 := atr.Update(bars[2].h, bars[2].l, bars[2].c)
	require.True(t, r3.WarmedUp)
	tr3 := 4.0
	expected3 := (first*1 + tr3) / 2
	require.InDelta(t, expected3, r3.Value, 0.0001)

	r4 := atr.Update(bars[3].h, bars[3].l, bars[3].c)
	require.True(t, r4.WarmedUp)
	tr4 := 6.0
	expected4 := (expected3*1 + tr4) / 2
	require.InDelta(t, expected4, r4.Value, 0.0001)
}
