package config

import (
	"fmt"

	"github.com/vanam-gangireddy/option-engine/internal/recommendationstate"
)

// RecommendationStateConfig controls the recommendation state manager.
type RecommendationStateConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	SubscriberBuffer int  `mapstructure:"subscriber_buffer"`
	MaxActive        int  `mapstructure:"max_active"`
}

// RecommendationStateEngineSettings maps intelligence recommendation state settings.
func (c *Config) RecommendationStateEngineSettings() RecommendationStateConfig {
	return c.Intelligence.RecommendationState
}

// BuildRecommendationStateEngineConfig maps application config into the recommendation state engine config.
func BuildRecommendationStateEngineConfig(cfg RecommendationStateConfig) (recommendationstate.Config, error) {
	out := recommendationstate.Config{
		Enabled:          cfg.Enabled,
		SubscriberBuffer: cfg.SubscriberBuffer,
		MaxActive:        cfg.MaxActive,
	}.WithDefaults()
	if err := out.Validate(); err != nil {
		return recommendationstate.Config{}, fmt.Errorf("recommendation state config: %w", err)
	}
	return out, nil
}
