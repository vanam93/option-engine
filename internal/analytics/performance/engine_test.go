package performance_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/performance"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func baseCfg() performance.Config {
	return performance.Config{
		Enabled:          true,
		SubscriberBuffer: 16,
	}
}

func portfolioUpdate(symbol string, open bool, realized, unrealized float64, at time.Time) performance.InputUpdate {
	return performance.InputUpdate{
		Symbol:        symbol,
		PositionOpen:  open,
		RealizedPnL:   realized,
		UnrealizedPnL: unrealized,
		Timestamp:     at,
	}
}

func TestWinningTradePositivePnL(t *testing.T) {
	cache := performance.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	cache.Apply(portfolioUpdate("NIFTY", true, 0, 0, at))
	exitAt := at.Add(time.Minute)
	result := cache.Apply(portfolioUpdate("NIFTY", false, 10, 0, exitAt))

	require.Equal(t, 1, result.Snapshot.TotalTrades)
	require.Equal(t, 1, result.Snapshot.WinningTrades)
	require.InDelta(t, 10, result.Snapshot.Trades[0].PnL, 0.0001)
	require.InDelta(t, 10, result.Snapshot.RealizedPnL, 0.0001)
	require.InDelta(t, 1.0, result.Snapshot.WinRate, 0.0001)
}

func TestLosingTradeNegativePnL(t *testing.T) {
	cache := performance.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	cache.Apply(portfolioUpdate("NIFTY", true, 0, 0, at))
	exitAt := at.Add(time.Minute)
	result := cache.Apply(portfolioUpdate("NIFTY", false, -5, 0, exitAt))

	require.Equal(t, 1, result.Snapshot.TotalTrades)
	require.Equal(t, 1, result.Snapshot.LosingTrades)
	require.InDelta(t, -5, result.Snapshot.Trades[0].PnL, 0.0001)
	require.InDelta(t, 0, result.Snapshot.WinRate, 0.0001)
}

func TestDrawdownCalculation(t *testing.T) {
	cache := performance.NewCache()
	base := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	cache.Apply(portfolioUpdate("NIFTY", true, 0, 0, base))
	cache.Apply(portfolioUpdate("NIFTY", false, 10, 0, base.Add(time.Minute)))

	cache.Apply(portfolioUpdate("BANKNIFTY", true, 10, 0, base.Add(2*time.Minute)))
	cache.Apply(portfolioUpdate("BANKNIFTY", false, 5, 0, base.Add(3*time.Minute)))

	cache.Apply(portfolioUpdate("RELIANCE", true, 5, 0, base.Add(4*time.Minute)))
	result := cache.Apply(portfolioUpdate("RELIANCE", false, 15, 0, base.Add(5*time.Minute)))

	require.InDelta(t, 5, result.Snapshot.MaxDrawdown, 0.0001)
	require.InDelta(t, 0, result.Snapshot.CurrentDrawdown, 0.0001)
}

func TestMaxDrawdownMetric(t *testing.T) {
	require.InDelta(t, 5, performance.MaxDrawdown([]float64{0, 10, 5, 15}), 0.0001)
}

func TestEnginePublishesPerformanceUpdated(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	engine, err := performance.New(baseCfg(), bus, clk)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	updates := make(chan performance.PerformanceUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.PerformanceUpdated
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var update performance.PerformanceUpdated
			if err := json.Unmarshal(evt.Payload, &update); err != nil {
				continue
			}
			updates <- update
		}
	}()

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"symbol":         "NIFTY",
		"position":       map[string]any{"symbol": "NIFTY"},
		"realized_pnl":   0,
		"unrealized_pnl": 0,
		"timestamp":      at,
	})
	require.NoError(t, err)
	bus.Publish(events.Event{
		Type:      events.PortfolioUpdated,
		Source:    "portfolio_engine",
		Timestamp: at,
		Payload:   payload,
	})

	select {
	case update := <-updates:
		require.Equal(t, 0, update.TotalTrades)
		require.InDelta(t, 0, update.RealizedPnL, 0.0001)
	case <-time.After(time.Second):
		t.Fatal("expected performance.updated event")
	}

	require.NoError(t, engine.Close())
}
