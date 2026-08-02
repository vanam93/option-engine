package indicators_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
)

func TestEMAWarmUpAndIncremental(t *testing.T) {
	ema := indicators.NewEMA(3)
	closes := []float64{10, 20, 30, 40}

	r1 := ema.Update(closes[0])
	require.False(t, r1.WarmedUp)
	require.Equal(t, 1, r1.Samples)

	r2 := ema.Update(closes[1])
	require.False(t, r2.WarmedUp)

	r3 := ema.Update(closes[2])
	require.True(t, r3.WarmedUp)
	require.InDelta(t, 20.0, r3.Value, 0.0001)

	r4 := ema.Update(closes[3])
	require.True(t, r4.WarmedUp)
	require.InDelta(t, 30.0, r4.Value, 0.0001)
}

func TestSMAWarmUpAndIncremental(t *testing.T) {
	sma := indicators.NewSMA(3)
	closes := []float64{10, 20, 30, 40}

	r1 := sma.Update(closes[0])
	require.False(t, r1.WarmedUp)

	r2 := sma.Update(closes[1])
	require.False(t, r2.WarmedUp)

	r3 := sma.Update(closes[2])
	require.True(t, r3.WarmedUp)
	require.InDelta(t, 20.0, r3.Value, 0.0001)

	r4 := sma.Update(closes[3])
	require.True(t, r4.WarmedUp)
	require.InDelta(t, 30.0, r4.Value, 0.0001)
}
