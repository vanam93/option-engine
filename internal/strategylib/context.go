package strategylib

import (
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

// Context supplies market and portfolio state for strategy evaluation.
// Optional fields may be unset; strategies must ignore unused values.
type Context struct {
	Symbol    string
	Timeframe string
	Candle    market.Candle
	History   []market.Candle
	Position  Position

	MarketRegime      string
	Volatility        float64
	Session           string
	GapPercent        float64
	NewsFlag          bool
	HigherTFTrend     string
	PortfolioExposure float64
	Indicators        map[string]float64
	Timestamp         time.Time
}

// TimestampOrCandle returns the evaluation timestamp from context or candle close time.
func (c Context) TimestampOrCandle() time.Time {
	if !c.Timestamp.IsZero() {
		return c.Timestamp
	}
	if !c.Candle.CloseTime.IsZero() {
		return c.Candle.CloseTime
	}
	return time.Time{}
}

// AllCandles returns history plus the current candle when present.
func (c Context) AllCandles() []market.Candle {
	if c.Candle.Symbol == "" && c.Candle.Close == 0 && c.Candle.Open == 0 {
		return append([]market.Candle(nil), c.History...)
	}
	out := append([]market.Candle(nil), c.History...)
	out = append(out, c.Candle)
	return out
}

// ValidCandle reports whether OHLC data is usable for evaluation.
func ValidCandle(c market.Candle) bool {
	if c.Open <= 0 || c.High <= 0 || c.Low <= 0 || c.Close <= 0 {
		return false
	}
	if c.High < c.Low {
		return false
	}
	if c.Close > c.High || c.Close < c.Low {
		return false
	}
	if c.Open > c.High || c.Open < c.Low {
		return false
	}
	return true
}
