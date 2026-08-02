package market_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/cache"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/market/gateway"
	"github.com/vanam-gangireddy/option-engine/internal/market/normalizer"
	"github.com/vanam-gangireddy/option-engine/internal/market/registry"
	"github.com/vanam-gangireddy/option-engine/internal/market/snapshot"
	"github.com/vanam-gangireddy/option-engine/internal/market/subscription"
	"github.com/vanam-gangireddy/option-engine/internal/market/validator"
	"github.com/vanam-gangireddy/option-engine/internal/providers"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time                  { return c.now }
func (c fixedClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }
func (c fixedClock) Until(t time.Time) time.Duration { return t.Sub(c.now) }

func TestRuntimeIntegrationPipeline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	clk := fixedClock{now: now}
	reg := providers.DefaultRegistry()
	manager := providers.NewManager(reg, providers.ManagerConfig{ActiveProvider: "mock", ProviderCfg: map[string]any{"tick_interval": "10ms"}})
	require.NoError(t, manager.InitWithDeps(providers.FactoryConfig{ProviderCfg: map[string]any{"tick_interval": "10ms"}, Deps: providers.Dependencies{Clock: clk}}))
	provider, err := manager.Provider()
	require.NoError(t, err)
	require.NoError(t, manager.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	cacheStore := cache.New()
	bus := eventbus.New()
	regSymbols := registry.New()
	require.NoError(t, regSymbols.Load([]registry.Instrument{{Symbol: "NIFTY", Token: "1", Exchange: "NSE", InstrumentType: market.InstrumentIndex, LotSize: 25}}))
	val := validator.New(validator.Config{MaxAge: 2 * time.Minute, RequireRegisteredSymbol: true}, regSymbols)
	norm := normalizer.New(func() time.Time { return now })
	engine := gateway.New(manager.Session(), cacheStore, bus, val, norm, func() time.Time { return now })

	sub := bus.Subscribe(16, nil)
	defer sub.Close()

	require.NoError(t, engine.Start(ctx))

	var received events.Event
	select {
	case received = <-sub.C:
	case <-ctx.Done():
		t.Fatal("timed out waiting for canonical event")
	}

	var tick market.Tick
	require.NoError(t, json.Unmarshal(received.Payload, &tick))
	require.Equal(t, "NIFTY", tick.Symbol)

	cachedTick, ok := cacheStore.Tick("NIFTY")
	require.True(t, ok)
	require.Equal(t, "NIFTY", cachedTick.Symbol)

	snap := snapshot.New(cacheStore, now.Add(5*time.Second))
	require.Equal(t, "NIFTY", snap.Ticks["NIFTY"].Symbol)

	// Snapshot must be immutable from the cache's perspective.
	snap.Ticks["NIFTY"] = market.Tick{Symbol: "CHANGED"}
	refreshed, ok := cacheStore.Tick("NIFTY")
	require.True(t, ok)
	require.Equal(t, "NIFTY", refreshed.Symbol)

	require.NoError(t, engine.Close())
	require.NoError(t, manager.Disconnect(context.Background()))
}

func TestReconnectRestoresSubscriptionsAndTickFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	clk := fixedClock{now: now}
	reg := providers.DefaultRegistry()
	manager := providers.NewManager(reg, providers.ManagerConfig{
		ActiveProvider: "mock",
		ProviderCfg: map[string]any{
			"tick_interval": "10ms",
		},
		Subscription: providers.SubscriptionConfig{BatchSize: 1},
	})
	require.NoError(t, manager.InitWithDeps(providers.FactoryConfig{
		ProviderCfg: map[string]any{
			"tick_interval": "10ms",
		},
		Deps: providers.Dependencies{Clock: clk},
	}))

	provider, err := manager.Provider()
	require.NoError(t, err)

	cacheStore := cache.New()
	bus := eventbus.New()
	regSymbols := registry.New()
	require.NoError(t, regSymbols.Load([]registry.Instrument{{Symbol: "NIFTY", Token: "1", Exchange: "NSE", InstrumentType: market.InstrumentIndex, LotSize: 25}}))
	val := validator.New(validator.Config{MaxAge: 2 * time.Minute, RequireRegisteredSymbol: true}, regSymbols)
	norm := normalizer.New(func() time.Time { return now })
	engine := gateway.New(manager.Session(), cacheStore, bus, val, norm, func() time.Time { return now })

	subManager := subscription.New(provider, 1)
	manager.SetSubscriptionManager(subManager)
	sub := bus.Subscribe(16, nil)
	defer sub.Close()

	require.NoError(t, manager.Connect(ctx))
	require.NoError(t, subManager.Subscribe(ctx, []string{"NIFTY"}))
	require.NoError(t, engine.Start(ctx))

	select {
	case <-sub.C:
	case <-ctx.Done():
		t.Fatal("timed out waiting for initial tick")
	}

	require.NoError(t, manager.Disconnect(ctx))
	require.NoError(t, manager.Connect(ctx))

	select {
	case <-sub.C:
	case <-ctx.Done():
		t.Fatal("timed out waiting for tick after reconnect")
	}

	cachedTick, ok := cacheStore.Tick("NIFTY")
	require.True(t, ok)
	require.Equal(t, "NIFTY", cachedTick.Symbol)

	require.NoError(t, engine.Close())
}
