package csv

import (
	"fmt"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/backtest"
	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
	"github.com/vanam-gangireddy/option-engine/internal/providers/api"
)

// Config holds CSV historical provider settings.
type Config struct {
	Enabled       bool
	RootDirectory string
	Symbol        string
	Exchange      string
	Segment       string
	Timeframe     market.Timeframe
	ReplaySpeed   float64
	InstantReplay bool
	PublishDelay  bool
	Loop          bool
	BatchSize     int
	StartTime     time.Time
	EndTime       time.Time
}

// ParseConfig maps factory configuration into a typed CSV config.
func ParseConfig(cfg api.FactoryConfig) (Config, error) {
	m := cfg.ProviderCfg
	out := Config{
		Enabled:       getBool(m, "enabled", true),
		RootDirectory: getStringDefault(m, "root_directory", "data/raw"),
		Symbol:        getStringDefault(m, "symbol", "NIFTY50"),
		Exchange:      getStringDefault(m, "exchange", "NSE"),
		Segment:       getStringDefault(m, "segment", "INDICES"),
		Timeframe:     market.Timeframe(getStringDefault(m, "timeframe", "5m")),
		PublishDelay:  getBool(m, "publish_delay", true),
		Loop:          getBool(m, "loop", false),
		BatchSize:     getInt(m, "batch_size", 1000),
	}

	if out.BatchSize <= 0 {
		out.BatchSize = 1000
	}

	speedRaw := m["replay_speed"]
	if speedRaw == nil {
		speedRaw = m["speed"]
	}
	speed, instant, err := parseReplaySpeed(speedRaw)
	if err != nil {
		return Config{}, fmt.Errorf("csv replay_speed: %w", err)
	}
	out.ReplaySpeed = speed
	out.InstantReplay = instant

	if start := getString(m, "start_time"); start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return Config{}, fmt.Errorf("csv start_time: %w", err)
		}
		out.StartTime = t
	}
	if end := getString(m, "end_time"); end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return Config{}, fmt.Errorf("csv end_time: %w", err)
		}
		out.EndTime = t
	}

	if !out.Enabled {
		return out, nil
	}
	if out.Symbol == "" {
		return Config{}, fmt.Errorf("csv symbol is required")
	}
	if _, ok := timeframeToFilename[out.Timeframe]; !ok {
		return Config{}, fmt.Errorf("csv timeframe %q is not supported", out.Timeframe)
	}
	return out, nil
}

// DataFilePath resolves the CSV file path for the configured symbol and timeframe.
func (c Config) DataFilePath() string {
	dir := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.Symbol), " ", ""))
	filename := timeframeToFilename[c.Timeframe]
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.RootDirectory, "/\\"), dir, filename)
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
