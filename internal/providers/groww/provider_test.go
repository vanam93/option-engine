package groww_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vanam-gangireddy/option-engine/internal/core/clock"
	"github.com/vanam-gangireddy/option-engine/internal/domain/events"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/market/registry"
	"github.com/vanam-gangireddy/option-engine/internal/providers"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
	"github.com/vanam-gangireddy/option-engine/internal/providers/groww"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/token/api/access":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "SUCCESS",
				"payload": map[string]any{
					"token": "test-token",
				},
			})
		case "/v1/historical/candles":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "SUCCESS",
				"payload": map[string]any{
					"candles": [][]any{
						{"2026-08-01T09:15:00", 100.0, 101.0, 99.0, 100.5, 1000, nil},
						{"2026-08-01T09:20:00", 100.5, 102.0, 100.0, 101.5, 1200, nil},
					},
				},
			})
		case "/v1/historical/expiries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "SUCCESS",
				"payload": map[string]any{
					"expiries": []string{"2026-08-27"},
				},
			})
		case "/v1/historical/contracts":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "SUCCESS",
				"payload": map[string]any{
					"contracts": []string{"NSE-NIFTY-27Aug26-25000-CE"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func testProviderConfig(baseURL string) map[string]any {
	return map[string]any{
		"enabled":              true,
		"api_key":              "key",
		"api_secret":           "secret",
		"base_url":             baseURL,
		"requests_per_second":  100.0,
		"retry_attempts":       2,
		"retry_backoff_ms":     10,
		"replay_speed":         "instant",
		"candle_interval":      "5minute",
		"timeframe":            "5m",
		"exchange":             "NSE",
		"segment":              "CASH",
		"start_time":           "2026-08-01T09:15:00+05:30",
		"end_time":             "2026-08-01T09:30:00+05:30",
	}
}

func TestAuthenticationChecksum(t *testing.T) {
	secret := "secret"
	ts := "1719830400"
	sum := sha256.Sum256([]byte(secret + ts))
	expected := hex.EncodeToString(sum[:])
	got := groww.GenerateChecksumForTest(secret, ts)
	require.Equal(t, expected, got)
}

func TestProviderLifecycleAndStreaming(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg := registry.New()
	require.NoError(t, reg.Load([]registry.Instrument{{
		Symbol: "NIFTY", Token: "1", Exchange: "NSE", InstrumentType: market.InstrumentIndex, LotSize: 25,
	}}))

	provider, err := groww.NewFromConfig(api.FactoryConfig{
		ProviderCfg: testProviderConfig(srv.URL),
		Deps: api.Dependencies{
			Clock:          clock.NewSystem(),
			SymbolRegistry: reg,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "groww", provider.Name())
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
	require.Equal(t, "groww", evt.Source)

	health := provider.Health()
	require.True(t, health.Connected)
	require.Equal(t, "true", health.Details["authenticated"])
	require.NotEmpty(t, health.Details["candles_streamed"])

	require.NoError(t, provider.Disconnect(ctx))
}

func TestHistoricalServiceEndpoints(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	cfg, err := groww.ParseConfig(api.FactoryConfig{ProviderCfg: testProviderConfig(srv.URL)})
	require.NoError(t, err)

	metrics := groww.NewHealthMetricsForTest()
	client := groww.NewClientForTest(cfg, metrics)
	auth := groww.NewAuthenticatorForTest(cfg, client)
	hist := groww.NewHistoricalServiceForTest(cfg, client, auth)

	ctx := context.Background()
	expiries, err := hist.FetchExpiries(ctx, "NIFTY", 2026, 8)
	require.NoError(t, err)
	require.Equal(t, []string{"2026-08-27"}, expiries)

	contracts, err := hist.FetchOptionChain(ctx, "NIFTY", "2026-08-27")
	require.NoError(t, err)
	require.Len(t, contracts, 1)

	candles, err := hist.FetchHistoricalCandles(ctx, groww.CandleRequest{
		Exchange: "NSE", Segment: "CASH", GrowwSymbol: "NSE-NIFTY",
		StartTime: time.Date(2026, 8, 1, 9, 15, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC),
		CandleInterval: "5minute",
	}, "NIFTY")
	require.NoError(t, err)
	require.Len(t, candles, 2)
	require.Equal(t, "NIFTY", candles[0].Symbol)
}

func TestRateLimiter(t *testing.T) {
	limiter := groww.NewRateLimiterForTest(50)
	ctx := context.Background()
	start := time.Now()
	require.NoError(t, limiter.Wait(ctx))
	require.NoError(t, limiter.Wait(ctx))
	require.Less(t, time.Since(start), 200*time.Millisecond)
}

func TestRetryOnRateLimit(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"status":"FAILURE","error":{"code":"GA003","message":"rate limit"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "SUCCESS",
			"payload": map[string]any{
				"expiries": []string{"2026-08-27"},
			},
		})
	}))
	defer srv.Close()

	cfg, err := groww.ParseConfig(api.FactoryConfig{ProviderCfg: testProviderConfig(srv.URL)})
	require.NoError(t, err)
	metrics := groww.NewHealthMetricsForTest()
	client := groww.NewClientForTest(cfg, metrics)
	auth := groww.NewAuthenticatorForTest(cfg, client)
	auth.SetTokenForTest("token")
	hist := groww.NewHistoricalServiceForTest(cfg, client, auth)

	expiries, err := hist.FetchExpiries(context.Background(), "NIFTY", 2026, 8)
	require.NoError(t, err)
	require.Len(t, expiries, 1)
	require.GreaterOrEqual(t, attempts, 2)
	require.GreaterOrEqual(t, metrics.RetriesForTest(), uint64(1))
}

