package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/recommendation"
)

// RecommendationConfig controls the recommendation engine.
type RecommendationConfig struct {
	Enabled            bool    `mapstructure:"enabled"`
	SubscriberBuffer   int     `mapstructure:"subscriber_buffer"`
	StrongBuyThreshold float64 `mapstructure:"strong_buy_threshold"`
	BuyThreshold       float64 `mapstructure:"buy_threshold"`
	WatchThreshold     float64 `mapstructure:"watch_threshold"`
}

// RecommendationEngineConfig is the validated recommendation configuration used by DI wiring.
type RecommendationEngineConfig struct {
	Enabled            bool
	SubscriberBuffer   int
	StrongBuyThreshold float64
	BuyThreshold       float64
	WatchThreshold     float64
}

// RecommendationEngineSettings maps intelligence recommendation settings.
func (c *Config) RecommendationEngineSettings() RecommendationEngineConfig {
	return RecommendationEngineConfig{
		Enabled:            c.Intelligence.Recommendation.Enabled,
		SubscriberBuffer:   c.Intelligence.Recommendation.SubscriberBuffer,
		StrongBuyThreshold: c.Intelligence.Recommendation.StrongBuyThreshold,
		BuyThreshold:       c.Intelligence.Recommendation.BuyThreshold,
		WatchThreshold:     c.Intelligence.Recommendation.WatchThreshold,
	}
}

// BuildRecommendationEngineConfig maps application config into the recommendation engine config.
func BuildRecommendationEngineConfig(cfg RecommendationEngineConfig) (recommendation.Config, error) {
	out := recommendation.Config{
		Enabled:            cfg.Enabled,
		SubscriberBuffer:   cfg.SubscriberBuffer,
		StrongBuyThreshold: cfg.StrongBuyThreshold,
		BuyThreshold:       cfg.BuyThreshold,
		WatchThreshold:     cfg.WatchThreshold,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return recommendation.Config{}, fmt.Errorf("recommendation config: %w", err)
	}
	return out, nil
}
