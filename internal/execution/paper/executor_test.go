package paper_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/execution/paper"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
)

func baseCfg() paper.Config {
	return paper.Config{
		Enabled:         true,
		SlippagePercent: 0.05,
		DefaultPrice:    "100",
	}
}

func approvedIntent(action string, refPrice float64, at time.Time) paper.InputIntent {
	return paper.InputIntent{
		ID:             uuid.New(),
		Symbol:         "NIFTY",
		Timeframe:      "1m",
		Status:         "APPROVED",
		Action:         action,
		Quantity:       1,
		Strategy:       "trend_following",
		ReferencePrice: refPrice,
		Timestamp:      at,
	}
}

func TestApprovedLongIntentFilled(t *testing.T) {
	exec := paper.NewExecutor(baseCfg())
	cache := paper.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	report := exec.Execute(approvedIntent("LONG_ENTRY", 100, at), cache)

	require.Equal(t, paper.Filled, report.Status)
	require.Equal(t, "LONG_ENTRY", report.Action)
	require.Equal(t, "NIFTY", report.Symbol)
	require.Equal(t, 1, report.Quantity)
	require.InDelta(t, 100.05, report.ExecutionPrice, 0.0001)
	require.NotEmpty(t, report.OrderID)
}

func TestRejectedIntentIgnored(t *testing.T) {
	bus := eventbus.New()
	clk := clock.NewSystem()
	engine, err := paper.New(baseCfg(), bus, clk)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, engine.Start(ctx))

	reports := make(chan paper.ExecutionReport, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ExecutionReport
	})
	defer sub.Close()

	go func() {
		for evt := range sub.C {
			var report paper.ExecutionReport
			if err := json.Unmarshal(evt.Payload, &report); err != nil {
				continue
			}
			reports <- report
		}
	}()

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"id":        uuid.New(),
		"symbol":    "NIFTY",
		"timeframe": "1m",
		"status":    "REJECTED",
		"action":    "LONG_ENTRY",
		"quantity":  1,
		"strategy":  "trend_following",
		"timestamp": at,
	})
	require.NoError(t, err)
	bus.Publish(events.Event{
		Type:      events.ApprovedTradeIntent,
		Source:    "risk_engine",
		Timestamp: at,
		Payload:   payload,
	})

	select {
	case <-reports:
		t.Fatal("expected rejected intent to be ignored")
	case <-time.After(100 * time.Millisecond):
	}

	health := engine.Health()
	require.Equal(t, "0", health.Details["orders_received"])
	require.Equal(t, "0", health.Details["orders_filled"])

	require.NoError(t, engine.Close())
}

func TestPositionUpdateCreatesOpenPosition(t *testing.T) {
	exec := paper.NewExecutor(baseCfg())
	cache := paper.NewCache()
	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)

	report := exec.Execute(approvedIntent("LONG_ENTRY", 100, at), cache)
	require.Equal(t, paper.Filled, report.Status)
	require.Equal(t, 1, cache.ActivePositions())
}