func TestProviderManagerIntegration(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg := providers.DefaultRegistry()
	manager := providers.NewManager(reg, providers.ManagerConfig{
		ActiveProvider: "groww",
		ProviderCfg:    testProviderConfig(srv.URL),
	})
	require.NoError(t, manager.InitWithDeps(providers.FactoryConfig{
		ProviderCfg: testProviderConfig(srv.URL),
		Deps: providers.Dependencies{
			Clock: clock.NewSystem(),
		},
	}))

	provider, err := manager.Provider()
	require.NoError(t, err)
	require.Equal(t, "groww", provider.Name())

	require.NoError(t, manager.Connect(ctx))
	require.NoError(t, provider.Subscribe(ctx, []string{"NIFTY"}))

	select {
	case <-provider.Events():
	case <-ctx.Done():
		t.Fatal("timed out waiting for manager-backed groww event")
	}

	require.NoError(t, manager.Disconnect(ctx))
}

func TestAuthTokenRequestBody(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/token/api/access", r.URL.Path)
		require.Equal(t, "Bearer key", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "SUCCESS",
			"payload": map[string]any{"token": "issued-token"},
		})
	}))
	defer srv.Close()

	cfg, err := groww.ParseConfig(api.FactoryConfig{ProviderCfg: testProviderConfig(srv.URL)})
	require.NoError(t, err)
	client := groww.NewClientForTest(cfg, groww.NewHealthMetricsForTest())
	auth := groww.NewAuthenticatorForTest(cfg, client)
	token, err := auth.Authenticate(context.Background())
	require.NoError(t, err)
	require.Equal(t, "issued-token", token)
	require.Equal(t, "approval", captured["key_type"])
	require.NotEmpty(t, captured["checksum"])
	_, err = strconv.Atoi(captured["timestamp"].(string))
	require.NoError(t, err)
}

func TestMapGrowwCandles(t *testing.T) {
	candles, err := groww.MapGrowwCandlesForTest([][]any{
		{"2026-08-01T09:15:00", 1.0, 2.0, 0.5, 1.5, 100.0, nil},
	}, "NIFTY", market.TF5m)
	require.NoError(t, err)
	require.Len(t, candles, 1)
	require.Equal(t, 1.5, candles[0].Close)
}

func TestProviderInterfaceCompliance(t *testing.T) {
	var _ api.Provider = (*groww.Provider)(nil)
}
