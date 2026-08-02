package candle_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/candle"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time                  { return c.now }
func (c fixedClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }
func (c fixedClock) Until(t time.Time) time.Duration { return t.Sub(c.now) }

func baseConfig() candle.Config {
	return candle.Config{
		Enabled:          true,
		Timeframes:       []market.Timeframe{market.TF1m},
		Timezone:         "UTC",
		SubscriberBuffer: 8,
		FlushOnShutdown:  true,
		VolumeMode:       candle.VolumeCumulative,
		OrderPolicy:      candle.OrderRejectOlder,
	}
}

func TestEnginePublishesCandleClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	bus := eventbus.New()
	defer bus.Close()

	engine, err := candle.New(baseConfig(), bus, fixedClock{now: now})
	require.NoError(t, err)

	candleSub := bus.Subscribe(8, func(e events.Event) bool {
		return e.Type == events.CandleClosed
	})
	defer candleSub.Close()

	require.NoError(t, engine.Start(ctx))
	defer func() { _ = engine.Close() }()

	tick1, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 100, ProviderTS: now,
	}, now)
	require.NoError(t, err)
	bus.Publish(tick1)

	tick2, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 110, ProviderTS: now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	bus.Publish(tick2)

	select {
	case evt := <-candleSub.C:
		require.Equal(t, events.CandleClosed, evt.Type)
		var closed market.Candle
		require.NoError(t, json.Unmarshal(evt.Payload, &closed))
		require.Equal(t, "NIFTY", closed.Symbol)
		require.Equal(t, market.TF1m, closed.Timeframe)
		require.Equal(t, 100.0, closed.Open)
		require.Equal(t, 100.0, closed.Close)
	case <-ctx.Done():
		t.Fatal("timed out waiting for candle closed event")
	}

	report := engine.Health()
	require.Equal(t, "candle_engine", report.Component)
	require.True(t, report.Connected)
}

func TestEngineDisabledIsNoOp(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	engine, err := candle.New(candle.Config{Enabled: false}, bus, clock.NewSystem())
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))
	require.NoError(t, engine.Close())
}

func TestEngineCloseStopsGoroutine(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	engine, err := candle.New(baseConfig(), bus, clock.NewSystem())
	require.NoError(t, err)

	require.NoError(t, engine.Start(context.Background()))
	require.NoError(t, engine.Close())
	require.NoError(t, engine.Close())
}

func TestEngineShutdownFlushPublishesInProgressBar(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := candle.New(baseConfig(), bus, fixedClock{now: now})
	require.NoError(t, err)

	candleSub := bus.Subscribe(4, func(e events.Event) bool {
		return e.Type == events.CandleClosed
	})
	defer candleSub.Close()

	require.NoError(t, engine.Start(context.Background()))

	tick, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 100, Volume: 10, ProviderTS: now,
	}, now)
	require.NoError(t, err)
	bus.Publish(tick)

	require.NoError(t, engine.Close())

	select {
	case evt := <-candleSub.C:
		var closed market.Candle
		require.NoError(t, json.Unmarshal(evt.Payload, &closed))
		require.Equal(t, 100.0, closed.Close)
		require.Equal(t, int64(10), closed.Volume)
	default:
		t.Fatal("expected flush candle on shutdown")
	}
}

func TestEngineStartupSubscriptionBeforePublish(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	engine, err := candle.New(baseConfig(), bus, fixedClock{now: now})
	require.NoError(t, err)

	candleSub := bus.Subscribe(4, func(e events.Event) bool {
		return e.Type == events.CandleClosed
	})
	defer candleSub.Close()

	require.NoError(t, engine.Start(context.Background()))

	first, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 100, ProviderTS: now,
	}, now)
	require.NoError(t, err)
	bus.Publish(first)

	second, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 110, ProviderTS: now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	bus.Publish(second)

	select {
	case evt := <-candleSub.C:
		var closed market.Candle
		require.NoError(t, json.Unmarshal(evt.Payload, &closed))
		require.Equal(t, 100.0, closed.Open)
	case <-time.After(time.Second):
		t.Fatal("first session candle must not be lost after startup subscribe")
	}

	require.NoError(t, engine.Close())
}

func TestEngineRejectsOutOfOrderWithoutCorruption(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	now := time.Date(2026, 8, 2, 10, 5, 0, 0, time.UTC)
	engine, err := candle.New(baseConfig(), bus, fixedClock{now: now})
	require.NoError(t, err)

	require.NoError(t, engine.Start(context.Background()))
	defer func() { _ = engine.Close() }()

	candleSub := bus.Subscribe(4, func(e events.Event) bool {
		return e.Type == events.CandleClosed
	})
	defer candleSub.Close()

	inOrder, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 100, ProviderTS: now,
	}, now)
	require.NoError(t, err)
	bus.Publish(inOrder)

	late, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 50, ProviderTS: now.Add(-5 * time.Minute),
	}, now.Add(-5*time.Minute))
	require.NoError(t, err)
	bus.Publish(late)

	rollover, err := events.NewEventWithTime(events.MarketDataReceived, "test", market.Tick{
		Symbol: "NIFTY", LTP: 110, ProviderTS: now.Add(time.Minute),
	}, now.Add(time.Minute))
	require.NoError(t, err)
	bus.Publish(rollover)

	select {
	case evt := <-candleSub.C:
		var closed market.Candle
		require.NoError(t, json.Unmarshal(evt.Payload, &closed))
		require.Equal(t, 100.0, closed.Close)
	case <-time.After(time.Second):
		t.Fatal("expected closed candle")
	}

	report := engine.Health()
	require.Equal(t, "1", report.Details["rejected"])
}

func TestEngineConcurrentCloseNoLeak(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	engine, err := candle.New(baseConfig(), bus, clock.NewSystem())
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = engine.Close()
		}()
	}
	wg.Wait()
}
