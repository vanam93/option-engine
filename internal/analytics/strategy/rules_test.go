package strategy_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/strategy"
)

func baseCfg() strategy.Config {
	return strategy.Config{
		Enabled: true,
		TrendFollowing: strategy.TrendFollowingConfig{
			Enabled: true,
		},
		MeanReversion: strategy.MeanReversionConfig{
			Enabled: true,
		},
	}
}

func inputSignal(signalStrategy, signalType string, confidence float64, at time.Time) strategy.InputSignal {
	return strategy.InputSignal{
		Symbol:     "NIFTY",
		Timeframe:  "1m",
		Signal:     signalType,
		Strategy:   signalStrategy,
		Confidence: confidence,
		Timestamp:  at,
	}
}

func TestTrendFollowingLongEntry(t *testing.T) {
	cfg := baseCfg()
	eval := strategy.NewEvaluator(cfg)
	cache := strategy.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_ = eval.Process(inputSignal("ema_cross", "BUY", 0.75, at), cache)
	decisions := eval.Process(inputSignal("macd_cross", "BUY", 0.75, at.Add(time.Minute)), cache)

	require.NotEmpty(t, decisions)
	var trend *strategy.StrategyDecision
	for _, d := range decisions {
		if d.Strategy == "trend_following" {
			trend = &d
			break
		}
	}
	require.NotNil(t, trend)
	require.Equal(t, strategy.LongEntry, trend.Decision)
}

func TestMeanReversionLongEntry(t *testing.T) {
	cfg := baseCfg()
	eval := strategy.NewEvaluator(cfg)
	cache := strategy.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_ = eval.Process(inputSignal("rsi", "BUY", 0.70, at), cache)
	decisions := eval.Process(inputSignal("bollinger", "BUY", 0.70, at.Add(time.Minute)), cache)

	require.NotEmpty(t, decisions)
	var meanRev *strategy.StrategyDecision
	for _, d := range decisions {
		if d.Strategy == "mean_reversion" {
			meanRev = &d
			break
		}
	}
	require.NotNil(t, meanRev)
	require.Equal(t, strategy.LongEntry, meanRev.Decision)
}

func TestHoldOnConflictingSignals(t *testing.T) {
	cfg := baseCfg()
	eval := strategy.NewEvaluator(cfg)
	cache := strategy.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_ = eval.Process(inputSignal("ema_cross", "BUY", 0.75, at), cache)
	decisions := eval.Process(inputSignal("macd_cross", "SELL", 0.75, at.Add(time.Minute)), cache)

	require.NotEmpty(t, decisions)
	var trend *strategy.StrategyDecision
	for _, d := range decisions {
		if d.Strategy == "trend_following" {
			trend = &d
			break
		}
	}
	require.NotNil(t, trend)
	require.Equal(t, strategy.Hold, trend.Decision)
}
