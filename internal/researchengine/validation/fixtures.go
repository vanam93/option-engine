package validation

import (
	"fmt"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib/testutil"
)

// EquityCurve returns closes for a controlled drawdown scenario.
// Peak 130, trough 115 from 100 start => max DD 15 on 130 peak = 11.54% of peak, 15% of initial 100.
func EquityCurveDrawdown() []market.Candle {
	return testutil.ClosesToCandles([]float64{100, 110, 120, 115, 130, 115})
}

// LongWinSeries produces a single winning long trade fixture.
func LongWinSeries() []market.Candle {
	base := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	out := make([]market.Candle, 30)
	for i := range out {
		close := 100.0
		if i >= 10 {
			close = 110.0
		}
		out[i] = testutil.MakeSessionCandle(base.Add(time.Duration(i*5)*time.Minute), close, 1000)
	}
	return out
}

// MultiSessionBreakoutDays builds days with breakout then exit to allow re-entry.
func MultiSessionBreakoutDays(days int) []market.Candle {
	if days < 2 {
		days = 2
	}
	var out []market.Candle
	for day := 0; day < days; day++ {
		base := time.Date(2024, 1, 15+day, 9, 15, 0, 0, time.UTC)
		for i := 0; i < 25; i++ {
			close := 100.0
			high := 101.0
			low := 99.0
			if i >= 4 && i < 8 {
				close = 110.0
				high = 111.0
			}
			if i >= 12 {
				close = 97.0
				low = 96.0
			}
			out = append(out, market.Candle{
				Symbol:    "TEST",
				Timeframe: market.TF5m,
				Open:      close - 0.25,
				High:      high,
				Low:       low,
				Close:     close,
				Volume:    1000,
				OpenTime:  base.Add(time.Duration(i*5) * time.Minute),
				CloseTime: base.Add(time.Duration(i*5+5) * time.Minute),
			})
		}
	}
	return out
}

// EMABullishCrossCloses returns closes that produce a fast/slow EMA bullish cross.
func EMABullishCrossCloses() []float64 {
	return []float64{100, 99, 98, 97, 96, 95, 94, 93, 92, 91, 90, 95, 100, 105, 110, 115, 120}
}

// FormatCandleSummary returns a short dataset description.
func FormatCandleSummary(candles []market.Candle) string {
	if len(candles) == 0 {
		return "empty dataset"
	}
	zeroVol := 0
	for _, c := range candles {
		if c.Volume <= 0 {
			zeroVol++
		}
	}
	return fmt.Sprintf("%d candles, %d with zero volume", len(candles), zeroVol)
}
