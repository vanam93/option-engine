package signal_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/signal"
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
)

func indicatorValue(name domainindicator.Name, symbol, timeframe string, at time.Time, values map[string]float64) domainindicator.IndicatorValue {
	vals := map[string]float64{"warmed_up": 1}
	for k, v := range values {
		vals[k] = v
	}
	return domainindicator.IndicatorValue{
		ID:         uuid.New(),
		Name:       name,
		Symbol:     symbol,
		Timeframe:  timeframe,
		Values:     vals,
		ComputedAt: at,
	}
}

func TestEMACrossoverBuy(t *testing.T) {
	cfg := signal.Config{
		Enabled: true,
		EMACross: signal.EMACrossConfig{
			Enabled:    true,
			FastPeriod: 9,
			SlowPeriod: 21,
		},
	}
	eval := signal.NewEvaluator(cfg)
	cache := signal.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_ = eval.Evaluate(indicatorValue(domainindicator.EMA, "NIFTY", "1m", at, map[string]float64{
		"period": 9,
		"value":  98,
	}), cache)
	_ = eval.Evaluate(indicatorValue(domainindicator.EMA, "NIFTY", "1m", at, map[string]float64{
		"period": 21,
		"value":  100,
	}), cache)

	signals := eval.Evaluate(indicatorValue(domainindicator.EMA, "NIFTY", "1m", at.Add(time.Minute), map[string]float64{
		"period": 9,
		"value":  102,
	}), cache)
	require.Len(t, signals, 1)
	require.Equal(t, signal.Buy, signals[0].Signal)
	require.Equal(t, "ema_cross", signals[0].Strategy)
}

func TestMACDCrossoverSell(t *testing.T) {
	cfg := signal.Config{
		Enabled:   true,
		MACDCross: signal.MACDCrossConfig{Enabled: true},
	}
	eval := signal.NewEvaluator(cfg)
	cache := signal.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_ = eval.Evaluate(indicatorValue(domainindicator.MACD, "NIFTY", "1m", at, map[string]float64{
		"macd": 1.2, "signal": 1.0,
	}), cache)

	signals := eval.Evaluate(indicatorValue(domainindicator.MACD, "NIFTY", "1m", at.Add(time.Minute), map[string]float64{
		"macd": 0.8, "signal": 1.0,
	}), cache)
	require.Len(t, signals, 1)
	require.Equal(t, signal.Sell, signals[0].Signal)
	require.Equal(t, "macd_cross", signals[0].Strategy)
}

func TestRSIThresholdBuy(t *testing.T) {
	cfg := signal.Config{
		Enabled: true,
		RSI: signal.RSIConfig{
			Enabled:    true,
			Oversold:   30,
			Overbought: 70,
		},
	}
	eval := signal.NewEvaluator(cfg)
	cache := signal.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	signals := eval.Evaluate(indicatorValue(domainindicator.RSI, "NIFTY", "1m", at, map[string]float64{
		"value": 25,
	}), cache)
	require.Len(t, signals, 1)
	require.Equal(t, signal.Buy, signals[0].Signal)
	require.Equal(t, "rsi", signals[0].Strategy)
}

func TestBollingerThresholdSell(t *testing.T) {
	cfg := signal.Config{
		Enabled:   true,
		Bollinger: signal.BollingerConfig{Enabled: true},
	}
	eval := signal.NewEvaluator(cfg)
	cache := signal.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	signals := eval.Evaluate(indicatorValue(domainindicator.BollingerBands, "NIFTY", "1m", at, map[string]float64{
		"upper": 100, "middle": 95, "lower": 90, "percent_b": 1.2,
	}), cache)
	require.Len(t, signals, 1)
	require.Equal(t, signal.Sell, signals[0].Signal)
	require.Equal(t, "bollinger", signals[0].Strategy)
}
