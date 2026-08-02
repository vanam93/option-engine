package indicator_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator"
	domainindicator "github.com/vanam-gangireddy/option-engine/internal/domain/indicator"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func TestEngineWarmUpPublishesAfterPeriod(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	engine, err := indicator.New(indicator.Config{
		Enabled:          true,
		SubscriberBuffer: 8,
		EMA:              []indicator.PeriodConfig{{Period: 2}},
		SMA:              []indicator.PeriodConfig{{Period: 2}},
	}, bus, nil)
	require.NoError(t, err)

	indicatorSub := bus.Subscribe(16, func(e events.Event) bool {
		return e.Type == events.IndicatorUpdated
	})
	defer indicatorSub.Close()

	require.NoError(t, engine.Start(context.Background()))
	defer func() { _ = engine.Close() }()

	ts := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	for i, close := range []float64{10, 20, 30} {
		candle := market.Candle{
			ID: uuid.New(), Symbol: "NIFTY", Timeframe: market.TF1m,
			Close: close, OpenTime: ts.Add(time.Duration(i) * time.Minute),
			CloseTime: ts.Add(time.Duration(i+1) * time.Minute),
		}
		evt, err := events.NewEventWithTime(events.CandleClosed, "candle_engine", candle, candle.CloseTime)
		require.NoError(t, err)
		bus.Publish(evt)
	}

	received := 0
	deadline := time.After(2 * time.Second)
	for received < 4 {
		select {
		case evt := <-indicatorSub.C:
			var value domainindicator.IndicatorValue
			require.NoError(t, json.Unmarshal(evt.Payload, &value))
			require.Equal(t, "NIFTY", value.Symbol)
			require.Equal(t, float64(1), value.Values["warmed_up"])
			received++
		case <-deadline:
			t.Fatalf("expected 4 warmed indicator events, got %d", received)
		}
	}
}
