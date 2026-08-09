package strategy

import (
	"math"
	"time"
)

// InputSignal mirrors the SignalGenerated payload consumed by the strategy engine.
type InputSignal struct {
	Symbol     string
	Timeframe  string
	Signal     string
	Strategy   string
	Confidence float64
	Timestamp  time.Time
}

// Evaluator applies enabled strategies to accumulated signal state.
type Evaluator struct {
	cfg Config
}

// NewEvaluator creates a strategy evaluator from configuration.
func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{cfg: cfg.WithDefaults()}
}

// Process stores a signal, evaluates enabled strategies, and updates position state.
func (e *Evaluator) Process(sig InputSignal, cache *Cache) []StrategyDecision {
	cache.mu.Lock()
	key := cache.update(sig)
	decisions := e.Evaluate(key, cache)
	for _, d := range decisions {
		cache.recordDecision(key, d.Decision, d.Timestamp)
	}
	cache.mu.Unlock()
	return decisions
}

// Evaluate returns zero or more decisions for the current series state.
func (e *Evaluator) Evaluate(key seriesKey, cache *Cache) []StrategyDecision {
	state := cache.state(key)
	if state == nil {
		return nil
	}

	var out []StrategyDecision
	if e.cfg.TrendFollowing.Enabled {
		if d, ok := e.evalTrendFollowing(key, state); ok {
			out = append(out, d)
		}
	}
	if e.cfg.MeanReversion.Enabled {
		if d, ok := e.evalMeanReversion(key, state); ok {
			out = append(out, d)
		}
	}
	if e.cfg.Breakout.Enabled {
		if d, ok := e.evalBreakout(key, state); ok {
			out = append(out, d)
		}
	}
	return out
}

func (e *Evaluator) evalTrendFollowing(key seriesKey, state *seriesState) (StrategyDecision, bool) {
	ema, hasEMA := state.signals[signalEMACross]
	macd, hasMACD := state.signals[signalMACDCross]
	if !hasEMA || !hasMACD {
		return StrategyDecision{}, false
	}

	at := laterTime(ema.at, macd.at)
	reason, decision, confidence, ok := e.alignEntrySignals(ema, macd, state.position)
	if !ok {
		return newStrategyDecision(strategyTrendFollowing, Hold, key.symbol, key.timeframe, confidence, at, reason), true
	}
	if decision == Hold {
		return newStrategyDecision(strategyTrendFollowing, Hold, key.symbol, key.timeframe, confidence, at, reason), true
	}
	if !e.positionAllowsEntry(decision, state.position) {
		return StrategyDecision{}, false
	}
	return newStrategyDecision(strategyTrendFollowing, decision, key.symbol, key.timeframe, confidence, at, reason), true
}

func (e *Evaluator) evalMeanReversion(key seriesKey, state *seriesState) (StrategyDecision, bool) {
	rsi, hasRSI := state.signals[signalRSI]
	boll, hasBoll := state.signals[signalBollinger]
	if !hasRSI || !hasBoll {
		return StrategyDecision{}, false
	}

	at := laterTime(rsi.at, boll.at)
	reason, decision, confidence, ok := e.alignEntrySignals(rsi, boll, state.position)
	if !ok {
		return newStrategyDecision(strategyMeanReversion, Hold, key.symbol, key.timeframe, confidence, at, reason), true
	}
	if decision == Hold {
		return newStrategyDecision(strategyMeanReversion, Hold, key.symbol, key.timeframe, confidence, at, reason), true
	}
	if !e.positionAllowsEntry(decision, state.position) {
		return StrategyDecision{}, false
	}
	return newStrategyDecision(strategyMeanReversion, decision, key.symbol, key.timeframe, confidence, at, reason), true
}

func (e *Evaluator) evalBreakout(key seriesKey, state *seriesState) (StrategyDecision, bool) {
	at := state.lastDecisionAt
	if at.IsZero() {
		return StrategyDecision{}, false
	}
	return newStrategyDecision(strategyBreakout, Hold, key.symbol, key.timeframe, 0, at, "breakout strategy not implemented"), true
}

func (e *Evaluator) alignEntrySignals(primary, confirm storedSignal, position positionState) (string, Decision, float64, bool) {
	confidence := math.Min(primary.confidence, confirm.confidence)
	if confidence < e.cfg.MinConfidence {
		return "insufficient confidence", Hold, confidence, true
	}

	switch {
	case primary.signal == inputBuy && confirm.signal == inputBuy:
		return "bullish alignment", LongEntry, confidence, true
	case primary.signal == inputSell && confirm.signal == inputSell:
		return "bearish alignment", ShortEntry, confidence, true
	case primary.signal == inputExitLong && position == positionLong:
		return "exit long signal", LongExit, confidence, true
	case primary.signal == inputExitShort && position == positionShort:
		return "exit short signal", ShortExit, confidence, true
	case signalsConflict(primary.signal, confirm.signal):
		return "conflicting signals", Hold, confidence, true
	default:
		return "", Hold, confidence, false
	}
}

func (e *Evaluator) positionAllowsEntry(decision Decision, position positionState) bool {
	switch decision {
	case LongEntry:
		return position == positionFlat || position == positionShort
	case ShortEntry:
		return position == positionFlat || position == positionLong
	default:
		return true
	}
}

func signalsConflict(a, b string) bool {
	if a == inputBuy && b == inputSell {
		return true
	}
	if a == inputSell && b == inputBuy {
		return true
	}
	return false
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
