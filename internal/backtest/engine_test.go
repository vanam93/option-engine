package backtest_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

func sampleCandles() []market.Candle {
	base := time.Date(2024, 1, 15, 9, 15, 0, 0, time.UTC)
	return []market.Candle{
		{
			ID: uuid.New(), Symbol: "NIFTY", Timeframe: market.TF1m,
			Open: 100, High: 101, Low: 99, Close: 100.5, Volume: 1000,
			OpenTime: base, CloseTime: base.Add(time.Minute),
		},
		{
			ID: uuid.New(), Symbol: "NIFTY", Timeframe: market.TF1m,
			Open: 100.5, High: 102, Low: 100, Close: 101.5, Volume: 1200,
			OpenTime: base.Add(time.Minute), CloseTime: base.Add(2 * time.Minute),
		},
		{
			ID: uuid.New(), Symbol: "NIFTY", Timeframe: market.TF1m,
			Open: 101.5, High: 103, Low: 101, Close: 102.5, Volume: 900,
			OpenTime: base.Add(2 * time.Minute), CloseTime: base.Add(3 * time.Minute),
		},
	}
}

func TestLoadHistoricalCandlesReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dir := t.TempDir()
	path := filepath.Join(dir, "candles.json")
	data, err := json.Marshal(sampleCandles())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	cfg := backtest.Config{
		Enabled:   true,
		Speed:     1000,
		Symbols:   []string{"NIFTY"},
		DataPath:  path,
		Timeframe: market.TF1m,
	}
	engine, err := backtest.New(cfg, clock.NewReplay(sampleCandles()[0].CloseTime))
	require.NoError(t, err)
	defer func() { _ = engine.Close() }()

	provider := engine.Provider()
	require.NoError(t, provider.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	received := make([]events.Event, 0, 3)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 3; i++ {
			select {
			case evt := <-provider.Events():
				received = append(received, evt)
			case <-ctx.Done():
				return
			}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for replay events")
	}

	require.Len(t, received, 3)
	require.Equal(t, events.MarketDataReceived, received[0].Type)
	require.Equal(t, uint64(3), engine.ProcessedCandles())
	require.Equal(t, backtest.ReplayStatusCompleted, engine.Status())

	report := engine.Health()
	require.Equal(t, "backtest_engine", report.Component)
	require.Equal(t, "3", report.Details["candles_replayed"])
	require.Equal(t, "1", report.Details["symbols_loaded"])
}

func TestReplayOrdering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := backtest.Config{
		Enabled:   true,
		Speed:     1000,
		Symbols:   []string{"NIFTY"},
		Timeframe: market.TF1m,
	}
	engine, err := backtest.NewWithCandles(cfg, sampleCandles(), clock.NewReplay(sampleCandles()[0].CloseTime))
	require.NoError(t, err)
	defer func() { _ = engine.Close() }()

	provider := engine.Provider()
	require.NoError(t, provider.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	var prev time.Time
	for i := 0; i < 3; i++ {
		select {
		case evt := <-provider.Events():
			require.True(t, evt.Timestamp.After(prev) || prev.IsZero())
			prev = evt.Timestamp
		case <-ctx.Done():
			t.Fatal("timed out waiting for ordered replay event")
		}
	}
}

func TestShutdownCleanStop(t *testing.T) {
	ctx := context.Background()

	cfg := backtest.Config{
		Enabled:   true,
		Speed:     1,
		Symbols:   []string{"NIFTY"},
		Timeframe: market.TF1m,
	}
	engine, err := backtest.NewWithCandles(cfg, sampleCandles(), clock.NewReplay(sampleCandles()[0].CloseTime))
	require.NoError(t, err)

	provider := engine.Provider()
	require.NoError(t, provider.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	disconnectDone := make(chan struct{})
	go func() {
		_ = provider.Disconnect(ctx)
		close(disconnectDone)
	}()

	select {
	case <-disconnectDone:
	case <-time.After(2 * time.Second):
		t.Fatal("disconnect did not complete promptly")
	}

	require.NoError(t, engine.Close())
	report := engine.Health()
	require.Equal(t, backtest.ReplayStatusStopped, backtest.ReplayStatus(report.Details["status"]))
}
