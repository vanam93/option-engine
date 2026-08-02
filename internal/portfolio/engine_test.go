package portfolio_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/portfolio"
)

func baseCfg() portfolio.Config {
	return portfolio.Config{
		Enabled:          true,
		SubscriberBuffer: 16,
	}
}

func filledReport(action string, symbol string, price float64, at time.Time) portfolio.InputReport {
	return portfolio.InputReport{
		OrderID:        "1",
		Symbol:         symbol,
		Timeframe:      "1m",
		Action:         action,
		Quantity:       1,
		ExecutionPrice: price,
		Status:         "FILLED",
		Strategy:       "trend_following",
		Timestamp:      at,
	}
}

func TestLongFilledExecutionCreatesPosition(t *testing.T) {
	cache := portfolio.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	result, ok := cache.Apply(filledReport("LONG_ENTRY", "NIFTY", 100, at))
	require.True(t, ok)
	require.NotNil(t, result.Updated.Position)
	require.Equal(t, portfolio.SideLong, result.Updated.Position.Side)
	require.Equal(t, 1, result.Updated.Position.Quantity)
	require.InDelta(t, 100, result.Updated.Position.AveragePrice, 0.0001)

	state := cache.Snapshot()
	require.Len(t, state.Positions, 1)
	require.Equal(t, "NIFTY", state.Positions[0].Symbol)
}

func TestLongEntryAndExitCalculatesRealizedPnL(t *testing.T) {
	cache := portfolio.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_, ok := cache.Apply(filledReport("LONG_ENTRY", "NIFTY", 100, at))
	require.True(t, ok)

	exitAt := at.Add(time.Minute)
	result, ok := cache.Apply(filledReport("LONG_EXIT", "NIFTY", 110, exitAt))
	require.True(t, ok)
	require.Nil(t, result.Updated.Position)
	require.InDelta(t, 10, result.Updated.RealizedPnL, 0.0001)

	state := cache.Snapshot()
	require.Empty(t, state.Positions)
	require.InDelta(t, 10, state.RealizedPnL, 0.0001)
	require.InDelta(t, 0, state.UnrealizedPnL, 0.0001)
}

func TestMultiplePositionsTrackedIndependently(t *testing.T) {
	cache := portfolio.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	_, ok := cache.Apply(filledReport("LONG_ENTRY", "NIFTY", 100, at))
	require.True(t, ok)
	_, ok = cache.Apply(filledReport("SHORT_ENTRY", "BANKNIFTY", 200, at))
	require.True(t, ok)

	state := cache.Snapshot()
	require.Len(t, state.Positions, 2)

	nifty, bank := state.Positions[0], state.Positions[1]
	if nifty.Symbol != "NIFTY" {
		nifty, bank = bank, nifty
	}
	require.Equal(t, portfolio.SideLong, nifty.Side)
	require.Equal(t, portfolio.SideShort, bank.Side)
	require.InDelta(t, 100, nifty.AveragePrice, 0.0001)
	require.InDelta(t, 200, bank.AveragePrice, 0.0001)
}

func TestRejectedExecutionIgnored(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	engine, err := portfolio.New(baseCfg(), bus, clk)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	updates := make(chan portfolio.PortfolioUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.PortfolioUpdated
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var update portfolio.PortfolioUpdated
			if err := json.Unmarshal(evt.Payload, &update); err != nil {
				continue
			}
			updates <- update
		}
	}()

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"order_id":        "1",
		"symbol":          "NIFTY",
		"timeframe":       "1m",
		"action":          "LONG_ENTRY",
		"quantity":        1,
		"execution_price": 100,
		"status":          "REJECTED",
		"strategy":        "trend_following",
		"timestamp":       at,
	})
	require.NoError(t, err)
	bus.Publish(events.Event{
		Type:      events.ExecutionReport,
		Source:    "paper_execution_engine",
		Timestamp: at,
		Payload:   payload,
	})

	select {
	case <-updates:
		t.Fatal("expected rejected execution to be ignored")
	case <-time.After(100 * time.Millisecond):
	}

	state := engine.State()
	require.Empty(t, state.Positions)
	require.Equal(t, "0", engine.Health().Details["trades_processed"])

	require.NoError(t, engine.Close())
}

func TestEnginePublishesPortfolioUpdated(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	engine, err := portfolio.New(baseCfg(), bus, clk)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	updates := make(chan portfolio.PortfolioUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.PortfolioUpdated
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var update portfolio.PortfolioUpdated
			if err := json.Unmarshal(evt.Payload, &update); err != nil {
				continue
			}
			updates <- update
		}
	}()

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"order_id":        "1",
		"symbol":          "NIFTY",
		"timeframe":       "1m",
		"action":          "LONG_ENTRY",
		"quantity":        1,
		"execution_price": 100,
		"status":          "FILLED",
		"strategy":        "trend_following",
		"timestamp":       at,
	})
	require.NoError(t, err)
	bus.Publish(events.Event{
		Type:      events.ExecutionReport,
		Source:    "paper_execution_engine",
		Timestamp: at,
		Payload:   payload,
	})

	select {
	case update := <-updates:
		require.Equal(t, "NIFTY", update.Symbol)
		require.NotNil(t, update.Position)
		require.Equal(t, portfolio.SideLong, update.Position.Side)
	case <-time.After(time.Second):
		t.Fatal("expected portfolio.updated event")
	}

	require.NoError(t, engine.Close())
}
