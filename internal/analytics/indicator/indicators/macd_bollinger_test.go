package indicators_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
)

func TestMACDWarmUp(t *testing.T) {
	macd := indicators.NewMACD(2, 3, 2)
	closes := []float64{10, 20, 30, 40}

	r1 := macd.Update(closes[0])
	require.False(t, r1.WarmedUp)

	r2 := macd.Update(closes[1])
	require.False(t, r2.WarmedUp)

	r3 := macd.Update(closes[2])
	require.False(t, r3.WarmedUp)

	r4 := macd.Update(closes[3])
	require.True(t, r4.WarmedUp)
	require.InDelta(t, r4.MACD-r4.Signal, r4.Histogram, 0.0001)
}

func TestMACDIncrementalUpdate(t *testing.T) {
	macd := indicators.NewMACD(2, 3, 2)
	closes := []float64{10, 20, 30, 40, 50}

	var last indicators.MACDResult
	for _, close := range closes {
		last = macd.Update(close)
	}
	require.True(t, last.WarmedUp)
	require.NotZero(t, last.MACD)
	require.NotZero(t, last.Signal)
}

func TestBollingerWarmUp(t *testing.T) {
	bb := indicators.NewBollinger(3, 2.0)
	closes := []float64{10, 20, 30}

	r1 := bb.Update(closes[0])
	require.False(t, r1.WarmedUp)

	r2 := bb.Update(closes[1])
	require.False(t, r2.WarmedUp)

	r3 := bb.Update(closes[2])
	require.True(t, r3.WarmedUp)
	require.InDelta(t, 20.0, r3.Middle, 0.0001)
	require.Greater(t, r3.Upper, r3.Middle)
	require.Less(t, r3.Lower, r3.Middle)
	require.Greater(t, r3.BandWidth, 0.0)
}

func TestBollingerIncrementalUpdate(t *testing.T) {
	bb := indicators.NewBollinger(3, 2.0)
	closes := []float64{10, 20, 30, 40, 50}

	var last indicators.BollingerResult
	for _, close := range closes {
		last = bb.Update(close)
	}
	require.True(t, last.WarmedUp)
	require.InDelta(t, 40.0, last.Middle, 0.0001)
	require.Greater(t, last.Upper, last.Middle)
	require.Less(t, last.Lower, last.Middle)
}
