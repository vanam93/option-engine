package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/indicator"
)

// BuildIndicatorEngineConfig maps application config into the indicator engine config.
func BuildIndicatorEngineConfig(cfg IndicatorEngineConfig) (indicator.Config, error) {
	out := indicator.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
	}
	for _, p := range cfg.EMA {
		if p.Period < 1 {
			return indicator.Config{}, fmt.Errorf("indicator ema period must be >= 1")
		}
		out.EMA = append(out.EMA, indicator.PeriodConfig{Period: p.Period})
	}
	for _, p := range cfg.SMA {
		if p.Period < 1 {
			return indicator.Config{}, fmt.Errorf("indicator sma period must be >= 1")
		}
		out.SMA = append(out.SMA, indicator.PeriodConfig{Period: p.Period})
	}
	return out, nil
}
