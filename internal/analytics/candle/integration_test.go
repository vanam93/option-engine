package candle_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/candle"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/market/gateway"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	"github.com/vanam-gangireddy/option-engine/internal/market/registry"
	"github.com/vanam-gangireddy/option-engine/internal/market/validator"
	"github.com/vanam-gangireddy/option-engine/internal/providers"
)

func TestCandleEngineConsumesGatewayCanonicalEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	clk := fixedClock{now: now}

	reg := providers.DefaultRegistry()
	manager := providers.NewManager(reg, providers.ManagerConfig{
		ActiveProvider: "mock",
		ProviderCfg:    map[string]any{"tick_interval": "20ms", "seed": 7},
	})
	require.NoError(t, manager.InitWithDeps(providers.FactoryConfig{
		ProviderCfg: map[string]any{"tick_interval": "20ms", "seed": 7},
		Deps:        providers.Dependencies{Clock: clk},
	}))
	require.NoError(t, manager.Connect(ctx))
	provider, err := manager.Provider()
	require.NoError(t, err)
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	bus := eventbus.New()
	cacheStore := cache.New()
	symbols := registry.New()
	require.NoError(t, symbols.Load([]registry.Instrument{{
		Symbol: "NIFTY", Token: "1", Exchange: "NSE", InstrumentType: market.InstrumentIndex, LotSize: 25,
	}}))
	val := validator.New(validator.Config{MaxAge: time.Minute, RequireRegisteredSymbol: true}, symbols)
	norm := normalizer.New(func() time.Time { return now })
	gw := gateway.New(manager.Session(), cacheStore, bus, val, norm, func() time.Time { return now })

	engine, err := candle.New(candle.Config{
		Enabled:          true,
		Timeframes:       []market.Timeframe{market.TF1m},
		Timezone:         "UTC",
		SubscriberBuffer: 16,
		FlushOnShutdown:  true,
	}, bus, clk)
	require.NoError(t, err)

	tickSub := bus.Subscribe(16, func(e events.Event) bool {
		return e.Type == events.MarketDataReceived
	})
	defer tickSub.Close()
	candleSub := bus.Subscribe(16, func(e events.Event) bool {
		return e.Type == events.CandleClosed
	})
	defer candleSub.Close()

	require.NoError(t, engine.Start(ctx))
	require.NoError(t, gw.Start(ctx))

	select {
	case evt := <-tickSub.C:
		require.Equal(t, events.MarketDataReceived, evt.Type)
	case <-ctx.Done():
		t.Fatal("timed out waiting for gateway canonical tick")
	}

	rollover, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 110, ProviderTS: now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	bus.Publish(rollover)

	select {
	case evt := <-candleSub.C:
		var closed market.Candle
		require.NoError(t, json.Unmarshal(evt.Payload, &closed))
		require.Equal(t, "NIFTY", closed.Symbol)
		require.Equal(t, market.TF1m, closed.Timeframe)
	case <-ctx.Done():
		t.Fatal("timed out waiting for candle from gateway pipeline")
	}

	require.NoError(t, engine.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, manager.Disconnect(context.Background()))
}
