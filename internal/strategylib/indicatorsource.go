package strategylib

import (
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
)

// IndicatorSource provides precomputed indicator series for research backtests.
// Implementations must be safe for concurrent reads after lazy series creation.
type IndicatorSource interface {
	Len() int
	EMA(period, bar int) (indicators.Result, bool)
	SMA(period, bar int) (indicators.Result, bool)
	RSI(period, bar int) (indicators.Result, bool)
	ATR(period, bar int) (indicators.Result, bool)
	MACD(fast, slow, signal, bar int) (indicators.MACDResult, bool)
	Bollinger(period int, stddev float64, bar int) (indicators.BollingerResult, bool)
	ADX(period, bar int) (indicators.ADXResult, bool)
	Donchian(period, bar int) (indicators.DonchianResult, bool)
	SuperTrend(atrPeriod int, multiplier float64, bar int) (indicators.SuperTrendResult, bool)
	OpeningRange(windowMinutes, bar int) (indicators.OpeningRangeResult, bool)
	SessionVWAP(bar int) (indicators.VWAPResult, bool)
}

// HasIndicatorStore reports whether context carries a shared indicator cache.
func (c Context) HasIndicatorStore() bool {
	return c.IndicatorStore != nil && c.BarIndex >= 0
}
