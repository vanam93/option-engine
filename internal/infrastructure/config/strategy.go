package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/analytics/strategy"
)

// BuildStrategyEngineConfig maps application config into the strategy engine config.
func BuildStrategyEngineConfig(cfg StrategyEngineConfig) (strategy.Config, error) {
	out := strategy.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
		MinConfidence:    cfg.MinConfidence,
		TrendFollowing: strategy.TrendFollowingConfig{
			Enabled: cfg.TrendFollowing.Enabled,
		},
		MeanReversion: strategy.MeanReversionConfig{
			Enabled: cfg.MeanReversion.Enabled,
		},
		Breakout: strategy.BreakoutConfig{
			Enabled: cfg.Breakout.Enabled,
		},
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return strategy.Config{}, fmt.Errorf("strategy config: %w", err)
	}
	return out, nil
}
