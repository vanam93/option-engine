package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/signal"
)

// BuildSignalEngineConfig maps application config into the signal engine config.
func BuildSignalEngineConfig(cfg SignalEngineConfig) (signal.Config, error) {
	out := signal.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
		EMACross: signal.EMACrossConfig{
			Enabled:    cfg.EMACross.Enabled,
			FastPeriod: cfg.EMACross.FastPeriod,
			SlowPeriod: cfg.EMACross.SlowPeriod,
		},
		MACDCross: signal.MACDCrossConfig{
			Enabled: cfg.MACDCross.Enabled,
		},
		RSI: signal.RSIConfig{
			Enabled:    cfg.RSI.Enabled,
			Oversold:   cfg.RSI.Oversold,
			Overbought: cfg.RSI.Overbought,
		},
		Bollinger: signal.BollingerConfig{
			Enabled: cfg.Bollinger.Enabled,
		},
	}
	if err := out.Validate(); err != nil {
		return signal.Config{}, fmt.Errorf("signal config: %w", err)
	}
	return out, nil
}
