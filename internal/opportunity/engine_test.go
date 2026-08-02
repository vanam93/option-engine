package opportunity_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/opportunity"
)

func TestEnginePublishesOpportunityUpdated(t *testing.T) {
	bus := eventbus.New()
	engine, err := opportunity.New(testConfig(), bus, clock.NewSystem())
	require.NoError(t, err)

	updates := make(chan opportunity.OpportunityUpdated, 2)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.OpportunityUpdated
	})
	go func() {
		for evt := range sub.C {
			var update opportunity.OpportunityUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	signalEvt, err := events.NewEventWithTime(events.SignalGenerated, "signal", map[string]any{
		"symbol": "NIFTY", "timeframe": "1m", "confidence": 0.9, "timestamp": at,
	}, at)
	require.NoError(t, err)
	bus.Publish(signalEvt)

	strategyEvt, err := events.NewEventWithTime(events.StrategyDecision, "strategy", map[string]any{
		"symbol": "NIFTY", "timeframe": "1m", "confidence": 0.85, "timestamp": at,
	}, at)
	require.NoError(t, err)
	bus.Publish(strategyEvt)

	riskEvt, err := events.NewEventWithTime(events.ApprovedTradeIntent, "risk", map[string]any{
		"symbol": "NIFTY", "timeframe": "1m", "status": "APPROVED", "timestamp": at,
	}, at)
	require.NoError(t, err)
	bus.Publish(riskEvt)

	scannerEvt, err := events.NewEventWithTime(events.ScannerUpdated, "scanner", map[string]any{
		"symbol": "NIFTY", "timeframe": "1m", "scanner_name": "ema",
		"status": "MATCH", "score": 0.9, "confidence": 0.9, "timestamp": at,
	}, at)
	require.NoError(t, err)
	bus.Publish(scannerEvt)

	select {
	case update := <-updates:
		require.Equal(t, "NIFTY", update.Symbol)
		require.Equal(t, 1, update.Rank)
		require.NotEmpty(t, update.Classification)
		require.Greater(t, update.Confidence, 0.0)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for opportunity.updated")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}
