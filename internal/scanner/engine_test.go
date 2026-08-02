package scanner_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/core/health"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/market/eventbus"
	"github.com/vanam-gangireddy/option-engine/internal/scanner"
)

func baseCfg() scanner.Config {
	return scanner.Config{
		Enabled:          true,
		Symbols:          []string{"NIFTY"},
		SubscriberBuffer: 16,
		MinConfidence:    0.5,
		Scanners: scanner.ScannersConfig{
			EMA:     true,
			RSI:     true,
			MACD:    true,
			Trend:   true,
			Ranking: true,
		},
	}
}

func TestEMAScannerPublishesMatch(t *testing.T) {
	bus := eventbus.New()
	engine, err := scanner.New(baseCfg(), bus, clock.NewSystem())
	require.NoError(t, err)

	updates := make(chan scanner.ScannerUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ScannerUpdated
	})
	go func() {
		for evt := range sub.C {
			var update scanner.ScannerUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	evt, err := events.NewEventWithTime(events.SignalGenerated, "signal", map[string]any{
		"symbol":     "NIFTY",
		"timeframe":  "1m",
		"signal":     "BUY",
		"strategy":   "ema_cross",
		"confidence": 0.8,
		"timestamp":  at,
	}, at)
	require.NoError(t, err)
	bus.Publish(evt)

	select {
	case update := <-updates:
		require.Equal(t, "ema", update.ScannerName)
		require.Equal(t, scanner.StatusMatch, update.Status)
		require.Equal(t, "NIFTY", update.Symbol)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for scanner.updated")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestRSIScannerPublishesWatchOnLowConfidence(t *testing.T) {
	bus := eventbus.New()
	cfg := baseCfg()
	cfg.Scanners = scanner.ScannersConfig{RSI: true}
	engine, err := scanner.New(cfg, bus, clock.NewSystem())
	require.NoError(t, err)

	updates := make(chan scanner.ScannerUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ScannerUpdated
	})
	go func() {
		for evt := range sub.C {
			var update scanner.ScannerUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	evt, err := events.NewEventWithTime(events.SignalGenerated, "signal", map[string]any{
		"symbol":     "NIFTY",
		"timeframe":  "1m",
		"signal":     "SELL",
		"strategy":   "rsi",
		"confidence": 0.3,
		"timestamp":  at,
	}, at)
	require.NoError(t, err)
	bus.Publish(evt)

	select {
	case update := <-updates:
		require.Equal(t, "rsi", update.ScannerName)
		require.Equal(t, scanner.StatusWatch, update.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for scanner.updated")
	}

	require.NoError(t, engine.Close())
	sub.Close()
}

func TestRankingScannerPublishesTopSymbol(t *testing.T) {
	bus := eventbus.New()
	cfg := baseCfg()
	cfg.Symbols = []string{"NIFTY", "BANKNIFTY"}
	cfg.Scanners = scanner.ScannersConfig{Ranking: true}
	engine, err := scanner.New(cfg, bus, clock.NewSystem())
	require.NoError(t, err)

	updates := make(chan scanner.ScannerUpdated, 1)
	sub := bus.Subscribe(8, func(evt events.Event) bool {
		return evt.Type == events.ScannerUpdated
	})
	go func() {
		for evt := range sub.C {
			var update scanner.ScannerUpdated
			if err := json.Unmarshal(evt.Payload, &update); err == nil {
				updates <- update
			}
		}
	}()

	require.NoError(t, engine.Start(context.Background()))

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	evt1, err := events.NewEventWithTime(events.PerformanceUpdated, "performance", map[string]any{
		"symbol":         "NIFTY",
		"timeframe":      "1m",
		"strategy":       "trend_following",
		"total_trades":   10,
		"win_rate":       0.4,
		"realized_pnl":   100,
		"unrealized_pnl": 0,
		"drawdown":       10,
		"timestamp":      at,
	}, at)
	require.NoError(t, err)
	bus.Publish(evt1)

	evt2, err := events.NewEventWithTime(events.PerformanceUpdated, "performance", map[string]any{
		"symbol":         "BANKNIFTY",
		"timeframe":      "1m",
		"strategy":       "trend_following",
		"total_trades":   12,
		"win_rate":       0.8,
		"realized_pnl":   500,
		"unrealized_pnl": 0,
		"drawdown":       5,
		"timestamp":      at,
	}, at)
	require.NoError(t, err)
	bus.Publish(evt2)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case update := <-updates:
			if update.ScannerName == "ranking" && update.Symbol == "BANKNIFTY" {
				require.Equal(t, scanner.StatusMatch, update.Status)
				require.NoError(t, engine.Close())
				sub.Close()
				return
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Fatal("timeout waiting for BANKNIFTY ranking scanner.updated")
}

func TestHealthReporting(t *testing.T) {
	bus := eventbus.New()
	engine, err := scanner.New(baseCfg(), bus, clock.NewSystem())
	require.NoError(t, err)

	require.NoError(t, engine.Start(context.Background()))

	report := engine.Health()
	require.Equal(t, "scanner_engine", report.Component)
	require.Equal(t, health.StatusHealthy, report.Status)
	require.True(t, report.Connected)
	require.Equal(t, "true", report.Details["enabled"])
	require.Equal(t, "5", report.Details["scanner_count"])

	require.NoError(t, engine.Close())
}

func TestEventPublishIncrementsHealthCounters(t *testing.T) {
	bus := eventbus.New()
	engine, err := scanner.New(baseCfg(), bus, clock.NewSystem())
	require.NoError(t, err)
	require.NoError(t, engine.Start(context.Background()))

	at := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	evt, err := events.NewEventWithTime(events.SignalGenerated, "signal", map[string]any{
		"symbol":     "NIFTY",
		"timeframe":  "1m",
		"signal":     "BUY",
		"strategy":   "macd_cross",
		"confidence": 0.9,
		"timestamp":  at,
	}, at)
	require.NoError(t, err)
	bus.Publish(evt)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		report := engine.Health()
		if report.Details["matches_found"] == "1" {
			require.Equal(t, "1", report.Details["events_processed"])
			require.NoError(t, engine.Close())
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected health counters to increment")
}
