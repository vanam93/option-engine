package indicators_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
)

func TestSuperTrendWarmsUp(t *testing.T) {
	st := indicators.NewSuperTrend(3, 3.0)
	for i := 0; i < 5; i++ {
		close := 100.0 + float64(i)
		res := st.Update(close+1, close-1, close)
		if i < 2 {
			require.False(t, res.WarmedUp)
			continue
		}
		require.True(t, res.WarmedUp)
		require.NotZero(t, res.Value)
	}
}

func TestSessionVWAPAccumulates(t *testing.T) {
	v := indicators.NewSessionVWAP()
	open := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	r1 := v.Update(open, 101, 99, 100, 1000)
	require.True(t, r1.WarmedUp)
	require.InDelta(t, 100.0, r1.Value, 0.01)

	r2 := v.Update(open.Add(5*time.Minute), 103, 101, 102, 1000)
	require.InDelta(t, 101.0, r2.Value, 0.01)
}

func TestDonchianChannel(t *testing.T) {
	ch := indicators.NewDonchianChannel(3)
	highs := []float64{10, 12, 11, 15}
	lows := []float64{8, 9, 7, 10}
	var last indicators.DonchianResult
	for i := range highs {
		last = ch.Update(highs[i], lows[i])
	}
	require.True(t, last.WarmedUp)
	require.Equal(t, 15.0, last.Upper)
	require.Equal(t, 7.0, last.Lower)
}

func TestOpeningRangeWindow(t *testing.T) {
	or := indicators.NewOpeningRange(15)
	open := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	r1 := or.Update(open, 105, 100)
	require.True(t, r1.WithinWindow)

	r2 := or.Update(open.Add(20*time.Minute), 110, 108)
	require.True(t, r2.WindowClosed)
	require.Equal(t, 105.0, r2.High)
	require.Equal(t, 100.0, r2.Low)
}

func TestADXWarmsUp(t *testing.T) {
	adx := indicators.NewADX(3)
	for i := 0; i < 20; i++ {
		high := 100.0 + float64(i)
		low := 99.0 + float64(i)
		close := 99.5 + float64(i)
		res := adx.Update(high, low, close)
		if res.WarmedUp {
			require.Greater(t, res.ADX, 0.0)
			return
		}
	}
	t.Fatal("ADX did not warm up")
}
