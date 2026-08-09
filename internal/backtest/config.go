package backtest

import (
	"fmt"
	"strings"
	"time"

	"github.com/vanam-gangireddy/option-engine/internal/domain/market"
)

const engineName = "backtest_engine"

// Config controls historical candle replay through the market pipeline.
type Config struct {
	Enabled   bool
	Speed     float64
	Symbols   []string
	StartTime time.Time
	EndTime   time.Time
	DataPath  string
	Timeframe market.Timeframe
}

func (c Config) WithDefaults() Config {
	if c.Speed <= 0 {
		c.Speed = 1.0
	}
	if len(c.Symbols) == 0 {
		c.Symbols = []string{"NIFTY"}
	}
	if c.Timeframe == "" {
		c.Timeframe = market.TF1m
	}
	return c
}

// Validate checks replay configuration.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Speed <= 0 {
		return fmt.Errorf("backtest speed must be positive")
	}
	if len(c.Symbols) == 0 {
		return fmt.Errorf("backtest symbols required")
	}
	if !c.EndTime.IsZero() && !c.StartTime.IsZero() && c.EndTime.Before(c.StartTime) {
		return fmt.Errorf("backtest end_time must be after start_time")
	}
	return nil
}

// ParseSpeed converts values like "1x" or 1.0 into a replay multiplier.
func ParseSpeed(raw any) (float64, error) {
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "x"))
		if s == "" {
			return 1.0, nil
		}
		var speed float64
		if _, err := fmt.Sscanf(s, "%f", &speed); err != nil {
			return 0, fmt.Errorf("parse speed %q: %w", raw, err)
		}
		if speed <= 0 {
			return 0, fmt.Errorf("speed must be positive")
		}
		return speed, nil
	case float64:
		if v <= 0 {
			return 0, fmt.Errorf("speed must be positive")
		}
		return v, nil
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("speed must be positive")
		}
		return float64(v), nil
	default:
		return 1.0, nil
	}
}
