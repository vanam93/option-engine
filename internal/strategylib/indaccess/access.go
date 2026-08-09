package indaccess

import (
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator/indicators"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// EMA reads EMA at the current bar from the store or falls back to incremental update.
func EMA(ctx strategylib.Context, period int, fallback *indicators.EMA) indicators.Result {
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.EMA(period, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(ctx.Candle.Close)
}

// EMAAt reads EMA at a specific bar index from the store.
func EMAAt(ctx strategylib.Context, period, bar int) (indicators.Result, bool) {
	if ctx.IndicatorStore == nil || bar < 0 {
		return indicators.Result{}, false
	}
	return ctx.IndicatorStore.EMA(period, bar)
}

// SMA reads SMA at the current bar from the store or falls back to incremental update.
func SMA(ctx strategylib.Context, period int, fallback *indicators.SMA) indicators.Result {
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.SMA(period, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(ctx.Candle.Close)
}

// RSI reads RSI at the current bar from the store or falls back to incremental update.
func RSI(ctx strategylib.Context, period int, fallback *indicators.RSI) indicators.Result {
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.RSI(period, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(ctx.Candle.Close)
}

// ATR reads ATR at the current bar from the store or falls back to incremental update.
func ATR(ctx strategylib.Context, period int, fallback *indicators.ATR) indicators.Result {
	c := ctx.Candle
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.ATR(period, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(c.High, c.Low, c.Close)
}

// MACD reads MACD at the current bar from the store or falls back to incremental update.
func MACD(ctx strategylib.Context, fast, slow, signal int, fallback *indicators.MACD) indicators.MACDResult {
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.MACD(fast, slow, signal, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(ctx.Candle.Close)
}

// MACDAt reads MACD at a specific bar index from the store.
func MACDAt(ctx strategylib.Context, fast, slow, signal, bar int) (indicators.MACDResult, bool) {
	if ctx.IndicatorStore == nil || bar < 0 {
		return indicators.MACDResult{}, false
	}
	return ctx.IndicatorStore.MACD(fast, slow, signal, bar)
}

// Bollinger reads Bollinger bands at the current bar from the store or falls back.
func Bollinger(ctx strategylib.Context, period int, stddev float64, fallback *indicators.Bollinger) indicators.BollingerResult {
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.Bollinger(period, stddev, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(ctx.Candle.Close)
}

// ADX reads ADX at the current bar from the store or falls back.
func ADX(ctx strategylib.Context, period int, fallback *indicators.ADX) indicators.ADXResult {
	c := ctx.Candle
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.ADX(period, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(c.High, c.Low, c.Close)
}

// Donchian reads Donchian channel at the current bar from the store or falls back.
func Donchian(ctx strategylib.Context, period int, fallback *indicators.DonchianChannel) indicators.DonchianResult {
	c := ctx.Candle
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.Donchian(period, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(c.High, c.Low)
}

// DonchianAt reads Donchian channel at a specific bar index from the store.
func DonchianAt(ctx strategylib.Context, period, bar int) (indicators.DonchianResult, bool) {
	if ctx.IndicatorStore == nil || bar < 0 {
		return indicators.DonchianResult{}, false
	}
	return ctx.IndicatorStore.Donchian(period, bar)
}

// SuperTrend reads SuperTrend at the current bar from the store or falls back.
func SuperTrend(ctx strategylib.Context, atrPeriod int, multiplier float64, fallback *indicators.SuperTrend) indicators.SuperTrendResult {
	c := ctx.Candle
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.SuperTrend(atrPeriod, multiplier, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(c.High, c.Low, c.Close)
}

// SuperTrendAt reads SuperTrend at a specific bar index from the store.
func SuperTrendAt(ctx strategylib.Context, atrPeriod int, multiplier float64, bar int) (indicators.SuperTrendResult, bool) {
	if ctx.IndicatorStore == nil || bar < 0 {
		return indicators.SuperTrendResult{}, false
	}
	return ctx.IndicatorStore.SuperTrend(atrPeriod, multiplier, bar)
}

// OpeningRange reads opening range at the current bar from the store or falls back.
func OpeningRange(ctx strategylib.Context, windowMinutes int, fallback *indicators.OpeningRange) indicators.OpeningRangeResult {
	c := ctx.Candle
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.OpeningRange(windowMinutes, ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(c.OpenTime, c.High, c.Low)
}

// SessionVWAP reads session VWAP at the current bar from the store or falls back.
func SessionVWAP(ctx strategylib.Context, fallback *indicators.SessionVWAP) indicators.VWAPResult {
	c := ctx.Candle
	if ctx.HasIndicatorStore() {
		if r, ok := ctx.IndicatorStore.SessionVWAP(ctx.BarIndex); ok {
			return r
		}
	}
	return fallback.Update(c.OpenTime, c.High, c.Low, c.Close, c.Volume)
}
