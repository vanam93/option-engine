package csv_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/registry"
	"github.com/vanam-gangireddy/option-engine/internal/providers"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
	"github.com/vanam-gangireddy/option-engine/internal/providers/csv"
)

func testCSVContent(extraRows ...string) string {
	content := "date,open,high,low,close,volume\n"
	content += "2015-01-09 09:15:00+05:30,8285.45,8301.3,8285.45,8301.2,0\n"
	content += "2015-01-09 09:20:00+05:30,8300.5,8303.0,8293.25,8301.0,100\n"
	content += "2015-01-09 09:25:00+05:30,8301.65,8302.55,8286.8,8294.15,250\n"
	for _, row := range extraRows {
		content += row + "\n"
	}
	return content
}

func writeTestCSV(t *testing.T, root, symbol, filename, content string) string {
	t.Helper()
	dir := filepath.Join(root, symbol)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func testProviderConfig(root string) map[string]any {
	return map[string]any{
		"enabled":        true,
		"root_directory": root,
		"symbol":         "NIFTY50",
		"exchange":       "NSE",
		"timeframe":      "5m",
		"replay_speed":   "instant",
		"publish_delay":  false,
		"loop":           false,
		"batch_size":     1000,
	}
}

func TestParseRowAndHeader(t *testing.T) {
	require.NoError(t, csv.ParseHeader([]string{"date", "open", "high", "low", "close", "volume"}))
	require.Error(t, csv.ParseHeader([]string{"date", "open"}))

	row, err := csv.ParseRow(2, []string{"2015-01-09 09:15:00+05:30", "8285.45", "8301.3", "8285.45", "8301.2", "0"})
	require.NoError(t, err)
	require.Equal(t, 8301.2, row.Close)

	_, err = csv.ParseRow(3, []string{"bad", "1", "2", "3", "4", "5"})
	require.Error(t, err)

	_, err = csv.ParseRow(4, []string{"2015-01-09 09:15:00+05:30", "x", "2", "3", "4", "5"})
	require.Error(t, err)
}

func TestIteratorSkipsMalformedRows(t *testing.T) {
	root := t.TempDir()
	content := testCSVContent("not,a,valid,row,here,now", "2015-01-09 09:30:00+05:30,8294.1,8295.75,8280.65,8288.5,10")
	writeTestCSV(t, root, "nifty50", "5min.csv", content)

	it, err := csv.OpenDataFile(csv.Config{
		RootDirectory: root,
		Symbol:        "NIFTY50",
		Timeframe:     market.TF5m,
		BatchSize:     1000,
	})
	require.NoError(t, err)
	defer it.Close()

	count := 0
	for {
		_, ok, err := it.Next()
		require.NoError(t, err)
		if !ok {
			break
		}
		count++
	}
	require.Equal(t, 4, count)
	require.Equal(t, int64(1), it.ParseErrors())
}

func TestIteratorConstantMemory(t *testing.T) {
	root := t.TempDir()
	var sb []byte
	sb = append(sb, []byte("date,open,high,low,close,volume\n")...)
	for i := 0; i < 5000; i++ {
		sb = append(sb, []byte("2015-01-09 09:15:00+05:30,100,101,99,100.5,1\n")...)
	}
	writeTestCSV(t, root, "nifty50", "5min.csv", string(sb))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	it, err := csv.OpenDataFile(csv.Config{
		RootDirectory: root,
		Symbol:        "NIFTY50",
		Timeframe:     market.TF5m,
		BatchSize:     1000,
	})
	require.NoError(t, err)
	defer it.Close()

	count := 0
	for {
		_, ok, err := it.Next()
		require.NoError(t, err)
		if !ok {
			break
		}
		count++
	}
	require.Equal(t, 5000, count)

	runtime.GC()
	runtime.ReadMemStats(&after)
	growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	require.Less(t, growth, int64(5*1024*1024), "iterator should not retain large heap growth")
}

func TestReplaySpeeds(t *testing.T) {
	prev := time.Date(2015, 1, 9, 9, 15, 0, 0, time.UTC)
	curr := prev.Add(5 * time.Minute)

	instant := csv.NewStreamer(csv.Config{PublishDelay: true, InstantReplay: true})
	require.Equal(t, time.Duration(0), instant.Delay(prev, curr))

	realtime := csv.NewStreamer(csv.Config{PublishDelay: true, ReplaySpeed: 1.0})
	require.Equal(t, 5*time.Minute, realtime.Delay(prev, curr))

	fast := csv.NewStreamer(csv.Config{PublishDelay: true, ReplaySpeed: 10.0})
	require.Equal(t, 30*time.Second, fast.Delay(prev, curr))

	noDelay := csv.NewStreamer(csv.Config{PublishDelay: false, ReplaySpeed: 1.0})
	require.Equal(t, time.Duration(0), noDelay.Delay(prev, curr))
}

func TestProviderLifecycleAndStreaming(t *testing.T) {
	root := t.TempDir()
	writeTestCSV(t, root, "nifty50", "5min.csv", testCSVContent())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg := registry.New()
	require.NoError(t, reg.Load([]registry.Instrument{{
		Symbol: "NIFTY", Token: "1", Exchange: "NSE", InstrumentType: market.InstrumentIndex, LotSize: 25,
	}}))

	provider, err := csv.NewFromConfig(api.FactoryConfig{
		ProviderCfg: testProviderConfig(root),
		Deps: api.Dependencies{
			Clock:          clock.NewSystem(),
			SymbolRegistry: reg,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "csv", provider.Name())
	require.True(t, provider.Capabilities().HistoricalData)
	require.True(t, provider.Capabilities().Replay)

	require.NoError(t, provider.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	var evt events.Event
	select {
	case evt = <-provider.Events():
	case <-ctx.Done():
		t.Fatal("timed out waiting for streamed candle")
	}
	require.Equal(t, events.MarketDataReceived, evt.Type)
	require.Equal(t, "csv", evt.Source)

	health := provider.Health()
	require.True(t, health.Connected)
	require.NotEmpty(t, health.Details["rows_read"])
	require.NotEmpty(t, health.Details["candles_published"])
	require.Contains(t, health.Details["current_file"], "5min.csv")

	require.NoError(t, provider.Disconnect(ctx))
}

func TestProviderManagerIntegration(t *testing.T) {
	root := t.TempDir()
	writeTestCSV(t, root, "nifty50", "5min.csv", testCSVContent())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg := providers.DefaultRegistry()
	manager := providers.NewManager(reg, providers.ManagerConfig{
		ActiveProvider: "csv",
		ProviderCfg:    testProviderConfig(root),
	})
	require.NoError(t, manager.InitWithDeps(providers.FactoryConfig{
		ProviderCfg: testProviderConfig(root),
		Deps: providers.Dependencies{
			Clock: clock.NewSystem(),
		},
	}))

	provider, err := manager.Provider()
	require.NoError(t, err)
	require.Equal(t, "csv", provider.Name())

	require.NoError(t, manager.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY50"}))

	select {
	case <-provider.Events():
	case <-ctx.Done():
		t.Fatal("timed out waiting for manager-backed csv event")
	}

	require.NoError(t, manager.Disconnect(ctx))
}

func TestSymbolsMatch(t *testing.T) {
	require.True(t, csv.SymbolsMatch("NIFTY", "NIFTY50"))
	require.True(t, csv.SymbolsMatch("NIFTY50", "NIFTY"))
	require.False(t, csv.SymbolsMatch("BANKNIFTY", "NIFTY50"))
}

func TestParseConfigReplaySpeeds(t *testing.T) {
	cfg, err := csv.ParseConfig(api.FactoryConfig{ProviderCfg: map[string]any{
		"enabled": true, "symbol": "NIFTY50", "timeframe": "5m", "replay_speed": "100x",
	}})
	require.NoError(t, err)
	require.Equal(t, 100.0, cfg.ReplaySpeed)
	require.False(t, cfg.InstantReplay)

	cfg, err = csv.ParseConfig(api.FactoryConfig{ProviderCfg: map[string]any{
		"enabled": true, "symbol": "NIFTY50", "timeframe": "5m", "replay_speed": "instant",
	}})
	require.NoError(t, err)
	require.True(t, cfg.InstantReplay)
}

func TestLargeFileIntegration(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "raw", "nifty50", "5min.csv")
	if _, err := os.Stat(path); err != nil {
		t.Skip("large csv fixture not available")
	}

	root := filepath.Join("..", "..", "..", "data", "raw")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := csv.NewFromConfig(api.FactoryConfig{
		ProviderCfg: map[string]any{
			"enabled": true, "root_directory": root, "symbol": "NIFTY50",
			"timeframe": "5m", "replay_speed": "instant", "publish_delay": false,
			"start_time": "2015-01-09T09:15:00+05:30",
			"end_time":   "2015-01-09T10:00:00+05:30",
		},
		Deps: api.Dependencies{Clock: clock.NewSystem()},
	})
	require.NoError(t, err)
	require.NoError(t, provider.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY50"}))

	received := 0
	deadline := time.After(5 * time.Second)
	for received < 10 {
		select {
		case <-provider.Events():
			received++
		case <-deadline:
			t.Fatalf("expected 10 candles, got %d", received)
		}
	}
	require.NoError(t, provider.Disconnect(ctx))
}

func TestProviderInterfaceCompliance(t *testing.T) {
	var _ api.Provider = (*csv.Provider)(nil)
}
