package groww

import (
	"fmt"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

const defaultBaseURL = "https://api.groww.in"

// Config holds Groww provider settings.
type Config struct {
	Enabled           bool
	APIKey            string
	APISecret         string
	AccessToken       string
	BaseURL           string
	RequestsPerSecond float64
	RetryAttempts     int
	RetryBackoff      time.Duration
	ReplaySpeed       float64
	InstantReplay     bool
	CandleInterval    string
	Timeframe         market.Timeframe
	Exchange          string
	Segment           string
	StartTime         time.Time
	EndTime           time.Time
	RequestTimeout    time.Duration
}

// ParseConfig maps factory configuration into a typed Groww config.
func ParseConfig(cfg api.FactoryConfig) (Config, error) {
	m := cfg.ProviderCfg
	out := Config{
		Enabled:           getBool(m, "enabled", true),
		APIKey:            getString(m, "api_key"),
		APISecret:         getString(m, "api_secret"),
		AccessToken:       getString(m, "access_token"),
		BaseURL:           getStringDefault(m, "base_url", defaultBaseURL),
		RequestsPerSecond: getFloat(m, "requests_per_second", 5),
		RetryAttempts:     getInt(m, "retry_attempts", 3),
		RetryBackoff:      time.Duration(getInt(m, "retry_backoff_ms", 500)) * time.Millisecond,
		CandleInterval:    resolveCandleInterval(m),
		Timeframe:         market.Timeframe(getStringDefault(m, "timeframe", "5m")),
		Exchange:          getStringDefault(m, "exchange", "NSE"),
		Segment:           getStringDefault(m, "segment", "CASH"),
		RequestTimeout:    api.ParseDuration(getString(m, "request_timeout"), "30s"),
	}

	if out.RequestsPerSecond <= 0 {
		out.RequestsPerSecond = 5
	}
	if out.RetryAttempts < 0 {
		out.RetryAttempts = 0
	}
	if out.RetryBackoff <= 0 {
		out.RetryBackoff = 500 * time.Millisecond
	}

	speedRaw := m["replay_speed"]
	if speedRaw == nil {
		speedRaw = m["speed"]
	}
	speed, instant, err := parseReplaySpeed(speedRaw)
	if err != nil {
		return Config{}, fmt.Errorf("groww replay_speed: %w", err)
	}
	out.ReplaySpeed = speed
	out.InstantReplay = instant

	if start := getString(m, "start_time"); start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return Config{}, fmt.Errorf("groww start_time: %w", err)
		}
		out.StartTime = t
	}
	if end := getString(m, "end_time"); end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return Config{}, fmt.Errorf("groww end_time: %w", err)
		}
		out.EndTime = t
	}

	if !out.Enabled {
		return out, nil
	}
	if out.AccessToken == "" && (out.APIKey == "" || out.APISecret == "") {
		return Config{}, fmt.Errorf("groww requires access_token or api_key and api_secret")
	}
	return out, nil
}

func resolveCandleInterval(m map[string]any) string {
	if raw := getString(m, "candle_interval"); raw != "" {
		return raw
	}
	tf := getStringDefault(m, "timeframe", "5m")
	if mapped, ok := timeframeToGroww[tf]; ok {
		return mapped
	}
	return "5minute"
}

func parseReplaySpeed(raw any) (float64, bool, error) {
	if raw == nil {
		return 1.0, false, nil
	}
	if s, ok := raw.(string); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "", "realtime", "1x":
			return 1.0, false, nil
		case "instant":
			return 0, true, nil
		}
	}
	speed, err := backtest.ParseSpeed(raw)
	if err != nil {
		return 0, false, err
	}
	return speed, false, nil
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func getStringDefault(m map[string]any, key, def string) string {
	if v := getString(m, key); v != "" {
		return v
	}
	return def
}

func getBool(m map[string]any, key string, def bool) bool {
	if m == nil {
		return def
	}
	v, ok := m[key].(bool)
	if !ok {
		return def
	}
	return v
}

func getInt(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return def
	}
}

func getFloat(m map[string]any, key string, def float64) float64 {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return def
	}
}
