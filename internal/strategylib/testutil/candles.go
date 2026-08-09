package testutil

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// MakeCandle builds a synthetic OHLCV candle for strategy tests.
func MakeCandle(close float64, index int) market.Candle {
	open := close - 0.25
	return market.Candle{
		Symbol:    "TEST",
		Timeframe: market.TF5m,
		Open:      open,
		High:      close + 1.0,
		Low:       close - 1.0,
		Close:     close,
		Volume:    1000,
		OpenTime:  time.Date(2024, 1, 15, 9, 15+index*5, 0, 0, time.UTC),
		CloseTime: time.Date(2024, 1, 15, 9, 15+index*5+5, 0, 0, time.UTC),
	}
}

// MakeSessionCandle builds a candle at a specific session open time.
func MakeSessionCandle(openTime time.Time, close float64, volume int64) market.Candle {
	return market.Candle{
		Symbol:    "TEST",
		Timeframe: market.TF5m,
		Open:      close - 0.25,
		High:      close + 1.0,
		Low:       close - 1.0,
		Close:     close,
		Volume:    volume,
		OpenTime:  openTime,
		CloseTime: openTime.Add(5 * time.Minute),
	}
}

// ClosesToCandles maps close prices to synthetic candles.
func ClosesToCandles(closes []float64) []market.Candle {
	out := make([]market.Candle, len(closes))
	for i, c := range closes {
		out[i] = MakeCandle(c, i)
	}
	return out
}
