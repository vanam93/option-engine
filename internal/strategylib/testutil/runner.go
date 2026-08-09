package testutil

import (
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/strategylib"
)

// EvaluateSeries runs a strategy across candles with a fixed position state.
func EvaluateSeries(s strategylib.Strategy, candles []market.Candle, pos strategylib.Position) []strategylib.Signal {
	signals := make([]strategylib.Signal, len(candles))
	history := make([]market.Candle, 0, len(candles))
	for i, c := range candles {
		ctx := strategylib.Context{
			Symbol:    c.Symbol,
			Timeframe: string(c.Timeframe),
			Candle:    c,
			History:   append([]market.Candle(nil), history...),
			Position:  pos,
		}
		signals[i] = s.Evaluate(ctx)
		history = append(history, c)
	}
	return signals
}

// HasDecision reports whether any signal in the series matches target decision.
func HasDecision(signals []strategylib.Signal, target strategylib.Decision) bool {
	for _, sig := range signals {
		if sig.Decision == target {
			return true
		}
	}
	return false
}
