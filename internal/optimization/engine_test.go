package optimization_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/optimization"
)

func baseCfg() optimization.Config {
	return optimization.Config{
		Enabled:          true,
		SubscriberBuffer: 16,
		Scoring: optimization.ScoringConfig{
			ProfitFactorWeight: 0.40,
			WinRateWeight:      0.30,
			ExpectancyWeight:   0.20,
			DrawdownPenalty:    0.10,
		},
	}
}

func performancePayload(strategy string, totalTrades int, winRate, realized, unrealized, drawdown float64, at time.Time) []byte {
	payload, err := json.Marshal(map[string]any{
		"strategy":       strategy,
		"symbol":         "NIFTY",
		"timeframe":      "5m",
		"total_trades":   totalTrades,
		"win_rate":       winRate,
		"realized_pnl":   realized,
		"unrealized_pnl": unrealized,
		"drawdown":       drawdown,
		"timestamp":      at,
	})
	if err != nil {
		panic(err)
	}
	return payload
}

func TestProfitableStrategyHighScore(t *testing.T) {
	cache := optimization.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	result := cache.Apply(optimization.InputUpdate{
		Strategy:      "trend_following",
		Symbol:        "NIFTY",
		Timeframe:     "5m",
		TotalTrades:   10,
		WinRate:       0.80,
		RealizedPnL:   500,
		UnrealizedPnL: 50,
		Drawdown:      20,
		ProfitFactor:  3.5,
		Timestamp:     at,
	}, baseCfg().Scoring)

	require.Greater(t, result.Record.Score, 0.5)
	require.Equal(t, 1, result.Record.Rank)
}

func TestHighDrawdownLowerScore(t *testing.T) {
	cfg := baseCfg()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	profitable := optimization.NewCache()
	profitableResult := profitable.Apply(optimization.InputUpdate{
		Strategy:     "profitable",
		Symbol:       "NIFTY",
		Timeframe:    "5m",
		TotalTrades:  10,
		WinRate:      0.70,
		RealizedPnL:  300,
		Drawdown:     30,
		ProfitFactor: 2.5,
		Timestamp:    at,
	}, cfg.Scoring)

	highDD := optimization.NewCache()
	highDDResult := highDD.Apply(optimization.InputUpdate{
		Strategy:     "high_drawdown",
		Symbol:       "NIFTY",
		Timeframe:    "5m",
		TotalTrades:  10,
		WinRate:      0.70,
		RealizedPnL:  300,
		Drawdown:     800,
		MaxDrawdown:  800,
		ProfitFactor: 2.5,
		Timestamp:    at,
	}, cfg.Scoring)

	require.Greater(t, profitableResult.Record.Score, highDDResult.Record.Score)
}

func TestMultipleStrategiesCorrectRanking(t *testing.T) {
	cache := optimization.NewCache()
	cfg := baseCfg()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	best := cache.Apply(optimization.InputUpdate{
		Strategy: "alpha", Symbol: "NIFTY", Timeframe: "5m",
		TotalTrades: 20, WinRate: 0.75, RealizedPnL: 1000,
		ProfitFactor: 4.0, Drawdown: 50, Timestamp: at,
	}, cfg.Scoring)

	mid := cache.Apply(optimization.InputUpdate{
		Strategy: "beta", Symbol: "NIFTY", Timeframe: "5m",
		TotalTrades: 15, WinRate: 0.55, RealizedPnL: 200,
		ProfitFactor: 1.5, Drawdown: 100, Timestamp: at.Add(time.Minute),
	}, cfg.Scoring)

	worst := cache.Apply(optimization.InputUpdate{
		Strategy: "gamma", Symbol: "NIFTY", Timeframe: "5m",
		TotalTrades: 10, WinRate: 0.30, RealizedPnL: -100,
		ProfitFactor: 0.5, Drawdown: 300, Timestamp: at.Add(2 * time.Minute),
	}, cfg.Scoring)

	require.Equal(t, 1, best.Record.Rank)
	require.Equal(t, 2, mid.Record.Rank)
	require.Equal(t, 3, worst.Record.Rank)
	require.Greater(t, best.Record.Score, mid.Record.Score)
	require.Greater(t, mid.Record.Score, worst.Record.Score)
}

func TestEnginePublishesOptimizationUpdated(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	engine, err := optimization.New(baseCfg(), bus, clk)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	updates := make(chan optimization.OptimizationUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.OptimizationUpdated
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var update optimization.OptimizationUpdated
			if err := json.Unmarshal(evt.Payload, &update); err != nil {
				continue
			}
			updates <- update
		}
	}()

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	bus.Publish(events.Event{
		Type:      events.PerformanceUpdated,
		Source:    "performance_engine",
		Timestamp: at,
		Payload:   performancePayload("trend_following", 5, 0.60, 100, 0, 10, at),
	})

	select {
	case update := <-updates:
		require.Equal(t, "trend_following", update.Strategy)
		require.Equal(t, "NIFTY", update.Symbol)
		require.Equal(t, 5, update.Metrics.TotalTrades)
		require.Greater(t, update.Score, 0.0)
		require.Equal(t, 1, update.Rank)
	case <-time.After(time.Second):
		t.Fatal("expected optimization.updated event")
	}

	report := engine.Health()
	require.Equal(t, "optimization_engine", report.Component)
	require.Equal(t, "1", report.Details["evaluations_processed"])

	require.NoError(t, engine.Close())
}
