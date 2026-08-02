package signal

import (
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
)

const (
	strategyEMACross   = "ema_cross"
	strategyMACDCross  = "macd_cross"
	strategyRSI        = "rsi"
	strategyBollinger  = "bollinger"
)

const (
	confidenceCrossover = 0.75
	confidenceRSI       = 0.70
	confidenceBollinger = 0.70
)

// Evaluator applies enabled signal rules to indicator updates.
type Evaluator struct {
	cfg Config
}

// NewEvaluator creates a rule evaluator from configuration.
func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{cfg: cfg.withDefaults()}
}

// Evaluate returns zero or more signals for a single indicator update.
func (e *Evaluator) Evaluate(value domainindicator.IndicatorValue, cache *Cache) []GeneratedSignal {
	if value.Values["warmed_up"] <= 0 {
		return nil
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	switch value.Name {
	case domainindicator.EMA:
		if e.cfg.EMACross.Enabled {
			if sig, ok := e.evalEMACross(value, cache); ok {
				return []GeneratedSignal{sig}
			}
		}
	case domainindicator.MACD:
		if e.cfg.MACDCross.Enabled {
			if sig, ok := e.evalMACDCross(value, cache); ok {
				return []GeneratedSignal{sig}
			}
		}
	case domainindicator.RSI:
		if e.cfg.RSI.Enabled {
			if sig, ok := e.evalRSI(value); ok {
				return []GeneratedSignal{sig}
			}
		}
	case domainindicator.BollingerBands:
		if e.cfg.Bollinger.Enabled {
			if sig, ok := e.evalBollinger(value); ok {
				return []GeneratedSignal{sig}
			}
		}
	}
	return nil
}

func (e *Evaluator) evalEMACross(value domainindicator.IndicatorValue, cache *Cache) (GeneratedSignal, bool) {
	period := int(value.Values["period"])
	if period != e.cfg.EMACross.FastPeriod && period != e.cfg.EMACross.SlowPeriod {
		return GeneratedSignal{}, false
	}

	cache.setEMAValue(value, period, value.Values["value"])
	fast, hasFast := cache.emaValue(value.Symbol, value.Timeframe, e.cfg.EMACross.FastPeriod)
	slow, hasSlow := cache.emaValue(value.Symbol, value.Timeframe, e.cfg.EMACross.SlowPeriod)
	if !hasFast || !hasSlow {
		return GeneratedSignal{}, false
	}

	key := seriesKey{symbol: value.Symbol, timeframe: value.Timeframe}
	state := cache.emaCrossState(key)
	defer func() {
		state.prevFast = fast
		state.prevSlow = slow
		state.hasPrev = true
	}()

	if !state.hasPrev {
		return GeneratedSignal{}, false
	}

	indicators := map[string]float64{
		"ema_fast": fast,
		"ema_slow": slow,
	}
	switch {
	case state.prevFast <= state.prevSlow && fast > slow:
		return newGeneratedSignal(strategyEMACross, Buy, value.Symbol, value.Timeframe, confidenceCrossover, value.ComputedAt, indicators), true
	case state.prevFast >= state.prevSlow && fast < slow:
		return newGeneratedSignal(strategyEMACross, Sell, value.Symbol, value.Timeframe, confidenceCrossover, value.ComputedAt, indicators), true
	default:
		return GeneratedSignal{}, false
	}
}

func (e *Evaluator) evalMACDCross(value domainindicator.IndicatorValue, cache *Cache) (GeneratedSignal, bool) {
	macdLine := value.Values["macd"]
	signalLine := value.Values["signal"]

	key := seriesKey{symbol: value.Symbol, timeframe: value.Timeframe}
	state := cache.macdCrossState(key)
	defer func() {
		state.prevMACD = macdLine
		state.prevSignal = signalLine
		state.hasPrev = true
	}()

	if !state.hasPrev {
		return GeneratedSignal{}, false
	}

	indicators := map[string]float64{
		"macd":   macdLine,
		"signal": signalLine,
	}
	switch {
	case state.prevMACD <= state.prevSignal && macdLine > signalLine:
		return newGeneratedSignal(strategyMACDCross, Buy, value.Symbol, value.Timeframe, confidenceCrossover, value.ComputedAt, indicators), true
	case state.prevMACD >= state.prevSignal && macdLine < signalLine:
		return newGeneratedSignal(strategyMACDCross, Sell, value.Symbol, value.Timeframe, confidenceCrossover, value.ComputedAt, indicators), true
	default:
		return GeneratedSignal{}, false
	}
}

func (e *Evaluator) evalRSI(value domainindicator.IndicatorValue) (GeneratedSignal, bool) {
	rsi := value.Values["value"]
	indicators := map[string]float64{"rsi": rsi}

	switch {
	case rsi < e.cfg.RSI.Oversold:
		return newGeneratedSignal(strategyRSI, Buy, value.Symbol, value.Timeframe, confidenceRSI, value.ComputedAt, indicators), true
	case rsi > e.cfg.RSI.Overbought:
		return newGeneratedSignal(strategyRSI, Sell, value.Symbol, value.Timeframe, confidenceRSI, value.ComputedAt, indicators), true
	default:
		return GeneratedSignal{}, false
	}
}

func (e *Evaluator) evalBollinger(value domainindicator.IndicatorValue) (GeneratedSignal, bool) {
	percentB := value.Values["percent_b"]
	indicators := map[string]float64{
		"upper":     value.Values["upper"],
		"middle":    value.Values["middle"],
		"lower":     value.Values["lower"],
		"percent_b": percentB,
	}

	switch {
	case percentB < 0:
		return newGeneratedSignal(strategyBollinger, Buy, value.Symbol, value.Timeframe, confidenceBollinger, value.ComputedAt, indicators), true
	case percentB > 1:
		return newGeneratedSignal(strategyBollinger, Sell, value.Symbol, value.Timeframe, confidenceBollinger, value.ComputedAt, indicators), true
	default:
		return GeneratedSignal{}, false
	}
}
